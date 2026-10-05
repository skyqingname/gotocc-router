package service

import (
	"context"
	"fmt"
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

func (s *defaultOpenAIAccountScheduler) selectBalancedOpenAIAccount(ctx context.Context, req OpenAIAccountScheduleRequest) (*AccountSelectionResult, OpenAIAccountScheduleDecision, error) {
	selection, count, topK, skew, err := s.selectByLoadBalance(ctx, req)
	decision := OpenAIAccountScheduleDecision{
		Layer:          openAIAccountScheduleLayerLoadBalance,
		CandidateCount: count,
		TopK:           topK,
		LoadSkew:       skew,
	}
	if selection != nil && selection.Account != nil {
		decision.SelectedAccountID = selection.Account.ID
		decision.SelectedAccountType = selection.Account.Type
		decision.StickyPreviousHit = req.StickyPreviousAccountID > 0 && req.StickyPreviousAccountID == selection.Account.ID
	}
	return selection, decision, err
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
		items = append(items, accountbalance.Candidate{
			ID:         account.ID,
			Priority:   account.Priority,
			Capability: capability,
			Available:  !candidate.loadKnown || account.Concurrency <= 0 || candidate.loadInfo.CurrentConcurrency < account.Concurrency,
			QuotaUsed:  quota,
			QuotaKnown: known,
		})
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
