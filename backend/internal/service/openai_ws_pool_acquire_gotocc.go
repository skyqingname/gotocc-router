package service

import (
	"context"
	"errors"
	"time"
)

// openAIWSConnectionKey is shared by reuse, queued requests and prewarm.
// Routing hints remain a preference; changing an upstream or proxy is a new target.
type openAIWSConnectionKey struct {
	wsURL     string
	proxyURL  string
	handshake openAIWSHandshakeCompatibilityKey
}

func openAIWSKeyForRequest(req openAIWSAcquireRequest) openAIWSConnectionKey {
	return openAIWSConnectionKey{
		wsURL: stringsTrim(req.WSURL), proxyURL: stringsTrim(req.ProxyURL),
		handshake: normalizeOpenAIWSHandshakeCompatibility(req.Headers),
	}
}

func (c *openAIWSConn) connectionKey() openAIWSConnectionKey {
	return openAIWSConnectionKey{wsURL: c.wsURL, proxyURL: c.proxyURL, handshake: c.handshakeCompatibility}
}

type openAIWSPoolWaiter struct {
	key       openAIWSConnectionKey
	preferred *openAIWSConn
	started   time.Time
}

func (p *openAIWSConnPool) registerWaiterLocked(ap *openAIWSAccountPool, key openAIWSConnectionKey, preferred *openAIWSConn, maxConns int) (*openAIWSPoolWaiter, error) {
	_, count := accountPoolLoadLocked(ap)
	limit := p.queueLimitPerConn()
	if count/limit >= maxConns || preferred != nil && int(preferred.waiters.Load()) >= limit {
		return nil, errOpenAIWSConnQueueFull
	}
	waiter := &openAIWSPoolWaiter{key: key, preferred: preferred, started: time.Now()}
	if preferred != nil {
		preferred.waiters.Add(1)
	} else {
		ap.waiters[key]++
	}
	return waiter, nil
}

func (ap *openAIWSAccountPool) removeWaiterLocked(waiter *openAIWSPoolWaiter) time.Duration {
	if waiter.preferred != nil {
		waiter.preferred.waiters.Add(-1)
	} else {
		ap.waiters[waiter.key]--
		if ap.waiters[waiter.key] == 0 {
			delete(ap.waiters, waiter.key)
		}
	}
	return time.Since(waiter.started)
}

func (p *openAIWSConnPool) acquireFromPool(ctx context.Context, req openAIWSAcquireRequest) (lease *openAIWSConnLease, err error) {
	if req.Account == nil || req.Account.ID <= 0 {
		return nil, errors.New("invalid ws acquire request")
	}
	if req.WSURL == "" {
		return nil, errors.New("ws url is empty")
	}
	key := openAIWSKeyForRequest(req)
	accountID := req.Account.ID
	affinity := normalizeOpenAIWSRoutingAffinity(req.Headers)
	forcePreferred := req.ForcePreferredConn && !req.ForceNewConn
	ap := p.getOrCreateAccountPool(accountID)
	var waiter *openAIWSPoolWaiter
	var waited time.Duration
	queued := false
	recoveryAttempts := 0
	finishWaiting := func() {
		if waiter != nil {
			waited += ap.removeWaiterLocked(waiter)
			waiter = nil
			p.requestCleanup()
		}
	}
	defer func() {
		ap.mu.Lock()
		finishWaiting()
		ap.mu.Unlock()
		if queued {
			p.metrics.acquireQueueWaitMs.Add(waited.Milliseconds())
		}
		if lease != nil {
			lease.queueWait = waited
		}
	}()

	for {
		if p.stopped.Load() {
			return nil, errOpenAIWSConnClosed
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		limit := p.effectiveMaxConnsByAccount(req.Account)
		if limit <= 0 {
			return nil, errOpenAIWSConnQueueFull
		}
		ap.mu.Lock()
		generation := ap.generation
		now := time.Now()
		retired := p.retireExpiredConnsLocked(ap, now)
		if ap.lastCleanupAt.IsZero() || now.Sub(ap.lastCleanupAt) >= openAIWSAcquireCleanupInterval {
			retired = append(retired, p.cleanupAccountLocked(ap, now, limit)...)
			ap.lastCleanupAt = now
		}
		pickStart := time.Now()
		var preferred, selected *openAIWSConn
		if forcePreferred {
			preferred = ap.conns[req.PreferredConnID]
			if preferred == nil || preferred.connectionKey() != key || p.connExpired(preferred, now) || preferred.isClosed() || preferred.isUnusable() {
				ap.mu.Unlock()
				closeOpenAIWSConns(retired)
				return nil, errOpenAIWSPreferredConnUnavailable
			}
			if preferred.tryAcquire() {
				selected = preferred
			}
		} else if !req.ForceNewConn {
			allowOtherAffinity := affinity == "" || waiter != nil || len(ap.conns)+ap.creating >= limit
			selected = p.takeAvailableConnLocked(ap, req.PreferredConnID, key, affinity, allowOtherAffinity)
		}
		pickDuration := time.Since(pickStart)
		p.recordConnPickDuration(pickDuration)
		if selected != nil {
			finishWaiting()
			ap.mu.Unlock()
			closeOpenAIWSConns(retired)
			if p.shouldHealthCheckConn(selected) {
				if pingErr := selected.pingWithTimeout(openAIWSConnHealthCheckTO); pingErr != nil {
					p.evictConn(accountID, selected.id)
					if recoveryAttempts == 0 {
						recoveryAttempts++
						continue
					}
					return nil, pingErr
				}
			}
			return p.deliverLease(ctx, req, generation, selected, true, pickDuration)
		}
		// Taking an idle connection can discover unread data or a closed peer.
		retired = append(retired, p.retireExpiredConnsLocked(ap, now)...)
		if !forcePreferred && len(ap.conns)+ap.creating >= limit {
			var idle *openAIWSConn
			if req.ForceNewConn {
				idle = p.pickOldestIdleConnLocked(ap)
			} else {
				idle = p.oldestReplaceableConnLocked(ap, key, false)
			}
			if idle != nil {
				delete(ap.conns, idle.id)
				retired = append(retired, idle)
				p.metrics.scaleDownTotal.Add(1)
			}
		}
		if !forcePreferred && len(ap.conns)+ap.creating < limit {
			finishWaiting()
			ap.creating++
			ap.mu.Unlock()
			closeOpenAIWSConns(retired)
			conn, dialErr := p.dialConn(ctx, req)
			ap.mu.Lock()
			ap.creating--
			ap.signalChangedLocked()
			if generation != ap.generation {
				ap.mu.Unlock()
				conn.close()
				if recoveryAttempts == 0 {
					recoveryAttempts++
					continue
				}
				return nil, errOpenAIWSConnClosed
			}
			if dialErr != nil {
				ap.prewarmFails++
				ap.prewarmFailAt = time.Now()
				ap.mu.Unlock()
				return nil, dialErr
			}
			if p.stopped.Load() || !conn.tryAcquire() {
				ap.mu.Unlock()
				conn.close()
				return nil, errOpenAIWSConnClosed
			}
			// The request that opened this connection owns its first lease.
			ap.conns[conn.id] = conn
			ap.prewarmFails = 0
			ap.prewarmFailAt = time.Time{}
			ap.mu.Unlock()
			return p.deliverLease(ctx, req, generation, conn, false, pickDuration)
		}
		if req.ForceNewConn {
			ap.mu.Unlock()
			closeOpenAIWSConns(retired)
			return nil, errOpenAIWSConnQueueFull
		}
		if waiter == nil {
			waiter, err = p.registerWaiterLocked(ap, key, preferred, limit)
			if err != nil {
				ap.mu.Unlock()
				closeOpenAIWSConns(retired)
				return nil, err
			}
			if !queued {
				p.metrics.acquireQueueWaitTotal.Add(1)
				queued = true
			}
		}
		changed := ap.changeChannelLocked()
		ap.mu.Unlock()
		closeOpenAIWSConns(retired)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-changed:
		}
	}
}

