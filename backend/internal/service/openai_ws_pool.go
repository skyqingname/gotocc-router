package service

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/config"
)

const (
	openAIWSConnMaxAge          = 60 * time.Minute
	openAIWSConnHealthCheckIdle = 90 * time.Second
	// 仅对没有常驻读循环的连接实现生效：这类连接空闲时无人应答上游 ping，须在
	// 上游保活窗口到期前回收。coder/websocket 连接由池常驻读循环应答 ping，不受此阈值约束。
	openAIWSConnIdleRecycleAfter = 90 * time.Second
	openAIWSConnHealthCheckTO    = 2 * time.Second
	// 不在请求热路径上的探活（后台巡检、轮次间预检）给经代理链路的 pong 留足余量，
	// 实测最大往返约 1.7s；误判的代价是换连甚至断会话，比多等几秒重得多。
	openAIWSProbePingTO            = 10 * time.Second
	openAIWSConnPrewarmExtraDelay  = 2 * time.Second
	openAIWSAcquireCleanupInterval = 3 * time.Second
	openAIWSBackgroundPingInterval = 30 * time.Second
	openAIWSBackgroundSweepTicker  = 30 * time.Second

	openAIWSPrewarmFailureWindow   = 30 * time.Second
	openAIWSPrewarmFailureSuppress = 2
)

var (
	errOpenAIWSConnClosed               = errors.New("openai ws connection closed")
	errOpenAIWSConnQueueFull            = errors.New("openai ws connection queue full")
	errOpenAIWSPreferredConnUnavailable = errors.New("openai ws preferred connection unavailable")
)

type openAIWSAccountPool struct {
	mu            sync.Mutex
	conns         map[string]*openAIWSConn
	pinnedConns   map[string]int
	changedCh     chan struct{}
	waiters       map[openAIWSConnectionKey]int
	creating      int
	generation    uint64
	lastCleanupAt time.Time
	lastAcquire   *openAIWSAcquireRequest
	prewarmActive bool
	prewarmUntil  time.Time
	prewarmFails  int
	prewarmFailAt time.Time
}

func (ap *openAIWSAccountPool) changeChannelLocked() chan struct{} {
	if ap.changedCh == nil {
		ap.changedCh = make(chan struct{})
	}
	return ap.changedCh
}

func (ap *openAIWSAccountPool) signalChangedLocked() {
	if ap == nil {
		return
	}
	if ap.changedCh != nil {
		close(ap.changedCh)
	}
	ap.changedCh = make(chan struct{})
}

type OpenAIWSPoolMetricsSnapshot struct {
	AcquireTotal            int64
	AcquireReuseTotal       int64
	AcquireCreateTotal      int64
	AcquireQueueWaitTotal   int64
	AcquireQueueWaitMsTotal int64
	ConnPickTotal           int64
	ConnPickMsTotal         int64
	ScaleUpTotal            int64
	ScaleDownTotal          int64
}

type openAIWSPoolMetrics struct {
	acquireTotal          atomic.Int64
	acquireReuseTotal     atomic.Int64
	acquireCreateTotal    atomic.Int64
	acquireQueueWaitTotal atomic.Int64
	acquireQueueWaitMs    atomic.Int64
	connPickTotal         atomic.Int64
	connPickMs            atomic.Int64
	scaleUpTotal          atomic.Int64
	scaleDownTotal        atomic.Int64
}

type openAIWSConnPool struct {
	cfg *config.Config
	// 通过接口解耦底层 WS 客户端实现，默认使用 coder/websocket。
	clientDialer openAIWSClientDialer

	accounts sync.Map // key: int64(accountID), value: *openAIWSAccountPool
	seq      atomic.Uint64

	metrics openAIWSPoolMetrics

	lifecycleMu     sync.Mutex
	lifecycleCtx    context.Context
	cancelLifecycle context.CancelFunc
	stopped         atomic.Bool
	acquireWG       sync.WaitGroup
	prewarmWG       sync.WaitGroup
	cleanupWakeCh   chan struct{}
	workerStopCh    chan struct{}
	workerWg        sync.WaitGroup
	closeOnce       sync.Once
}

