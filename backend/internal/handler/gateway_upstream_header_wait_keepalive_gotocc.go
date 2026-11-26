package handler

import (
	"errors"
	"context"
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

// streamErrorProtocol 选择心跳提交 200 之后错误改写成的流内终止事件，与各入口在流开始后已有的错误事件一致。
type streamErrorProtocol int

const (
	// streamErrorAnthropic：event: error 加 Anthropic 错误对象，官方 SDK 按事件名分发。
	streamErrorAnthropic streamErrorProtocol = iota
	// streamErrorResponses：response.failed，Codex 等严格客户端要求的终止事件。
	streamErrorResponses
	// streamErrorChat：event: error 加 OpenAI 错误对象。
	streamErrorChat
)

// upstreamHeaderWaitKeepalive 在流式请求等待上游响应头期间向下游发送 SSE 心跳。
//
// 网关拿到上游响应头才向客户端写 200 与 SSE。上游首字节慢时，客户端与中间代理
// （Cloudflare 约 100–125 秒）长时间收不到字节就会切断连接。有上游请求在等响应头，且距上一次
// 向客户端写出字节满 gateway.stream_keepalive_interval 时，提交 200 与 SSE 响应头并发出入站协议
// 已有的心跳帧，此后每个间隔一次。计时跨重试与换号连续计算，不因每次尝试重新开始。
//
// 心跳对请求侧不可见：只发过心跳时 Written() 为 false、Size() 不计心跳字节，请求侧的 Header()
// 拿到响应头副本，WriteHeader 只记下想回的状态码。网关原有的"是否已向客户端写出"判定（换号、
// 回退、兜底错误）因此保持没有保活时的行为。请求侧此时写出的错误响应（状态码 >= 400 的非 SSE
// 正文，或只有状态码）改写成入站协议的流内终止事件，按原状态码记入运维日志；改写之后请求侧
// 再写的内容丢弃。上游响应头（如 x-request-id、x-codex-turn-state）在提交之后无法再发给客户端。
//
// 后台 goroutine 也可能沿用请求 context 访问上游，c.Writer 因此换成 upstreamHeaderWaitWriter，
// 请求侧写出与心跳串行。可能被下层写出链路回调的方法（Header、Written、Size、Status）只取 mu，
// 状态读自本结构的镜像，不读下层 writer；mu 持有期间从不等待 writeMu。
type upstreamHeaderWaitKeepalive struct {
	mu sync.Mutex // 保护下列状态
	// writeMu 让心跳与请求侧写出互斥，持有期间调用下层 writer。
	writeMu  sync.Mutex
	c        *gin.Context
	writer   gin.ResponseWriter // 安装前的 writer
	frame    string
	protocol streamErrorProtocol
	interval time.Duration

	waits   int           // 进行中的上游等待
	window  chan struct{} // 当前心跳窗口，nil 为关闭
	stopped bool          // 请求结束
	dead    bool          // 心跳写失败，不再保活

	lastByte    time.Time // 最近一次向下游写出字节的时间，心跳按它计时
	initialSize int       // 安装时下层已写字节数，未写出为 -1
	size        int       // 下层已写字节数的镜像，未写出为 -1
	status      int       // 下层状态码的镜像

	ours         bool        // 由心跳提交了 200 与 SSE 响应头
	header       http.Header // 心跳提交后交给请求侧的响应头副本
	beatBytes    int         // 心跳字节
	requestBytes int         // 请求侧经本 writer 写出的字节
	wantStatus   int         // 心跳提交后请求侧想回的状态码
	errorSent    bool        // 已改写出流内终止事件
}

// installUpstreamHeaderWaitKeepalive 为客户端要求流式的请求挂上等待期心跳，返回请求结束时调用的停止函数。
// intervalSeconds 即 gateway.stream_keepalive_interval，0 表示关闭。pingFormat 取入站协议已有的心跳：
// Anthropic 网关的 Messages 用该入口排队心跳的帧，其余入口用 SSE 注释帧（data 帧会被官方 SDK 当作正文）。
func installUpstreamHeaderWaitKeepalive(c *gin.Context, stream bool, intervalSeconds int, pingFormat SSEPingFormat, protocol streamErrorProtocol) func() {
	if !stream || intervalSeconds <= 0 {
		return func() {}
	}
	original := c.Writer
	k := &upstreamHeaderWaitKeepalive{
		c:           c,
		writer:      original,
		frame:       string(pingFormat),
		protocol:    protocol,
		interval:    time.Duration(intervalSeconds) * time.Second,
		lastByte:    time.Now(),
		initialSize: original.Size(),
		size:        original.Size(),
		status:      original.Status(),
	}
	wrapped := &upstreamHeaderWaitWriter{ResponseWriter: original, k: k}
	c.Writer = wrapped
	c.Request = c.Request.WithContext(service.WithUpstreamHeaderWaitObserver(c.Request.Context(), k))
	return func() {
		k.stop()
		if c.Writer == wrapped {
			c.Writer = original
		}
	}
}

// BeginUpstreamHeaderWait 实现 service.UpstreamHeaderWaitObserver。请求侧尚未写出语义正文时计入等待，
// 第一个等待打开心跳窗口。每次上游尝试重新记录请求侧想回的状态码。
func (k *upstreamHeaderWaitKeepalive) BeginUpstreamHeaderWait() func() {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.stopped || k.dead || k.errorSent || !k.onlyNonSemanticLocked() {
		return func() {}
	}
	k.wantStatus = 0
	k.waits++
	if k.window == nil {
		k.window = make(chan struct{})
		go k.run(k.window)
	}
	return k.endWait
}

// onlyNonSemanticLocked 报告请求侧写出的只有非语义字节：排队心跳，以及上游响应头到达后的空闲保活
// （前一次尝试首字等待期间写出、随后安全换号的情况）。
func (k *upstreamHeaderWaitKeepalive) onlyNonSemanticLocked() bool {
	value, _ := k.c.Get(gatewayStreamHeartbeatBytesKey)
	queueBytes, _ := value.(int)
	return max(k.initialSize, 0)+k.requestBytes == queueBytes+service.OpenAIStreamKeepaliveBytes(k.c)
}

// hiddenLocked 报告心跳提交了响应而请求侧还没写出任何字节：此时心跳对请求侧不可见。
func (k *upstreamHeaderWaitKeepalive) hiddenLocked() bool {
	return k.ours && k.requestBytes == 0 && !k.stopped
}

// endWait 在上游响应头到达或请求失败时调用；最后一个等待结束即关闭窗口，
// 并等进行中的心跳写完，返回后请求侧可直接使用响应。
func (k *upstreamHeaderWaitKeepalive) endWait() {
	k.mu.Lock()
	k.waits--
	if k.waits == 0 {
		k.closeWindowLocked()
	}
	k.mu.Unlock()
	k.writeMu.Lock()
	k.writeMu.Unlock() //nolint:staticcheck // 空临界区：等进行中的心跳写完
}

// stop 在请求结束时调用。请求侧只设了错误状态码而没有写正文时，补写对应的流内终止事件；
// 之后不再有心跳，也不再访问 gin.Context。
func (k *upstreamHeaderWaitKeepalive) isClientClosedLocked() bool {
	if k.wantStatus == statusClientClosedRequest {
		return true
	}
	if k.c != nil && k.c.Request != nil && errors.Is(k.c.Request.Context().Err(), context.Canceled) {
		return true
	}
	return false
}

func (k *upstreamHeaderWaitKeepalive) stop() {
	k.mu.Lock()
	pending := k.hiddenLocked() && k.wantStatus >= http.StatusBadRequest && !k.isClientClosedLocked()
	status := k.wantStatus
	k.stopped = true
	k.closeWindowLocked()
	k.mu.Unlock()
	k.writeMu.Lock()
	defer k.writeMu.Unlock()
	if pending {
		_ = k.writeStreamError(status, nil)
	}
}

func (k *upstreamHeaderWaitKeepalive) closeWindowLocked() {
	if k.window != nil {
		close(k.window)
		k.window = nil
	}
}

// run 在窗口打开期间，按距上一次向下游写出字节满一个间隔发心跳；窗口关闭或下游写失败即退出。
func (k *upstreamHeaderWaitKeepalive) run(window chan struct{}) {
	timer := time.NewTimer(k.untilNextBeat())
	defer timer.Stop()
	for {
		select {
		case <-window:
			return
		case <-timer.C:
		}
		if wait := k.untilNextBeat(); wait > 0 {
			timer.Reset(wait)
			continue
		}
		if !k.beat(window) {
			return
		}
		timer.Reset(k.interval)
	}
}

func (k *upstreamHeaderWaitKeepalive) untilNextBeat() time.Duration {
	k.mu.Lock()
	defer k.mu.Unlock()
	return time.Until(k.lastByte.Add(k.interval))
}

// beat 写出一个心跳帧。下层尚未写出时先设 SSE 响应头，状态随首次写入提交为 200，并把响应头副本
// 交给请求侧，此后请求侧改响应头不再触及下层的响应头。下游写失败时停止保活，断开仍由请求侧
// 后续的写入与 context 取消归因（LC-017）。
func (k *upstreamHeaderWaitKeepalive) beat(window chan struct{}) bool {
	k.writeMu.Lock()
	defer k.writeMu.Unlock()
	k.mu.Lock()
	if k.window != window {
		k.mu.Unlock()
		return false
	}
	if k.size < 0 {
		header := k.writer.Header()
		header.Set("Content-Type", "text/event-stream")
		header.Set("Cache-Control", "no-cache")
		header.Set("Connection", "keep-alive")
		header.Set("X-Accel-Buffering", "no")
		k.header = header.Clone()
		k.ours = true
	}
	k.mu.Unlock()

	written, err := k.writer.WriteString(k.frame)
	if err == nil {
		k.writer.Flush()
	}

	k.mu.Lock()
	defer k.mu.Unlock()
	k.recordWriteLocked(written)
	k.beatBytes += written
	if err != nil {
		k.dead = true
		k.closeWindowLocked()
	}
	return err == nil
}

// recordWriteLocked 更新下层写出的镜像与最近写出时间。
func (k *upstreamHeaderWaitKeepalive) recordWriteLocked(written int) {
	if k.size < 0 {
		k.size = 0
	}
	k.size += written
	k.lastByte = time.Now()
}

type requestWriteAction int

const (
	requestWritePass requestWriteAction = iota
	requestWriteConvert
	requestWriteDrop
)

// requestWrite 关闭心跳窗口并决定请求侧这次写出的处理方式：已改写过终止事件则丢弃；心跳提交后
// 请求侧要回错误状态时改写（正文已是 SSE 帧的除外，由调用方判断）。调用方持有 writeMu。
func (k *upstreamHeaderWaitKeepalive) requestWrite() (requestWriteAction, int) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.closeWindowLocked()
	switch {
	case k.errorSent:
		return requestWriteDrop, 0
	case k.hiddenLocked() && k.isClientClosedLocked():
		return requestWriteDrop, 0
	case k.hiddenLocked() && k.wantStatus >= http.StatusBadRequest:
		return requestWriteConvert, k.wantStatus
	default:
		return requestWritePass, 0
	}
}

