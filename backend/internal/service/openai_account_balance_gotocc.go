package service

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/accountbalance"
)

func useBalancedOpenAISelection(ctx context.Context, platform, previousResponseID string, previousResponseCanMove bool, guardianParentID int64, preserveGuardian bool) bool {
	if platform != PlatformOpenAI || guardianParentID > 0 || preserveGuardian {
		return false
	}
	if route, ok := AutoRouteDecisionFromContext(ctx); ok && route.RequiredAccountID > 0 {
		return false
	}
	return strings.TrimSpace(previousResponseID) == "" || previousResponseCanMove
}

// overflowFullStickyAccount serves one request on a same-priority account with a
// free slot while the session's account is at its concurrency limit. The session
// binding is preserved, so later requests return to that account and its prompt
// cache. Without a free same-priority slot the caller keeps the session's wait plan.
func (s *defaultOpenAIAccountScheduler) overflowFullStickyAccount(ctx context.Context, req OpenAIAccountScheduleRequest, sticky *AccountSelectionResult) *AccountSelectionResult {
	if !req.BalanceSamePriority || sticky.Acquired || sticky.WaitPlan == nil {
		return nil
	}
	overflowReq := req
	overflowReq.PreserveStickyBinding = true
	overflowReq.ExcludedIDs = cloneExcludedAccountIDs(req.ExcludedIDs)
	if overflowReq.ExcludedIDs == nil {
		overflowReq.ExcludedIDs = make(map[int64]struct{}, 1)
	}
	overflowReq.ExcludedIDs[sticky.Account.ID] = struct{}{}
	selection, _, _, _, err := s.selectByLoadBalance(ctx, overflowReq)
	if err != nil || selection == nil || selection.Account == nil {
		return nil
	}
	if !selection.Acquired || selection.Account.Priority != sticky.Account.Priority {
		if selection.ReleaseFunc != nil {
			selection.ReleaseFunc()
		}
		return nil
	}
	slog.Info("sticky_overflow_selected", "sticky_account_id", sticky.Account.ID, "account_id", selection.Account.ID)
	selection.PreserveStickyBinding = true
	return selection
}

// BindStickySessionAfterSelection carries the scheduler's ownership decision
// through the handler's final admission step for both HTTP and WebSocket.
func (s *OpenAIGatewayService) BindStickySessionAfterSelection(ctx context.Context, groupID *int64, sessionHash string, selection *AccountSelectionResult) error {
	if selection.PreserveStickyBinding {
		return nil
	}
	return s.BindStickySessionAfterProfitAdmission(ctx, groupID, sessionHash, selection.Account.ID)
}

func (s *OpenAIGatewayService) openAIAccountSchedulerCore() OpenAIAccountScheduler {
	s.openaiSchedulerOnce.Do(func() {
		if s.openaiAccountStats == nil {
			s.openaiAccountStats = newOpenAIAccountRuntimeStats()
		}
		if s.openaiScheduler == nil {
			s.openaiScheduler = newDefaultOpenAIAccountScheduler(s, s.openaiAccountStats)
		}
	})
	return s.openaiScheduler
}

func (s *OpenAIGatewayService) balancedOpenAIAccountLoadPlan(req OpenAIAccountScheduleRequest, plan openAIAccountLoadPlan) openAIAccountLoadPlan {
	pool := plan.allCandidates
	items := make([]accountbalance.Candidate, 0, len(pool))
	byID := make(map[int64]openAIAccountCandidateScore, len(pool))
	now := time.Now()
	for _, candidate := range pool {
		account := candidate.account
		capability := 0
		if req.RequireCompact {
			capability = openAICompactSupportTier(account)
			if capability == 0 && s.schedulerSnapshot == nil {
				continue
			}
		}
		quota, known := openAIAccountQuotaPressure(account, now)
		item := accountbalance.Candidate{
			ID:         account.ID,
			Priority:   account.Priority,
			Capability: capability,
			Available:  !candidate.loadKnown || account.Concurrency <= 0 || candidate.loadInfo.CurrentConcurrency < account.Concurrency,
			QuotaUsed:  quota,
			QuotaKnown: known,
		}
		if account.LastUsedAt != nil {
			item.LastUsedAt = *account.LastUsedAt
		}
		items = append(items, item)
		byID[account.ID] = candidate
	}
	order := s.openaiAccountRotation.Order(items)
	plan.selectionOrder = make([]openAIAccountCandidateScore, 0, len(order))
	for _, id := range order {
		plan.selectionOrder = append(plan.selectionOrder, byID[id])
	}
	plan.topK = len(order)
	return plan
}

func openAIAccountQuotaPressure(account *Account, now time.Time) (float64, bool) {
	updated, err := parseTime(fmt.Sprint(account.Extra["codex_usage_updated_at"]))
	if err != nil || now.Sub(updated) >= time.Duration(accountbalance.QuotaSnapshotMaxAgeSeconds)*time.Second {
		return 0, false
	}
	short, long := openAICanonicalQuotaWindows(account.Extra, now)
	used, known := 0.0, false
	for _, window := range []openAICanonicalQuotaWindow{short, long} {
		if window.hasUsed && !window.reset {
			used = math.Max(used, window.usedPercent)
			known = true
		}
	}
	return used, known
}