func newOpenAIWSConnPool(cfg *config.Config) *openAIWSConnPool {
	lifecycleCtx, cancelLifecycle := context.WithCancel(context.Background())
	pool := &openAIWSConnPool{
		lifecycleCtx:    lifecycleCtx,
		cancelLifecycle: cancelLifecycle,
		cleanupWakeCh:   make(chan struct{}, 1),
		cfg:             cfg,
		clientDialer:    newDefaultOpenAIWSClientDialer(),
		workerStopCh:    make(chan struct{}),
	}
	pool.startBackgroundWorkers()
	return pool
}

func (p *openAIWSConnPool) SnapshotMetrics() OpenAIWSPoolMetricsSnapshot {
	if p == nil {
		return OpenAIWSPoolMetricsSnapshot{}
	}
	return OpenAIWSPoolMetricsSnapshot{
		AcquireTotal:            p.metrics.acquireTotal.Load(),
		AcquireReuseTotal:       p.metrics.acquireReuseTotal.Load(),
		AcquireCreateTotal:      p.metrics.acquireCreateTotal.Load(),
		AcquireQueueWaitTotal:   p.metrics.acquireQueueWaitTotal.Load(),
		AcquireQueueWaitMsTotal: p.metrics.acquireQueueWaitMs.Load(),
		ConnPickTotal:           p.metrics.connPickTotal.Load(),
		ConnPickMsTotal:         p.metrics.connPickMs.Load(),
		ScaleUpTotal:            p.metrics.scaleUpTotal.Load(),
		ScaleDownTotal:          p.metrics.scaleDownTotal.Load(),
	}
}

func (p *openAIWSConnPool) SnapshotTransportMetrics() OpenAIWSTransportMetricsSnapshot {
	if p == nil {
		return OpenAIWSTransportMetricsSnapshot{}
	}
	if dialer, ok := p.clientDialer.(openAIWSTransportMetricsDialer); ok {
		return dialer.SnapshotTransportMetrics()
	}
	return OpenAIWSTransportMetricsSnapshot{}
}

func (p *openAIWSConnPool) setClientDialerForTest(dialer openAIWSClientDialer) {
	if p == nil || dialer == nil {
		return
	}
	p.clientDialer = dialer
}

// Close 停止后台 worker 并关闭所有空闲连接，应在优雅关闭时调用。
func (p *openAIWSConnPool) Acquire(ctx context.Context, req openAIWSAcquireRequest) (*openAIWSConnLease, error) {
	if p == nil {
		return nil, errors.New("invalid ws acquire request")
	}
	p.lifecycleMu.Lock()
	if p.stopped.Load() {
		p.lifecycleMu.Unlock()
		return nil, errOpenAIWSConnClosed
	}
	p.acquireWG.Add(1)
	p.lifecycleMu.Unlock()
	defer p.acquireWG.Done()
	acquireCtx, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(p.lifecycleCtx, cancel)
	defer func() { stop(); cancel() }()
	p.metrics.acquireTotal.Add(1)
	return p.acquireFromPool(acquireCtx, cloneOpenAIWSAcquireRequest(req))
}

func (p *openAIWSConnPool) recordConnPickDuration(duration time.Duration) {
	if p == nil {
		return
	}
	if duration < 0 {
		duration = 0
	}
	p.metrics.connPickTotal.Add(1)
	p.metrics.connPickMs.Add(duration.Milliseconds())
}

func (p *openAIWSConnPool) recordLastSuccessfulAcquire(accountID int64, generation uint64, req openAIWSAcquireRequest) {
	if p == nil || accountID <= 0 {
		return
	}
	ap, ok := p.getAccountPool(accountID)
	if !ok || ap == nil {
		return
	}
	ap.mu.Lock()
	if ap.generation != generation {
		ap.mu.Unlock()
		return
	}
	ap.lastAcquire = cloneOpenAIWSAcquireRequestPtr(&req)
	ap.mu.Unlock()
}