func (k *upstreamHeaderWaitKeepalive) recordRequestWrite(written int) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.recordWriteLocked(written)
	k.requestBytes += written
}

// writeStreamError 把请求侧的错误响应改写成入站协议的流内终止事件写出，并按请求侧想回的
// 状态码记入运维日志；调用方持有 writeMu。
func (k *upstreamHeaderWaitKeepalive) writeStreamError(status int, body []byte) error {
	if !hasOpsUpstreamErrorContext(k.c) {
		service.MarkLocalOpenAIError(k.c)
	}
	errType, code, message := parseStreamErrorBody(status, body)
	written, err := k.writer.WriteString(k.streamErrorFrame(errType, code, message))
	if err == nil {
		k.writer.Flush()
	}
	k.mu.Lock()
	k.recordWriteLocked(written)
	k.requestBytes += written
	k.errorSent = true
	k.mu.Unlock()
	service.MarkOpsStreamFailure(k.c, errType, code, message, status)
	return err
}

// streamErrorFrame 生成与各入口流开始后已有错误事件相同的终止帧。
func (k *upstreamHeaderWaitKeepalive) streamErrorFrame(errType, code, message string) string {
	switch k.protocol {
	case streamErrorResponses:
		if code == "" {
			service.SetOpenAIErrorMessageGeneratedCode(k.c, mapResponsesErrorCode(errType, code))
		}
		payload, _ := json.Marshal(responsesFailedEvent{
			Type: "response.failed",
			Response: responsesFailedBody{
				ID:        synthesizeResponseID(k.c),
				Object:    "response",
				CreatedAt: time.Now().Unix(),
				Model:     requestModel(k.c),
				Status:    "failed",
				Output:    []any{},
				Error:     responsesFailedError{Code: mapResponsesErrorCode(errType, code), Message: message},
			},
		})
		return "event: response.failed\ndata: " + string(payload) + "\n\n"
	case streamErrorChat:
		errorObject := gin.H{"type": errType, "message": message}
		if code != "" {
			errorObject["code"] = code
		}
		payload, _ := json.Marshal(gin.H{"error": errorObject})
		return "event: error\ndata: " + string(payload) + "\n\n"
	default:
		payload, _ := json.Marshal(gin.H{"type": "error", "error": gin.H{"type": errType, "message": message}})
		return "event: error\ndata: " + string(payload) + "\n\n"
	}
}

