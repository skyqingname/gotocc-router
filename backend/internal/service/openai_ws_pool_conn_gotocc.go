package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type openAIWSDialError struct {
	StatusCode      int
	ResponseHeaders http.Header
	ResponseBody    []byte
	Err             error
}

func (e *openAIWSDialError) Error() string {
	if e == nil {
		return ""
	}
	if e.StatusCode > 0 {
		return fmt.Sprintf("openai ws dial failed: status=%d err=%v", e.StatusCode, e.Err)
	}
	return fmt.Sprintf("openai ws dial failed: %v", e.Err)
}

func (e *openAIWSDialError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type openAIWSAcquireRequest struct {
	Account *Account
	WSURL   string
	Headers http.Header
	// HeadersFactory is evaluated inside dialConn. It exists so credentials
	// whose authorization is per-dial (Agent Identity) are never cached in
	// lastAcquire.
	HeadersFactory  func(context.Context, http.Header) (http.Header, error)
	ProxyURL        string
	PreferredConnID string
	// ForceNewConn: 强制本次获取新连接（避免复用导致连接内续链状态互相污染）。
	ForceNewConn bool
	// ForcePreferredConn: 强制本次只使用 PreferredConnID，禁止漂移到其它连接。
	ForcePreferredConn bool
}

type openAIWSHandshakeCompatibilityKey struct {
	betaFeatures        string
	sessionIdentity     string
	userAgent           string
	originator          string
	version             string
	codexInstallationID string
	threadID            string
	clientRequestID     string
	codexWindowID       string
}

type openAIWSConnLease struct {
	pool       *openAIWSConnPool
	accountID  int64
	conn       *openAIWSConn
	queueWait  time.Duration
	connPick   time.Duration
	idleBefore time.Duration
	ageBefore  time.Duration
	reused     bool
	released   atomic.Bool
}

func (l *openAIWSConnLease) activeConn() (*openAIWSConn, error) {
	if l == nil || l.conn == nil {
		return nil, errOpenAIWSConnClosed
	}
	if l.released.Load() {
		return nil, errOpenAIWSConnClosed
	}
	return l.conn, nil
}

func (l *openAIWSConnLease) ConnID() string {
	if l == nil || l.conn == nil {
		return ""
	}
	return l.conn.id
}

func (l *openAIWSConnLease) QueueWaitDuration() time.Duration {
	if l == nil {
		return 0
	}
	return l.queueWait
}

func (l *openAIWSConnLease) ConnPickDuration() time.Duration {
	if l == nil {
		return 0
	}
	return l.connPick
}

func (l *openAIWSConnLease) Reused() bool {
	if l == nil {
		return false
	}
	return l.reused
}

// IdleBefore 返回借出时该连接已空闲的时长。
func (l *openAIWSConnLease) IdleBefore() time.Duration {
	if l == nil {
		return 0
	}
	return l.idleBefore
}

// AgeBefore 返回借出时该连接自建立起的时长。
func (l *openAIWSConnLease) AgeBefore() time.Duration {
	if l == nil {
		return 0
	}
	return l.ageBefore
}

func (l *openAIWSConnLease) UpstreamPingCount() int64 {
	if l == nil || l.conn == nil {
		return 0
	}
	return l.conn.upstreamPingCount()
}

func (l *openAIWSConnLease) HandshakeHeader(name string) string {
	if l == nil || l.conn == nil {
		return ""
	}
	return l.conn.handshakeHeader(name)
}

func (l *openAIWSConnLease) HandshakeHeaders() http.Header {
	if l == nil || l.conn == nil {
		return nil
	}
	return cloneHeader(l.conn.handshakeHeaders)
}

func (l *openAIWSConnLease) IsPrewarmed() bool {
	if l == nil || l.conn == nil {
		return false
	}
	return l.conn.isPrewarmed()
}

func (l *openAIWSConnLease) MarkPrewarmed() {
	if l == nil || l.conn == nil {
		return
	}
	l.conn.markPrewarmed()
}

func (l *openAIWSConnLease) WriteJSON(value any, timeout time.Duration) error {
	conn, err := l.activeConn()
	if err != nil {
		return err
	}
	return conn.writeJSONWithTimeout(context.Background(), value, timeout)
}

func (l *openAIWSConnLease) WriteJSONWithContextTimeout(ctx context.Context, value any, timeout time.Duration) error {
	conn, err := l.activeConn()
	if err != nil {
		return err
	}
	return conn.writeJSONWithTimeout(ctx, value, timeout)
}

func (l *openAIWSConnLease) WriteJSONContext(ctx context.Context, value any) error {
	conn, err := l.activeConn()
	if err != nil {
		return err
	}
	return conn.writeJSON(value, ctx)
}

func (l *openAIWSConnLease) ReadMessage(timeout time.Duration) ([]byte, error) {
	conn, err := l.activeConn()
	if err != nil {
		return nil, err
	}
	return conn.readMessageWithTimeout(timeout)
}

func (l *openAIWSConnLease) ReadMessageContext(ctx context.Context) ([]byte, error) {
	conn, err := l.activeConn()
	if err != nil {
		return nil, err
	}
	return conn.readMessage(ctx)
}

func (l *openAIWSConnLease) ReadMessageWithContextTimeout(ctx context.Context, timeout time.Duration) ([]byte, error) {
	conn, err := l.activeConn()
	if err != nil {
		return nil, err
	}
	return conn.readMessageWithContextTimeout(ctx, timeout)
}

func (l *openAIWSConnLease) PingWithTimeout(timeout time.Duration) error {
	conn, err := l.activeConn()
	if err != nil {
		return err
	}
	return conn.pingWithTimeout(timeout)
}

func (l *openAIWSConnLease) SupportsIdlePingWithoutReader() bool {
	conn, err := l.activeConn()
	if err != nil {
		return false
	}
	return conn.supportsIdlePingWithoutReader()
}

func (l *openAIWSConnLease) MarkBroken() {
	if l == nil || l.pool == nil || l.conn == nil || l.released.Load() {
		return
	}
	l.pool.evictConn(l.accountID, l.conn.id)
}

func (l *openAIWSConnLease) Release() {
	if l == nil || l.conn == nil || !l.released.CompareAndSwap(false, true) {
		return
	}
	l.conn.release()
	if l.pool != nil {
		if l.pool.stopped.Load() {
			l.pool.evictConn(l.accountID, l.conn.id)
			return
		}
		l.pool.notifyAccountPoolChanged(l.accountID)
		l.pool.requestCleanup()
	}
}

type openAIWSConn struct {
	id string
	ws openAIWSClientConn

	handshakeHeaders       http.Header
	handshakeCompatibility openAIWSHandshakeCompatibilityKey
	routingAffinity        string
	wsURL                  string
	proxyURL               string

	leaseCh   chan struct{}
	closedCh  chan struct{}
	closeOnce sync.Once

	readMu  sync.Mutex
	writeMu sync.Mutex

	// readerLoopResults 非 nil 表示池为该连接常驻了读循环：coder/websocket 只在
	// 阻塞读期间应答上游 ping，空闲连接没有读循环会被上游按保活超时关闭。
	readerLoopResults    chan []byte
	readerLoopErrMu      sync.Mutex
	readerLoopErr        error
	readerLoopPeerClosed atomic.Bool
	// onPeerClosed 由池在建连后设置：上游主动关闭时立刻把连接移出账号池，不等清理周期。
	onPeerClosed atomic.Pointer[func()]
	// unusable 表示空闲期收到数据被判为脏连接：持有令牌不再借出，由池在锁外关闭。
	unusable atomic.Bool

	waiters       atomic.Int32
	createdAtNano atomic.Int64
	lastUsedNano  atomic.Int64
	prewarmed     atomic.Bool
}

func newOpenAIWSConn(id string, _ int64, ws openAIWSClientConn, handshakeHeaders http.Header) *openAIWSConn {
	now := time.Now()
	conn := &openAIWSConn{
		id:               id,
		ws:               ws,
		handshakeHeaders: cloneHeader(handshakeHeaders),
		leaseCh:          make(chan struct{}, 1),
		closedCh:         make(chan struct{}),
	}
	conn.leaseCh <- struct{}{}
	conn.createdAtNano.Store(now.UnixNano())
	conn.lastUsedNano.Store(now.UnixNano())
	if capable, ok := ws.(openAIWSReaderLoopCapable); ok && capable.RequiresReaderLoop() {
		conn.readerLoopResults = make(chan []byte, 1)
		go conn.runReaderLoop()
	}
	return conn
}

func (c *openAIWSConn) runReaderLoop() {
	defer close(c.readerLoopResults)
	for {
		payload, err := c.ws.ReadMessage(context.Background())
		if err != nil {
			c.readerLoopErrMu.Lock()
			c.readerLoopErr = err
			c.readerLoopErrMu.Unlock()
			// 本地主动关闭时对端会回 close 帧，同样以读错误结束循环，不算上游事件。
			peerClosed := false
			select {
			case <-c.closedCh:
			default:
				peerClosed = true
				c.readerLoopPeerClosed.Store(true)
				now := time.Now()
				// 空闲连接被上游断开是常态，池已当场出池，只记 info；借出中断开会影响请求，记 warn。
				logClosed := logOpenAIWSModeInfo
				if c.isLeased() {
					logClosed = logOpenAIWSModeWarn
				}
				logClosed(
					"conn_reader_loop_closed conn_id=%s leased=%v idle_ms=%d age_ms=%d upstream_pings=%d cause=%s",
					c.id,
					c.isLeased(),
					c.idleDuration(now).Milliseconds(),
					c.age(now).Milliseconds(),
					c.upstreamPingCount(),
					truncateOpenAIWSLogValue(err.Error(), openAIWSLogValueMaxLen),
				)
			}
			c.close()
			if evict := c.onPeerClosed.Load(); peerClosed && evict != nil {
				(*evict)()
			}
			return
		}
		select {
		case c.readerLoopResults <- payload:
		case <-c.closedCh:
			return
		}
	}
}

func (c *openAIWSConn) hasReaderLoop() bool {
	return c != nil && c.readerLoopResults != nil
}

func (c *openAIWSConn) readerLoopClosedByPeer() bool {
	return c != nil && c.readerLoopPeerClosed.Load()
}

func (c *openAIWSConn) upstreamPingCount() int64 {
	if c == nil || c.ws == nil {
		return 0
	}
	if counter, ok := c.ws.(openAIWSUpstreamPingCounter); ok {
		return counter.UpstreamPingCount()
	}
	return 0
}

// readerLoopPending 报告空闲期是否已有数据消息被读循环缓存。len 不消费消息；
// 缓存满时读循环阻塞在投递上，因此最多只有这一条待接管消息。
func (c *openAIWSConn) readerLoopPending() bool {
	return c.hasReaderLoop() && len(c.readerLoopResults) > 0
}

func (c *openAIWSConn) readerLoopError() error {
	c.readerLoopErrMu.Lock()
	defer c.readerLoopErrMu.Unlock()
	if c.readerLoopErr != nil {
		return c.readerLoopErr
	}
	return errOpenAIWSConnClosed
}

// leaseTokenUsable 在拿到租约令牌后确认连接仍可借出：已关闭的连接退回令牌；
// 空闲期收到过数据消息的连接状态已不可信，直接关闭而不交给借用者。
func (c *openAIWSConn) leaseTokenUsable() bool {
	select {
	case <-c.closedCh:
		c.release()
		return false
	default:
	}
	if c.readerLoopPending() {
		// 只记事件类型，不记报文原文，避免模型输出进日志。
		eventType := ""
		select {
		case payload := <-c.readerLoopResults:
			eventType = effectiveOpenAISSEEventType(payload, "")
		default:
		}
		logOpenAIWSModeWarn(
			"conn_idle_dirty_discard conn_id=%s idle_ms=%d upstream_pings=%d event=%s",
			c.id,
			c.idleDuration(time.Now()).Milliseconds(),
			c.upstreamPingCount(),
			normalizeOpenAIWSLogValue(eventType),
		)
		// 关闭握手可能阻塞到一个 RTT，而 tryAcquire 在池锁内调用，这里只标记，出池后再关闭。
		c.unusable.Store(true)
		return false
	}
	return true
}

func (c *openAIWSConn) isClosed() bool {
	if c == nil {
		return true
	}
	select {
	case <-c.closedCh:
		return true
	default:
		return false
	}
}

func (c *openAIWSConn) isUnusable() bool {
	return c != nil && c.unusable.Load()
}

func (c *openAIWSConn) tryAcquire() bool {
	if c == nil {
		return false
	}
	select {
	case <-c.closedCh:
		return false
	default:
	}
	select {
	case <-c.leaseCh:
		return c.leaseTokenUsable()
	default:
		return false
	}
}

func (c *openAIWSConn) acquire(ctx context.Context) error {
	if c == nil {
		return errOpenAIWSConnClosed
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.closedCh:
			return errOpenAIWSConnClosed
		case <-c.leaseCh:
			// A cancellation and a lease delivery can become ready together. Once
			// the semaphore token has been consumed, check the context again and
			// return it before reporting cancellation so a canceled waiter cannot
			// strand a pooled connection.
			if err := ctx.Err(); err != nil {
				c.release()
				return err
			}
			if !c.leaseTokenUsable() {
				return errOpenAIWSConnClosed
			}
			return nil
		}
	}
}

