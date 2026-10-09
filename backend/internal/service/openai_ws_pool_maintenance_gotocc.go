package service

import (
	"context"
	"math"
	"sort"
	"time"

	"golang.org/x/sync/errgroup"
)

func (p *openAIWSConnPool) requestCleanup() {
	select {
	case p.cleanupWakeCh <- struct{}{}:
	default:
	}
}

func (p *openAIWSConnPool) connExpired(conn *openAIWSConn, now time.Time) bool {
	return p.maxConnAge() > 0 && conn.age(now) >= p.maxConnAge()
}

// retireExpiredConnsLocked only detaches connections; closing performs IO outside the lock.
func (p *openAIWSConnPool) retireExpiredConnsLocked(ap *openAIWSAccountPool, now time.Time) []*openAIWSConn {
	var retired []*openAIWSConn
	for id, conn := range ap.conns {
		if conn == nil {
			delete(ap.conns, id)
			delete(ap.pinnedConns, id)
			continue
		}
		discard := conn.isClosed() || conn.isUnusable()
		if !discard && !conn.isLeased() && !p.isConnPinnedLocked(ap, id) {
			discard = p.connExpired(conn, now) || !conn.supportsIdlePingWithoutReader() && conn.idleDuration(now) >= openAIWSConnIdleRecycleAfter
		}
		if discard {
			delete(ap.conns, id)
			delete(ap.pinnedConns, id)
			retired = append(retired, conn)
		}
	}
	if len(retired) > 0 {
		p.metrics.scaleDownTotal.Add(int64(len(retired)))
		ap.signalChangedLocked()
	}
	return retired
}

func (p *openAIWSConnPool) prewarmNeededLocked(ap *openAIWSAccountPool) bool {
	if ap.lastAcquire == nil {
		return false
	}
	limit := p.effectiveMaxConnsByAccount(ap.lastAcquire.Account)
	if len(ap.conns) >= p.targetConnCountLocked(ap, limit) {
		return false
	}
	idle := 0
	for id, conn := range ap.conns {
		if !conn.isLeased() && !p.isConnPinnedLocked(ap, id) {
			idle++
		}
	}
	return idle < p.maxIdlePerAccount()
}

func (p *openAIWSConnPool) Close() {
	if p == nil {
		return
	}
	p.closeOnce.Do(func() {
		p.lifecycleMu.Lock()
		p.stopped.Store(true)
		p.cancelLifecycle()
		close(p.workerStopCh)
		p.lifecycleMu.Unlock()
		// Dialing and queue waits observe lifecycle cancellation. No new worker
		// can register after stopped is published under lifecycleMu.
		p.acquireWG.Wait()
		p.prewarmWG.Wait()
		p.workerWg.Wait()
		var idle []*openAIWSConn
		p.accounts.Range(func(_, value any) bool {
			ap := value.(*openAIWSAccountPool)
			ap.mu.Lock()
			for id, conn := range ap.conns {
				if conn != nil && !conn.isLeased() {
					delete(ap.conns, id)
					delete(ap.pinnedConns, id)
					idle = append(idle, conn)
				}
			}
			ap.signalChangedLocked()
			ap.mu.Unlock()
			return true
		})
		// An in-flight turn keeps its lease and closes it when returned.
		closeOpenAIWSConns(idle)
	})
}

func (p *openAIWSConnPool) startBackgroundWorkers() {
	if p == nil || p.workerStopCh == nil {
		return
	}
	p.workerWg.Add(2)
	go func() {
		defer p.workerWg.Done()
		p.runBackgroundPingWorker()
	}()
	go func() {
		defer p.workerWg.Done()
		p.runBackgroundCleanupWorker()
	}()
}

type openAIWSIdlePingCandidate struct {
	accountID int64
	conn      *openAIWSConn
}

func (p *openAIWSConnPool) runBackgroundPingWorker() {
	if p == nil {
		return
	}
	ticker := time.NewTicker(openAIWSBackgroundPingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			p.runBackgroundPingSweep()
		case <-p.workerStopCh:
			return
		}
	}
}