// parseStreamErrorBody 从请求侧写出的错误正文取类型、代码与消息，兼容 Anthropic、OpenAI、Gemini 的错误对象与纯文本。
func parseStreamErrorBody(status int, body []byte) (errType, code, message string) {
	if gjson.ValidBytes(body) {
		errorObject := gjson.GetBytes(body, "error")
		if errorObject.Type == gjson.String {
			message = errorObject.String()
		} else {
			errType = errorObject.Get("type").String()
			if value := errorObject.Get("code"); value.Type == gjson.String {
				code = value.String()
			}
			message = errorObject.Get("message").String()
		}
		if message == "" {
			message = gjson.GetBytes(body, "message").String()
		}
	} else {
		message = strings.TrimSpace(string(body))
	}
	if errType == "" {
		errType = streamErrorTypeForStatus(status)
	}
	if message == "" {
		message = http.StatusText(status)
	}
	return errType, code, message
}

func streamErrorTypeForStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "invalid_request_error"
	case http.StatusUnauthorized:
		return "authentication_error"
	case http.StatusForbidden:
		return "permission_error"
	case http.StatusNotFound:
		return "not_found_error"
	case http.StatusRequestEntityTooLarge:
		return "request_too_large"
	case http.StatusTooManyRequests:
		return "rate_limit_error"
	case 529:
		return "overloaded_error"
	default:
		return "api_error"
	}
}

