package handler

import (
	"errors"

	"github.com/LuckyKuang/sub2api-plus/internal/service"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
)

// WebSocket 每轮准入与 HTTP 一致：用户或账号并发已满时在同一等待队列和上限内排队，
// 超时或队列满才关闭连接。排队期间连接由 keepalive 保活；c.Request 携带连接级
// context，客户端断开或连接租约失效时等待随之结束。

// acquireUserSlotWithWaitForAPIKey 按 HTTP 的用户槽规则排队；API Key 计数使用调用方
// 给出的 Key，与建连时登记的 Key 保持一致。
func (h *ConcurrencyHelper) acquireUserSlotWithWaitForAPIKey(c *gin.Context, userID int64, maxConcurrency int, apiKeyID int64) (func(), error) {
	ctx := c.Request.Context()
	release, acquired, err := h.TryAcquireUserSlot(ctx, userID, maxConcurrency)
	if err != nil {
		return nil, err
	}
	if !acquired {
		canWait, err := h.IncrementWaitCount(ctx, userID, max(service.CalculateMaxWait(maxConcurrency)-maxConcurrency, 1))
		if err != nil {
			return nil, err
		}
		if !canWait {
			return nil, &WaitQueueFullError{SlotType: "user"}
		}
		defer h.DecrementWaitCount(ctx, userID)
		streamStarted := false
		release, err = h.waitForSlotWithPingTimeout(c, "user", userID, maxConcurrency, maxConcurrencyWait, false, &streamStarted, false)
		if err != nil {
			return nil, err
		}
	}
	return h.withAPIKeySlot(ctx, apiKeyID, release), nil
}

// acquireAccountSlotWithWait 按等待计划排队获取账号槽，与 HTTP 共用账号等待计数。
func (h *ConcurrencyHelper) acquireAccountSlotWithWait(c *gin.Context, plan service.AccountWaitPlan) (func(), error) {
	ctx := c.Request.Context()
	release, acquired, err := h.TryAcquireAccountSlot(ctx, plan.AccountID, plan.MaxConcurrency)
	if err != nil || acquired {
		return release, err
	}
	canWait, err := h.IncrementAccountWaitCount(ctx, plan.AccountID, plan.MaxWaiting)
	if err != nil {
		return nil, err
	}
	if !canWait {
		return nil, &WaitQueueFullError{SlotType: "account"}
	}
	defer h.DecrementAccountWaitCount(ctx, plan.AccountID)
	streamStarted := false
	return h.waitForSlotWithPingTimeout(c, "account", plan.AccountID, plan.MaxConcurrency, plan.Timeout, false, &streamStarted, false)
}

// openAIWSBoundAccountWaitPlan 用于连接已绑定账号的后续轮次和同账号重试：
// 连接不能换号，等待限额与粘性会话相同。
func (h *OpenAIGatewayHandler) openAIWSBoundAccountWaitPlan(accountID int64, maxConcurrency int) service.AccountWaitPlan {
	return service.AccountWaitPlan{
		AccountID:      accountID,
		MaxConcurrency: maxConcurrency,
		Timeout:        h.cfg.Gateway.Scheduling.StickySessionWaitTimeout,
		MaxWaiting:     h.cfg.Gateway.Scheduling.StickySessionMaxWaiting,
	}
}

// openAIWSSlotWaitClose 把排队失败映射为原有的 WebSocket 关闭码与原因。
func openAIWSSlotWaitClose(err error, slotType string) (coderws.StatusCode, string) {
	var queueFull *WaitQueueFullError
	var concurrencyErr *ConcurrencyError
	if errors.As(err, &queueFull) || errors.As(err, &concurrencyErr) {
		if slotType == "user" {
			return coderws.StatusTryAgainLater, "too many concurrent requests, please retry later"
		}
		return coderws.StatusTryAgainLater, "account is busy, please retry later"
	}
	return coderws.StatusInternalError, "failed to acquire " + slotType + " concurrency slot"
}