// deliverLease centralizes cancellation, timing and successful-acquire accounting.
func (p *openAIWSConnPool) deliverLease(ctx context.Context, req openAIWSAcquireRequest, generation uint64, conn *openAIWSConn, reused bool, picked time.Duration) (*openAIWSConnLease, error) {
	if err := ctx.Err(); err != nil {
		conn.release()
		p.notifyAccountPoolChanged(req.Account.ID)
		p.requestCleanup()
		return nil, err
	}
	if p.stopped.Load() || conn.isClosed() || conn.isUnusable() {
		conn.release()
		p.notifyAccountPoolChanged(req.Account.ID)
		p.requestCleanup()
		return nil, errOpenAIWSConnClosed
	}
	now := time.Now()
	lease := &openAIWSConnLease{
		pool: p, accountID: req.Account.ID, conn: conn, reused: reused,
		connPick: picked, idleBefore: conn.idleDuration(now), ageBefore: conn.age(now),
	}
	if reused {
		p.metrics.acquireReuseTotal.Add(1)
	} else {
		p.metrics.acquireCreateTotal.Add(1)
	}
	p.recordLastSuccessfulAcquire(req.Account.ID, generation, req)
	p.ensureTargetIdleAsync(req.Account.ID)
	return lease, nil
}

func (p *openAIWSConnPool) takeAvailableConnLocked(ap *openAIWSAccountPool, preferredID string, key openAIWSConnectionKey, affinity string, allowOther bool) *openAIWSConn {
	available := func(c *openAIWSConn) bool {
		return c != nil && c.connectionKey() == key && !c.isClosed() && !c.isUnusable() &&
			!c.isLeased() && c.waiters.Load() == 0 && !p.connExpired(c, time.Now()) &&
			(!p.isConnPinnedLocked(ap, c.id) || c.id == preferredID)
	}
	if c := ap.conns[preferredID]; available(c) && c.tryAcquire() {
		return c
	}
	for {
		var matching, other *openAIWSConn
		for _, c := range ap.conns {
			if !available(c) {
				continue
			}
			if c.matchesRoutingAffinity(affinity) {
				if matching == nil || c.lastUsedAt().Before(matching.lastUsedAt()) {
					matching = c
				}
			} else if allowOther && (other == nil || c.lastUsedAt().Before(other.lastUsedAt())) {
				other = c
			}
		}
		if matching != nil && matching.tryAcquire() {
			return matching
		}
		if other != nil && other.tryAcquire() {
			return other
		}
		if matching == nil && other == nil {
			return nil
		}
	}
}

func (p *openAIWSConnPool) pickOldestIdleConnLocked(ap *openAIWSAccountPool) *openAIWSConn {
	return p.oldestReplaceableConnLocked(ap, openAIWSConnectionKey{}, true)
}

func (p *openAIWSConnPool) oldestReplaceableConnLocked(ap *openAIWSAccountPool, key openAIWSConnectionKey, forceNew bool) *openAIWSConn {
	var oldest *openAIWSConn
	for id, conn := range ap.conns {
		if conn == nil || conn.isLeased() || conn.waiters.Load() > 0 ||
			p.isConnPinnedLocked(ap, id) || ap.waiters[conn.connectionKey()] > 0 {
			continue
		}
		if !forceNew && conn.connectionKey() == key {
			continue
		}
		if oldest == nil || conn.lastUsedAt().Before(oldest.lastUsedAt()) {
			oldest = conn
		}
	}
	return oldest
}