var sseFieldPrefixes = [][]byte{[]byte("event:"), []byte("data:"), []byte(":"), []byte("id:"), []byte("retry:")}

// isSSEFrame 报告请求侧写出的内容已是 SSE 帧（流内错误事件或心跳），无需改写。
func isSSEFrame(data []byte) bool {
	trimmed := bytes.TrimLeft(data, " \t\r\n")
	for _, prefix := range sseFieldPrefixes {
		if bytes.HasPrefix(trimmed, prefix) {
			return true
		}
	}
	return false
}

// upstreamHeaderWaitWriter 是请求侧看到的 c.Writer，见 upstreamHeaderWaitKeepalive。
type upstreamHeaderWaitWriter struct {
	gin.ResponseWriter
	k *upstreamHeaderWaitKeepalive
}

// Header 在心跳提交之前返回下层响应头并关闭心跳窗口（请求侧开始构造自己的响应）；
// 提交之后返回副本。
func (w *upstreamHeaderWaitWriter) Header() http.Header {
	k := w.k
	k.mu.Lock()
	if k.hiddenLocked() && k.header != nil {
		h := k.header
		k.mu.Unlock()
		return h
	}
	k.closeWindowLocked()
	k.mu.Unlock()
	k.writeMu.Lock()
	k.writeMu.Unlock() //nolint:staticcheck // 等待进行中的心跳完成响应头读写与克隆
	k.mu.Lock()
	if k.hiddenLocked() && k.header != nil {
		h := k.header
		k.mu.Unlock()
		return h
	}
	k.mu.Unlock()
	return w.ResponseWriter.Header()
}

