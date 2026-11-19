package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/tidwall/gjson"
	"log/slog"
	"math"
	"net/http"
	"sort"
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

var ErrOpenAIAccountQueueFull = errors.New("account wait queue is full")

type openAIQueueReselectKey struct{}

// WithOpenAIQueueReselection scopes the extra capacity probe to one selection.
func WithOpenAIQueueReselection(ctx context.Context, id int64) context.Context {
	if id == 0 {
		return ctx
	}
	return context.WithValue(ctx, openAIQueueReselectKey{}, id)
}

func queueReselectAccount(ctx context.Context) int64 {
	id, _ := ctx.Value(openAIQueueReselectKey{}).(int64)
	return id
}

// Borrow an immediately acquired slot, retaining the session's original owner.
func (s *defaultOpenAIAccountScheduler) overflowFullStickyAccount(ctx context.Context, req OpenAIAccountScheduleRequest, sticky *AccountSelectionResult) *AccountSelectionResult {
	if !req.BalanceSamePriority || sticky.Acquired || sticky.WaitPlan == nil {
		return nil
	}
	if request, ok := ctx.Value(openAIClientRestrictionKey{}).(openAIClientRestrictionRequest); ok && !openAISessionHistoryComplete(request.body) {
		return nil
	}
	overflowReq := req
	overflowReq.overflowFrom = sticky.Account
	overflowReq.SubscriptionPriority = false
	overflowReq.PreserveStickyBinding = true
	overflowReq.ExcludedIDs = cloneExcludedAccountIDs(req.ExcludedIDs)
	if overflowReq.ExcludedIDs == nil {
		overflowReq.ExcludedIDs = make(map[int64]struct{}, 1)
	}
	overflowReq.ExcludedIDs[sticky.Account.ID] = struct{}{}
	if id := queueReselectAccount(ctx); id != 0 {
		overflowReq.ExcludedIDs[id] = struct{}{}
	}
	selection, _, _, _, err := s.selectByLoadBalance(ctx, overflowReq)
	if err != nil || selection == nil || selection.Account == nil {
		return nil
	}
	if !selection.Acquired || selection.Account.Priority < sticky.Account.Priority {
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
	if req.overflowFrom != nil {
		sort.SliceStable(plan.selectionOrder, func(i, j int) bool {
			return plan.selectionOrder[i].account.Priority < plan.selectionOrder[j].account.Priority
		})
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

// Canonical routes keep their generation and owner during temporary overflow.
// Only the request's execution account changes; complete-history validation at
// the forwarding boundary still applies before sending to a different account.
func (a *openAISessionRouteAttempt) expectedOwnerAccountID() int64 {
	if a.preserveOwner {
		return a.route.OwnerAccountID
	}
	return a.accountID
}

func (s *OpenAIGatewayService) overflowCanonicalOpenAIAccount(ctx context.Context, req OpenAIAccountScheduleRequest, sticky *AccountSelectionResult) *AccountSelectionResult {
	if sticky.Acquired || sticky.WaitPlan == nil {
		return nil
	}
	guardianParentID := int64(0)
	if strings.TrimSpace(req.PreviousResponseID) == "" {
		guardianParentID = s.resolveOpenAIGuardianParentAccountID(ctx, req.GroupID)
	}
	req.BalanceSamePriority = useBalancedOpenAISelection(ctx, NormalizeOpenAICompatiblePlatform(req.Platform), req.PreviousResponseID, req.PreviousResponseCanMove, guardianParentID, preserveOpenAIGuardianParentBinding(ctx, req.SessionHash))
	if !req.BalanceSamePriority {
		return nil
	}
	req.RequirePrivacySet = s.openAIGroupRequiresPrivacySet(ctx, req.GroupID)
	s.openAIAccountSchedulerCore()
	canonical := &defaultOpenAIAccountScheduler{service: s, stats: s.openaiAccountStats}
	overflow := canonical.overflowFullStickyAccount(ctx, req, sticky)
	if overflow != nil {
		if err := s.checkOpenAIClientRestriction(ctx, overflow.Account); err != nil {
			overflow.ReleaseFunc()
			return nil
		}
	}
	return overflow
}

// Explicit provider-wide and end-user budgets survive a change of account.
func openAISharedRateLimit(headers http.Header, body []byte) bool {
	sharedScope := func(scope string) bool {
		switch strings.ToLower(strings.TrimSpace(scope)) {
		case "global", "shared", "user", "organization", "project", "provider", "group":
			return true
		}
		return false
	}
	if sharedScope(headers.Get("X-RateLimit-Scope")) {
		return true
	}
	root := gjson.ParseBytes(body)
	for _, detail := range []gjson.Result{root.Get("response.error"), root.Get("error"), root} {
		if sharedScope(detail.Get("scope").String()) || sharedScope(detail.Get("metadata.scope").String()) {
			return true
		}
		switch detail.Get("code").String() {
		case "user_rate_limit_exceeded", "organization_rate_limit_exceeded", "project_rate_limit_exceeded", "global_rate_limit_exceeded":
			return true
		}
	}
	return false
}

func (a *Account) PreferNextAccountOn429(e *UpstreamFailoverError) bool {
	if a == nil || a.Platform != PlatformOpenAI || e == nil || e.StatusCode != http.StatusTooManyRequests ||
		!e.ShouldRetryNextAccount() || !e.ShouldReportAccountScheduleFailure() {
		return false
	}
	if a.IsPoolMode() {
		_, retryCountSet := a.Credentials["pool_mode_retry_count"]
		_, retryCodesSet := a.Credentials["pool_mode_retry_status_codes"]
		if retryCountSet || retryCodesSet {
			return false
		}
	}
	return true
}