func (p *openAIWSConnPool) runBackgroundPingSweep() {
	if p == nil {
		return
	}
	candidates := p.snapshotIdleConnsForPing()
	var g errgroup.Group
	g.SetLimit(10)
	for _, item := range candidates {
		item := item
		if item.conn == nil || item.conn.isLeased() || item.conn.waiters.Load() > 0 || !item.conn.supportsIdlePingWithoutReader() {
			continue
		}
		g.Go(func() error {
			started := time.Now()
			idleMs := item.conn.idleDuration(started).Milliseconds()
			if err := item.conn.pingWithTimeout(openAIWSProbePingTO); err != nil {
				// 只有拿到租约令牌才能剔除：判断与占有必须是同一个原子动作，否则
				// 等 pong 期间刚借出的连接会被从借用者手里关掉。已关闭或已判脏的连接
				// 没有借用者，照常剔除。
				if !item.conn.tryAcquire() && !item.conn.isClosed() && !item.conn.isUnusable() {
					logOpenAIWSModeWarn(
						"conn_background_ping_skip_leased conn_id=%s idle_ms=%d upstream_pings=%d cause=%s",
						item.conn.id,
						idleMs,
						item.conn.upstreamPingCount(),
						truncateOpenAIWSLogValue(err.Error(), openAIWSLogValueMaxLen),
					)
					return nil
				}
				logOpenAIWSModeWarn(
					"conn_background_ping_evict conn_id=%s idle_ms=%d upstream_pings=%d cause=%s",
					item.conn.id,
					idleMs,
					item.conn.upstreamPingCount(),
					truncateOpenAIWSLogValue(err.Error(), openAIWSLogValueMaxLen),
				)
				p.evictConn(item.accountID, item.conn.id)
				return nil
			}
			logOpenAIWSModeDebug(
				"conn_background_ping_ok conn_id=%s idle_ms=%d rtt_ms=%d upstream_pings=%d",
				item.conn.id,
				idleMs,
				time.Since(started).Milliseconds(),
				item.conn.upstreamPingCount(),
			)
			return nil
		})
	}
	_ = g.Wait()
}

func (p *openAIWSConnPool) snapshotIdleConnsForPing() []openAIWSIdlePingCandidate {
	if p == nil {
		return nil
	}
	candidates := make([]openAIWSIdlePingCandidate, 0)
	p.accounts.Range(func(key, value any) bool {
		accountID, ok := key.(int64)
		if !ok || accountID <= 0 {
			return true
		}
		ap, ok := value.(*openAIWSAccountPool)
		if !ok || ap == nil {
			return true
		}
		ap.mu.Lock()
		for _, conn := range ap.conns {
			if conn == nil || conn.isLeased() || conn.waiters.Load() > 0 {
				continue
			}
			candidates = append(candidates, openAIWSIdlePingCandidate{
				accountID: accountID,
				conn:      conn,
			})
		}
		ap.mu.Unlock()
		return true
	})
	return candidates
}

func (p *openAIWSConnPool) runBackgroundCleanupWorker() {
	ticker := time.NewTicker(openAIWSBackgroundSweepTicker)
	defer ticker.Stop()
	for {
		select {
		case <-p.workerStopCh:
			return
		case now := <-ticker.C:
			p.runBackgroundCleanupSweep(now)
		case <-p.cleanupWakeCh:
			p.runBackgroundCleanupSweep(time.Now())
		}
	}
}

func (p *openAIWSConnPool) runBackgroundCleanupSweep(now time.Time) {
	if p == nil {
		return
	}
	type cleanupResult struct {
		evicted []*openAIWSConn
	}
	results := make([]cleanupResult, 0)
	p.accounts.Range(func(_ any, value any) bool {
		ap, ok := value.(*openAIWSAccountPool)
		if !ok || ap == nil {
			return true
		}
		maxConns := p.maxConnsHardCap()
		ap.mu.Lock()
		if ap.lastAcquire != nil && ap.lastAcquire.Account != nil {
			maxConns = p.effectiveMaxConnsByAccount(ap.lastAcquire.Account)
		}
		evicted := p.cleanupAccountLocked(ap, now, maxConns)
		ap.lastCleanupAt = now
		ap.mu.Unlock()
		if len(evicted) > 0 {
			results = append(results, cleanupResult{evicted: evicted})
		}
		return true
	})
	for _, result := range results {
		closeOpenAIWSConns(result.evicted)
	}
}

func (p *openAIWSConnPool) cleanupAccountLocked(ap *openAIWSAccountPool, now time.Time, maxConns int) []*openAIWSConn {
	if ap == nil {
		return nil
	}
	retired := p.retireExpiredConnsLocked(ap, now)
	if maxConns <= 0 {
		maxConns = p.maxConnsHardCap()
	}
	maxIdle := min(p.maxIdlePerAccount(), maxConns)
	idleCount := 0
	var available []*openAIWSConn
	for id, conn := range ap.conns {
		if conn.isLeased() || p.isConnPinnedLocked(ap, id) {
			continue
		}
		idleCount++
		if conn.waiters.Load() == 0 && ap.waiters[conn.connectionKey()] == 0 {
			available = append(available, conn)
		}
	}
	sort.Slice(available, func(i, j int) bool { return available[i].lastUsedAt().Before(available[j].lastUsedAt()) })
	redundant := min(len(available), max(0, max(idleCount-maxIdle, len(ap.conns)-maxConns)))
	for _, conn := range available[:redundant] {
		delete(ap.conns, conn.id)
		delete(ap.pinnedConns, conn.id)
		retired = append(retired, conn)
	}
	if redundant > 0 {
		p.metrics.scaleDownTotal.Add(int64(redundant))
		ap.signalChangedLocked()
	}
	return retired
}