func (w *upstreamHeaderWaitWriter) Written() bool {
	k := w.k
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.hiddenLocked() {
		return false
	}
	return k.size >= 0
}

// Size 不计心跳字节，只发过心跳时为 -1（gin 的未写出值），请求结束后报告实际写出的字节。
func (w *upstreamHeaderWaitWriter) Size() int {
	k := w.k
	k.mu.Lock()
	defer k.mu.Unlock()
	switch {
	case k.hiddenLocked():
		return -1
	case k.stopped:
		return k.size
	default:
		return k.size - k.beatBytes
	}
}

func (w *upstreamHeaderWaitWriter) Status() int {
	k := w.k
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.hiddenLocked() && k.wantStatus != 0 {
		return k.wantStatus
	}
	return k.status
}

func (w *upstreamHeaderWaitWriter) WriteHeader(code int) {
	k := w.k
	k.mu.Lock()
	k.closeWindowLocked()
	if k.hiddenLocked() {
		k.wantStatus = code
		k.mu.Unlock()
		return
	}
	k.mu.Unlock()
	k.writeMu.Lock()
	defer k.writeMu.Unlock()
	w.ResponseWriter.WriteHeader(code)
	k.mu.Lock()
	if code > 0 && k.size < 0 {
		k.status = code
	}
	k.mu.Unlock()
}

func (w *upstreamHeaderWaitWriter) WriteHeaderNow() {
	k := w.k
	k.mu.Lock()
	k.closeWindowLocked()
	hidden := k.hiddenLocked()
	k.mu.Unlock()
	if hidden {
		return
	}
	k.writeMu.Lock()
	defer k.writeMu.Unlock()
	w.ResponseWriter.WriteHeaderNow()
	k.mu.Lock()
	k.recordWriteLocked(0)
	k.mu.Unlock()
}

func (w *upstreamHeaderWaitWriter) Flush() {
	k := w.k
	k.mu.Lock()
	k.closeWindowLocked()
	k.mu.Unlock()
	k.writeMu.Lock()
	defer k.writeMu.Unlock()
	w.ResponseWriter.Flush()
	k.mu.Lock()
	k.recordWriteLocked(0)
	k.mu.Unlock()
}

func (w *upstreamHeaderWaitWriter) Write(data []byte) (int, error) {
	k := w.k
	k.writeMu.Lock()
	defer k.writeMu.Unlock()
	action, status := k.requestWrite()
	switch {
	case action == requestWriteDrop:
		return len(data), nil
	case action == requestWriteConvert && !isSSEFrame(data):
		return len(data), k.writeStreamError(status, data)
	}
	written, err := w.ResponseWriter.Write(data)
	k.recordRequestWrite(written)
	return written, err
}

func (w *upstreamHeaderWaitWriter) WriteString(s string) (int, error) {
	k := w.k
	k.writeMu.Lock()
	defer k.writeMu.Unlock()
	action, status := k.requestWrite()
	switch {
	case action == requestWriteDrop:
		return len(s), nil
	case action == requestWriteConvert && !isSSEFrame([]byte(s)):
		return len(s), k.writeStreamError(status, []byte(s))
	}
	written, err := w.ResponseWriter.WriteString(s)
	k.recordRequestWrite(written)
	return written, err
}