func (c *openAIWSConn) release() {
	if c == nil {
		return
	}
	select {
	case c.leaseCh <- struct{}{}:
	default:
	}
	c.touch()
}

func (c *openAIWSConn) close() {
	c.closeWith(false)
}

// abort 不做关闭握手直接切断。读循环常驻持有读锁，礼貌关闭要等对端回 close 帧，
// 对端已不响应时库会等满 5 秒；读超时这类场景必须立即返回。
func (c *openAIWSConn) abort() {
	c.closeWith(true)
}

func (c *openAIWSConn) closeWith(force bool) {
	if c == nil {
		return
	}
	c.closeOnce.Do(func() {
		close(c.closedCh)
		if c.ws != nil {
			if forceCloser, ok := c.ws.(openAIWSForceCloser); ok && force {
				_ = forceCloser.CloseNow()
			} else {
				_ = c.ws.Close()
			}
		}
		select {
		case c.leaseCh <- struct{}{}:
		default:
		}
	})
}

func (c *openAIWSConn) writeJSONWithTimeout(parent context.Context, value any, timeout time.Duration) error {
	if c == nil {
		return errOpenAIWSConnClosed
	}
	select {
	case <-c.closedCh:
		return errOpenAIWSConnClosed
	default:
	}

	writeCtx := parent
	if writeCtx == nil {
		writeCtx = context.Background()
	}
	if timeout <= 0 {
		return c.writeJSON(value, writeCtx)
	}
	var cancel context.CancelFunc
	writeCtx, cancel = context.WithTimeout(writeCtx, timeout)
	defer cancel()
	return c.writeJSON(value, writeCtx)
}