func (p *openAIWSConnPool) getOrCreateAccountPool(accountID int64) *openAIWSAccountPool {
	if p == nil || accountID <= 0 {
		return nil
	}
	if existing, ok := p.accounts.Load(accountID); ok {
		if ap, typed := existing.(*openAIWSAccountPool); typed && ap != nil {
			return ap
		}
	}
	ap := &openAIWSAccountPool{
		waiters:     make(map[openAIWSConnectionKey]int),
		conns:       make(map[string]*openAIWSConn),
		pinnedConns: make(map[string]int),
		changedCh:   make(chan struct{}),
	}
	actual, _ := p.accounts.LoadOrStore(accountID, ap)
	if typed, ok := actual.(*openAIWSAccountPool); ok && typed != nil {
		return typed
	}
	return ap
}

// ensureAccountPoolLocked 兼容旧调用。
func (p *openAIWSConnPool) ensureAccountPoolLocked(accountID int64) *openAIWSAccountPool {
	return p.getOrCreateAccountPool(accountID)
}

func (p *openAIWSConnPool) getAccountPool(accountID int64) (*openAIWSAccountPool, bool) {
	if p == nil || accountID <= 0 {
		return nil, false
	}
	value, ok := p.accounts.Load(accountID)
	if !ok || value == nil {
		return nil, false
	}
	ap, typed := value.(*openAIWSAccountPool)
	return ap, typed && ap != nil
}

func (p *openAIWSConnPool) notifyAccountPoolChanged(accountID int64) {
	ap, ok := p.getAccountPool(accountID)
	if !ok || ap == nil {
		return
	}
	ap.mu.Lock()
	ap.signalChangedLocked()
	ap.mu.Unlock()
}

func (p *openAIWSConnPool) isConnPinnedLocked(ap *openAIWSAccountPool, connID string) bool {
	if ap == nil || connID == "" || len(ap.pinnedConns) == 0 {
		return false
	}
	return ap.pinnedConns[connID] > 0
}

func accountPoolLoadLocked(ap *openAIWSAccountPool) (inflight int, waiters int) {
	if ap == nil {
		return 0, 0
	}
	for _, conn := range ap.conns {
		if conn == nil {
			continue
		}
		if conn.isLeased() {
			inflight++
		}
		waiters += int(conn.waiters.Load())
	}
	for _, count := range ap.waiters {
		waiters += count
	}
	return inflight, waiters
}

// AccountPoolLoad 返回指定账号连接池的并发与排队快照。
func (p *openAIWSConnPool) AccountPoolLoad(accountID int64) (inflight int, waiters int, conns int) {
	if p == nil || accountID <= 0 {
		return 0, 0, 0
	}
	ap, ok := p.getAccountPool(accountID)
	if !ok || ap == nil {
		return 0, 0, 0
	}
	ap.mu.Lock()
	defer ap.mu.Unlock()
	inflight, waiters = accountPoolLoadLocked(ap)
	return inflight, waiters, len(ap.conns)
}

// ClearAccount closes all pooled connections and discards delayed prewarm
// state for one account. The generation guard prevents an in-flight prewarm
// started before credential recovery from re-entering the pool afterwards.
func (p *openAIWSConnPool) ClearAccount(accountID int64) {
	if p == nil || accountID <= 0 {
		return
	}
	ap, ok := p.getAccountPool(accountID)
	if !ok || ap == nil {
		return
	}
	ap.mu.Lock()
	ap.generation++
	conns := make([]*openAIWSConn, 0, len(ap.conns))
	for id, conn := range ap.conns {
		delete(ap.conns, id)
		delete(ap.pinnedConns, id)
		if conn != nil {
			conns = append(conns, conn)
		}
	}
	ap.lastAcquire = nil
	ap.prewarmUntil = time.Time{}
	ap.prewarmFails = 0
	ap.prewarmFailAt = time.Time{}
	ap.signalChangedLocked()
	ap.mu.Unlock()
	closeOpenAIWSConns(conns)
}

func (p *openAIWSConnPool) evictConn(accountID int64, connID string) {
	if p == nil || accountID <= 0 || stringsTrim(connID) == "" {
		return
	}
	var conn *openAIWSConn
	ap, ok := p.getAccountPool(accountID)
	if ok && ap != nil {
		ap.mu.Lock()
		if c, exists := ap.conns[connID]; exists {
			conn = c
			delete(ap.conns, connID)
			if len(ap.pinnedConns) > 0 {
				delete(ap.pinnedConns, connID)
			}
			ap.signalChangedLocked()
		}
		ap.mu.Unlock()
	}
	if conn != nil {
		conn.close()
	}
}

