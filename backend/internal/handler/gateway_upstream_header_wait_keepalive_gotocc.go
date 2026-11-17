package handler

import (
	"net/http"
	"sync"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
)

// upstreamHeaderWaitKeepalive 在流式请求等待上游响应头期间向下游发送 SSE 心跳。
//
// 网关拿到上游响应头才向客户端写 200 与 SSE。上游首字节慢时，客户端与中间代理
// （Cloudflare 约 100–125 秒）长时间收不到字节就会切断连接。上游请求发出后等待满
// gateway.stream_keepalive_interval 仍无响应头，就提交 200 与 SSE 响应头并发出入站协议
// 已有的心跳帧，此后每个间隔一次，直到响应头到达或请求结束。
//
// 心跳与排队心跳同属非语义字节：计入 gatewayStreamHeartbeatBytesKey 与
// service.RecordStreamKeepaliveBytes，只发过心跳的响应在换号判定中仍按"正文未开始"处理；
// HTTP 已提交后的最终失败经 streamStarted 走排队心跳之后已有的流内错误事件。
//
// 后台 goroutine 也可能沿用请求 context 访问上游，c.Writer 因此换成
// upstreamHeaderWaitWriter：请求侧构造响应前先关闭心跳窗口，写出与心跳串行。
type upstreamHeaderWaitKeepalive struct {
	mu sync.Mutex // 保护下列窗口状态
	// writeMu 让心跳写出与请求侧写出互斥；请求侧的 Header() 不取它，
	// 心跳写出链路（如 Server-Timing 首写）回调 c.Writer.Header() 时不会自锁。
	writeMu  sync.Mutex
	c        *gin.Context
	writer   gin.ResponseWriter // 安装前的 writer，心跳直接写入
	frame    string
	interval time.Duration
	waits    int           // 进行中的上游等待
	window   chan struct{} // 当前心跳窗口，nil 为关闭
	beating  bool          // 心跳正在写出；期间的 Header() 来自心跳自身的写出链路
	stopped  bool
}