func (c *openAIWSConn) writeJSON(value any, writeCtx context.Context) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if c.ws == nil {
		return errOpenAIWSConnClosed
	}
	if writeCtx == nil {
		writeCtx = context.Background()
	}
	if err := c.ws.WriteJSON(writeCtx, value); err != nil {
		return err
	}
	c.touch()
	return nil
}

func (c *openAIWSConn) readMessageWithTimeout(timeout time.Duration) ([]byte, error) {
	return c.readMessageWithContextTimeout(context.Background(), timeout)
}

func (c *openAIWSConn) readMessageWithContextTimeout(parent context.Context, timeout time.Duration) ([]byte, error) {
	if c == nil {
		return nil, errOpenAIWSConnClosed
	}
	// 有读循环时连接关闭后缓冲里可能还有未取走的消息，交给 readMessage 先排空再报错。
	if c.readerLoopResults == nil {
		select {
		case <-c.closedCh:
			return nil, errOpenAIWSConnClosed
		default:
		}
	}

	if parent == nil {
		parent = context.Background()
	}
	if timeout <= 0 {
		return c.readMessage(parent)
	}
	readCtx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	return c.readMessage(readCtx)
}

func (c *openAIWSConn) readMessage(readCtx context.Context) ([]byte, error) {
	c.readMu.Lock()
	defer c.readMu.Unlock()
	if c.ws == nil {
		return nil, errOpenAIWSConnClosed
	}
	if readCtx == nil {
		readCtx = context.Background()
	}
	if c.readerLoopResults == nil {
		payload, err := c.ws.ReadMessage(readCtx)
		if err != nil {
			return nil, err
		}
		c.touch()
		return payload, nil
	}
	select {
	case payload, ok := <-c.readerLoopResults:
		if !ok {
			return nil, c.readerLoopError()
		}
		c.touch()
		return payload, nil
	case <-readCtx.Done():
		// 与库在 ctx 取消时切断连接的语义一致：读超时后消息边界已不可信，且对端多半
		// 已不响应，直接切断而不做关闭握手。
		c.abort()
		return nil, readCtx.Err()
	}
}