func (p *openAIWSConnPool) PinConn(accountID int64, connID string) bool {
	if p == nil || accountID <= 0 {
		return false
	}
	connID = stringsTrim(connID)
	if connID == "" {
		return false
	}
	ap, ok := p.getAccountPool(accountID)
	if !ok || ap == nil {
		return false
	}
	ap.mu.Lock()
	defer ap.mu.Unlock()
	if _, exists := ap.conns[connID]; !exists {
		return false
	}
	if ap.pinnedConns == nil {
		ap.pinnedConns = make(map[string]int)
	}
	ap.pinnedConns[connID]++
	return true
}

func (p *openAIWSConnPool) UnpinConn(accountID int64, connID string) {
	if p == nil || accountID <= 0 {
		return
	}
	connID = stringsTrim(connID)
	if connID == "" {
		return
	}
	ap, ok := p.getAccountPool(accountID)
	if !ok || ap == nil {
		return
	}
	ap.mu.Lock()
	defer ap.mu.Unlock()
	if len(ap.pinnedConns) == 0 {
		return
	}
	count := ap.pinnedConns[connID]
	if count <= 1 {
		delete(ap.pinnedConns, connID)
		ap.signalChangedLocked()
		return
	}
	ap.pinnedConns[connID] = count - 1
	ap.signalChangedLocked()
}

func (p *openAIWSConnPool) dialConn(ctx context.Context, req openAIWSAcquireRequest) (*openAIWSConn, error) {
	if p == nil || p.clientDialer == nil {
		return nil, errors.New("openai ws client dialer is nil")
	}
	headers := cloneHeader(req.Headers)
	var err error
	if req.HeadersFactory != nil {
		headers, err = req.HeadersFactory(ctx, headers)
		if err != nil {
			return nil, err
		}
	}
	conn, status, handshakeHeaders, err := p.clientDialer.Dial(ctx, req.WSURL, headers, req.ProxyURL)
	if err != nil {
		var handshakeErr *openAIWSHandshakeError
		var responseBody []byte
		if errors.As(err, &handshakeErr) && handshakeErr != nil {
			responseBody = append([]byte(nil), handshakeErr.Body...)
		}
		return nil, &openAIWSDialError{
			StatusCode:      status,
			ResponseHeaders: cloneHeader(handshakeHeaders),
			ResponseBody:    responseBody,
			Err:             err,
		}
	}
	if conn == nil {
		return nil, &openAIWSDialError{
			StatusCode:      status,
			ResponseHeaders: cloneHeader(handshakeHeaders),
			Err:             errors.New("openai ws dialer returned nil connection"),
		}
	}
	id := p.nextConnID(req.Account.ID)
	pooledConn := newOpenAIWSConn(id, req.Account.ID, conn, handshakeHeaders)
	accountID := req.Account.ID
	evict := func() { p.evictConn(accountID, id) }
	pooledConn.onPeerClosed.Store(&evict)
	pooledConn.wsURL = stringsTrim(req.WSURL)
	pooledConn.proxyURL = stringsTrim(req.ProxyURL)
	pooledConn.handshakeCompatibility = normalizeOpenAIWSHandshakeCompatibility(req.Headers)
	pooledConn.routingAffinity = normalizeOpenAIWSRoutingAffinity(req.Headers)
	return pooledConn, nil
}

func (p *openAIWSConnPool) nextConnID(accountID int64) string {
	seq := p.seq.Add(1)
	buf := make([]byte, 0, 32)
	buf = append(buf, "oa_ws_"...)
	buf = strconv.AppendInt(buf, accountID, 10)
	buf = append(buf, '_')
	buf = strconv.AppendUint(buf, seq, 10)
	return string(buf)
}

func (p *openAIWSConnPool) shouldHealthCheckConn(conn *openAIWSConn) bool {
	if conn == nil || !conn.supportsIdlePingWithoutReader() {
		return false
	}
	// 有读循环的连接能即时感知上游关闭，半开探测已由后台巡检覆盖，借出前不再多付一个往返。
	if conn.hasReaderLoop() {
		return false
	}
	return conn.idleDuration(time.Now()) >= openAIWSConnHealthCheckIdle
}