// installUpstreamHeaderWaitKeepalive 为客户端要求流式的请求挂上等待期心跳，返回请求结束时调用的停止函数。
// intervalSeconds 即 gateway.stream_keepalive_interval，0 表示关闭。pingFormat 取入站协议已有的心跳：
// Anthropic Messages 用该入口排队心跳的帧，OpenAI Responses/Chat 用 SSE 注释帧（data 帧会被官方 SDK 当作正文）。
func installUpstreamHeaderWaitKeepalive(c *gin.Context, stream bool, intervalSeconds int, pingFormat SSEPingFormat) func() {
	if !stream || intervalSeconds <= 0 {
		return func() {}
	}
	original := c.Writer
	k := &upstreamHeaderWaitKeepalive{
		c:        c,
		writer:   original,
		frame:    string(pingFormat),
		interval: time.Duration(intervalSeconds) * time.Second,
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

// BeginUpstreamHeaderWait 实现 service.UpstreamHeaderWaitObserver。下游尚未写出正文
// （未写出或只写过心跳）时计入等待，第一个等待打开心跳窗口。
func (k *upstreamHeaderWaitKeepalive) BeginUpstreamHeaderWait() func() {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.stopped || !k.onlyHeartbeatsLocked() {
		return func() {}
	}
	k.waits++
	if k.window == nil {
		k.window = make(chan struct{})
		go k.run(k.window)
	}
	return k.endWait
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
	k.writeMu.Unlock()
}

// stop 在请求结束时调用；之后不再有心跳，也不再访问 gin.Context。
func (k *upstreamHeaderWaitKeepalive) stop() {
	k.mu.Lock()
	k.stopped = true
	k.closeWindowLocked()
	k.mu.Unlock()
	k.writeMu.Lock()
	k.writeMu.Unlock()
}

func (k *upstreamHeaderWaitKeepalive) closeWindow() {
	k.mu.Lock()
	k.closeWindowLocked()
	k.mu.Unlock()
}

func (k *upstreamHeaderWaitKeepalive) closeWindowLocked() {
	if k.window != nil {
		close(k.window)
		k.window = nil
	}
}

// onlyHeartbeatsLocked 报告下游尚未写出语义正文：没有写出任何字节，或写出的全是心跳。
func (k *upstreamHeaderWaitKeepalive) onlyHeartbeatsLocked() bool {
	if !k.writer.Written() {
		return true
	}
	value, _ := k.c.Get(gatewayStreamHeartbeatBytesKey)
	heartbeatBytes, _ := value.(int)
	return heartbeatBytes > 0 && k.writer.Size() == heartbeatBytes
}

// run 在窗口打开期间每隔一个间隔发一次心跳，窗口关闭或下游写失败即退出。
func (k *upstreamHeaderWaitKeepalive) run(window chan struct{}) {
	ticker := time.NewTicker(k.interval)
	defer ticker.Stop()
	for {
		select {
		case <-window:
			return
		case <-ticker.C:
			if !k.beat(window) {
				return
			}
		}
	}
}

// beat 写出一个心跳帧。首个心跳与排队心跳一样先设 SSE 响应头，状态随首次写入提交为 200。
// 下游写失败时停止保活，断开仍由请求侧后续的写入与 context 取消归因（LC-017）。
func (k *upstreamHeaderWaitKeepalive) beat(window chan struct{}) bool {
	k.writeMu.Lock()
	defer k.writeMu.Unlock()
	k.mu.Lock()
	if k.window != window {
		k.mu.Unlock()
		return false
	}
	k.beating = true
	k.mu.Unlock()

	if !k.writer.Written() {
		header := k.writer.Header()
		header.Set("Content-Type", "text/event-stream")
		header.Set("Cache-Control", "no-cache")
		header.Set("Connection", "keep-alive")
		header.Set("X-Accel-Buffering", "no")
	}
	written, err := k.writer.WriteString(k.frame)
	recordGatewayStreamHeartbeat(k.c, written)
	service.RecordStreamKeepaliveBytes(k.c, written)
	if err == nil {
		k.writer.Flush()
	}

	k.mu.Lock()
	k.beating = false
	if err != nil {
		k.stopped = true
		k.closeWindowLocked()
	}
	k.mu.Unlock()
	return err == nil
}

// upstreamHeaderWaitWriter 让请求侧构造响应前先关闭心跳窗口，写出与心跳串行；读方法原样透传。
type upstreamHeaderWaitWriter struct {
	gin.ResponseWriter
	k *upstreamHeaderWaitKeepalive
}

func (w *upstreamHeaderWaitWriter) Header() http.Header {
	w.k.mu.Lock()
	if !w.k.beating {
		w.k.closeWindowLocked()
	}
	w.k.mu.Unlock()
	return w.ResponseWriter.Header()
}

func (w *upstreamHeaderWaitWriter) Write(data []byte) (int, error) {
	w.k.closeWindow()
	w.k.writeMu.Lock()
	defer w.k.writeMu.Unlock()
	return w.ResponseWriter.Write(data)
}

func (w *upstreamHeaderWaitWriter) WriteString(s string) (int, error) {
	w.k.closeWindow()
	w.k.writeMu.Lock()
	defer w.k.writeMu.Unlock()
	return w.ResponseWriter.WriteString(s)
}

func (w *upstreamHeaderWaitWriter) WriteHeader(code int) {
	w.k.closeWindow()
	w.k.writeMu.Lock()
	defer w.k.writeMu.Unlock()
	w.ResponseWriter.WriteHeader(code)
}

func (w *upstreamHeaderWaitWriter) WriteHeaderNow() {
	w.k.closeWindow()
	w.k.writeMu.Lock()
	defer w.k.writeMu.Unlock()
	w.ResponseWriter.WriteHeaderNow()
}

func (w *upstreamHeaderWaitWriter) Flush() {
	w.k.closeWindow()
	w.k.writeMu.Lock()
	defer w.k.writeMu.Unlock()
	w.ResponseWriter.Flush()
}