func (c *openAIWSConn) pingWithTimeout(timeout time.Duration) error {
	if c == nil {
		return errOpenAIWSConnClosed
	}
	select {
	case <-c.closedCh:
		return errOpenAIWSConnClosed
	default:
	}

	// coder/websocket 除 Reader/Read 外的方法都可并发调用，控制帧由库内 writeFrameMu 串行化，
	// 这里不持有 writeMu，避免等 pong 期间阻塞借用者写请求。
	if c.ws == nil {
		return errOpenAIWSConnClosed
	}
	if timeout <= 0 {
		timeout = openAIWSConnHealthCheckTO
	}
	pingCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := c.ws.Ping(pingCtx); err != nil {
		return err
	}
	return nil
}

func (c *openAIWSConn) supportsIdlePingWithoutReader() bool {
	if c == nil || c.ws == nil {
		return false
	}
	if c.readerLoopResults != nil {
		return true
	}
	capable, ok := c.ws.(openAIWSIdlePingCapable)
	// Test and alternate implementations keep the historical probe behavior
	// unless they explicitly declare it unsafe.
	return !ok || capable.SupportsIdlePingWithoutReader()
}

func (c *openAIWSConn) touch() {
	if c == nil {
		return
	}
	c.lastUsedNano.Store(time.Now().UnixNano())
}