func (p *openAIWSConnPool) ensureTargetIdleAsync(accountID int64) {
	if p == nil || accountID <= 0 {
		return
	}

	p.lifecycleMu.Lock()
	defer p.lifecycleMu.Unlock()
	if p.stopped.Load() {
		return
	}
	var req openAIWSAcquireRequest
	generation := uint64(0)
	need := 0
	ap, ok := p.getAccountPool(accountID)
	if !ok || ap == nil {
		return
	}
	ap.mu.Lock()
	defer ap.mu.Unlock()
	if ap.lastAcquire == nil {
		return
	}
	if ap.prewarmActive {
		return
	}
	now := time.Now()
	if !ap.prewarmUntil.IsZero() && now.Before(ap.prewarmUntil) {
		return
	}
	if p.shouldSuppressPrewarmLocked(ap, now) {
		return
	}
	effectiveMaxConns := p.maxConnsHardCap()
	if ap.lastAcquire != nil && ap.lastAcquire.Account != nil {
		effectiveMaxConns = p.effectiveMaxConnsByAccount(ap.lastAcquire.Account)
	}
	if !p.prewarmNeededLocked(ap) {
		return
	}
	target := p.targetConnCountLocked(ap, effectiveMaxConns)
	current := len(ap.conns) + ap.creating
	if current >= target {
		return
	}
	need = target - current
	if need <= 0 {
		return
	}
	req = cloneOpenAIWSAcquireRequest(*ap.lastAcquire)
	generation = ap.generation
	ap.prewarmActive = true
	if cooldown := p.prewarmCooldown(); cooldown > 0 {
		ap.prewarmUntil = now.Add(cooldown)
	}
	ap.creating += need
	p.metrics.scaleUpTotal.Add(int64(need))

	p.prewarmWG.Add(1)
	go func() { defer p.prewarmWG.Done(); p.prewarmConns(accountID, req, need, generation) }()
}

func (p *openAIWSConnPool) targetConnCountLocked(ap *openAIWSAccountPool, maxConns int) int {
	if ap == nil {
		return 0
	}

	if maxConns <= 0 {
		return 0
	}

	minIdle := p.minIdlePerAccount()
	if minIdle < 0 {
		minIdle = 0
	}
	if minIdle > maxConns {
		minIdle = maxConns
	}

	inflight, waiters := accountPoolLoadLocked(ap)
	utilization := p.targetUtilization()
	demand := inflight + waiters
	if demand <= 0 {
		return minIdle
	}

	target := 1
	if demand > 1 {
		target = int(math.Ceil(float64(demand) / utilization))
	}
	if waiters > 0 && target < len(ap.conns)+1 {
		target = len(ap.conns) + 1
	}
	if target < minIdle {
		target = minIdle
	}
	if target > maxConns {
		target = maxConns
	}
	return target
}

func (p *openAIWSConnPool) prewarmConns(accountID int64, req openAIWSAcquireRequest, total int, generations ...uint64) {
	generation := uint64(0)
	if len(generations) > 0 {
		generation = generations[0]
	}
	remaining := total
	staleTarget := false
	ap, _ := p.getAccountPool(accountID)
	defer func() {
		ap.mu.Lock()
		ap.creating -= remaining
		ap.prewarmActive = false
		ap.signalChangedLocked()
		ap.mu.Unlock()
		if staleTarget {
			p.ensureTargetIdleAsync(accountID)
		}
	}()
	for remaining > 0 {
		ap.mu.Lock()
		if p.stopped.Load() || ap.generation != generation || ap.lastAcquire == nil {
			ap.mu.Unlock()
			return
		}
		if !sameOpenAIWSPrewarmTarget(req, *ap.lastAcquire) {
			staleTarget = true
			ap.mu.Unlock()
			return
		}
		if !p.prewarmNeededLocked(ap) {
			ap.mu.Unlock()
			return
		}
		ap.mu.Unlock()
		ctx, cancel := context.WithTimeout(p.lifecycleCtx, p.dialTimeout()+openAIWSConnPrewarmExtraDelay)
		conn, err := p.dialConn(ctx, req)
		cancel()
		ap.mu.Lock()
		remaining--
		ap.creating--
		ap.signalChangedLocked()
		if err != nil {
			ap.prewarmFails++
			ap.prewarmFailAt = time.Now()
			ap.mu.Unlock()
			if p.stopped.Load() {
				return
			}
			continue
		}
		currentTarget := ap.generation == generation && ap.lastAcquire != nil && sameOpenAIWSPrewarmTarget(req, *ap.lastAcquire)
		if !currentTarget || p.stopped.Load() || !p.prewarmNeededLocked(ap) {
			staleTarget = !currentTarget
			ap.mu.Unlock()
			conn.close()
			return
		}
		ap.conns[conn.id] = conn
		ap.prewarmFails = 0
		ap.prewarmFailAt = time.Time{}
		ap.mu.Unlock()
	}
}
