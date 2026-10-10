package service

import (
	"strings"
	"time"
)

// 探测只能替换或解除余额、订阅窗口能证明的冷却；组织限额等独立原因保留到期。
func clineWalletProbeMayReplaceCooldown(account *Account, scope string, now time.Time) bool {
	resetAt := account.modelRateLimitResetAt(scope)
	if resetAt == nil || !now.Before(*resetAt) {
		return true
	}
	reason := account.modelRateLimitReason(scope)
	switch scope {
	case clinePassRateLimitKey:
		return strings.HasPrefix(reason, clinePassLimitReason) || reason == clinePassNoSubscriptionReason
	case clineCreditsRateLimitKey:
		return strings.HasPrefix(reason, clineCreditsReason) || strings.HasPrefix(reason, clineCreditsLowReason)
	}
	return false
}