func (c *openAIWSConn) createdAt() time.Time {
	if c == nil {
		return time.Time{}
	}
	nano := c.createdAtNano.Load()
	if nano <= 0 {
		return time.Time{}
	}
	return time.Unix(0, nano)
}

func (c *openAIWSConn) lastUsedAt() time.Time {
	if c == nil {
		return time.Time{}
	}
	nano := c.lastUsedNano.Load()
	if nano <= 0 {
		return time.Time{}
	}
	return time.Unix(0, nano)
}

func (c *openAIWSConn) idleDuration(now time.Time) time.Duration {
	if c == nil {
		return 0
	}
	last := c.lastUsedAt()
	if last.IsZero() {
		return 0
	}
	return now.Sub(last)
}

func (c *openAIWSConn) age(now time.Time) time.Duration {
	if c == nil {
		return 0
	}
	created := c.createdAt()
	if created.IsZero() {
		return 0
	}
	return now.Sub(created)
}

func (c *openAIWSConn) isLeased() bool {
	if c == nil {
		return false
	}
	return len(c.leaseCh) == 0
}

func (c *openAIWSConn) handshakeHeader(name string) string {
	if c == nil || c.handshakeHeaders == nil {
		return ""
	}
	return strings.TrimSpace(c.handshakeHeaders.Get(strings.TrimSpace(name)))
}

func (c *openAIWSConn) matchesRoutingAffinity(routingAffinity string) bool {
	return c != nil && c.routingAffinity == routingAffinity
}

func (c *openAIWSConn) isPrewarmed() bool {
	if c == nil {
		return false
	}
	return c.prewarmed.Load()
}

func (c *openAIWSConn) markPrewarmed() {
	if c == nil {
		return
	}
	c.prewarmed.Store(true)
}
