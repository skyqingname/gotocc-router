package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
)

func isAsyncImageReadRequest(method, path string) bool {
	if method != http.MethodGet {
		return false
	}
	path = strings.TrimRight(path, "/")
	if isAsyncImageTaskManagement(method, path) {
		return true
	}
	for _, root := range []string{"/v1/images/objects/", "/images/objects/"} {
		remainder := strings.TrimPrefix(path, root)
		if remainder == path {
			continue
		}
		id, suffix, ok := strings.Cut(remainder, "/")
		if ok && id != "" && suffix == "url" {
			return true
		}
	}
	return false
}

func isBatchImageManagementRequest(method, path string) bool {
	path = strings.TrimRight(path, "/")
	for _, root := range []string{"/v1/images/batches", "/images/batches"} {
		if method == http.MethodGet && (path == root || strings.HasPrefix(path, root+"/")) {
			return true
		}
		if method == http.MethodDelete && strings.HasPrefix(path, root+"/") {
			return true
		}
		if method == http.MethodPost && strings.HasPrefix(path, root+"/") && strings.HasSuffix(path, "/cancel") {
			return true
		}
	}
	return false
}

func isAPIKeyNonConsumingRequest(method, path string) bool {
	path = strings.TrimRight(path, "/")
	if isVideoTaskReadRequest(method, path) {
		return true
	}
	if method == http.MethodGet {
		if path == "/v1/usage" || path == "/antigravity/v1/usage" || path == "/v1/sub2api/billing" || path == "/backend-api/wham/usage" {
			return true
		}
		if strings.HasSuffix(path, "/models") || isAsyncImageTaskManagement(method, path) || isAsyncImageReadRequest(method, path) || isBatchImageManagementRequest(method, path) {
			return true
		}
	}
	if isAsyncImageTaskManagement(method, path) || isBatchImageManagementRequest(method, path) {
		return true
	}
	return method == http.MethodPost && strings.HasSuffix(path, "/messages/count_tokens")
}

func isVideoTaskReadRequest(method, path string) bool {
	if method != http.MethodGet {
		return false
	}
	path = strings.TrimRight(path, "/")
	for _, root := range []string{"/v1/videos/generations/", "/v1/videos/edits/", "/v1/videos/extensions/", "/v1/videos/", "/v1/video/generations/"} {
		remainder := strings.TrimPrefix(path, root)
		if remainder == path {
			continue
		}
		id, suffix, hasSuffix := strings.Cut(remainder, "/")
		return id != "" && (!hasSuffix || suffix == "content")
	}
	return false
}

func abortTeamAPIKeyError(c *gin.Context, err error) bool {
	switch {
	case errors.Is(err, service.ErrTeamFundingSource):
		AbortWithError(c, http.StatusForbidden, "TEAM_FUNDING_SOURCE_MISMATCH", "团队当前负责人的额度来源与团队不一致")
	case errors.Is(err, service.ErrTeamBalanceInsufficient):
		AbortWithError(c, http.StatusForbidden, "TEAM_BALANCE_INSUFFICIENT", "团队公共余额不足")
	case errors.Is(err, service.ErrTeamMemberDailyExceeded):
		AbortWithError(c, http.StatusTooManyRequests, "TEAM_MEMBER_DAILY_LIMIT_EXCEEDED", "团队成员日限额已用完")
	case errors.Is(err, service.ErrTeamMemberWeeklyExceeded):
		AbortWithError(c, http.StatusTooManyRequests, "TEAM_MEMBER_WEEKLY_LIMIT_EXCEEDED", "团队成员周限额已用完")
	case errors.Is(err, service.ErrTeamMemberMonthlyExceeded):
		AbortWithError(c, http.StatusTooManyRequests, "TEAM_MEMBER_MONTHLY_LIMIT_EXCEEDED", "团队成员月限额已用完")
	case errors.Is(err, service.ErrTeamFeatureDisabled):
		AbortWithError(c, http.StatusForbidden, "TEAM_FEATURE_DISABLED", "团队功能未启用")
	case errors.Is(err, service.ErrTeamSuspended):
		AbortWithError(c, http.StatusForbidden, "TEAM_SUSPENDED", "团队已暂停")
	case errors.Is(err, service.ErrTeamMembershipRequired):
		AbortWithError(c, http.StatusForbidden, "TEAM_MEMBERSHIP_REQUIRED", "团队成员关系已失效")
	case errors.Is(err, service.ErrTeamActorInactive):
		AbortWithError(c, http.StatusForbidden, "TEAM_ACTOR_INACTIVE", "团队密钥所属成员已停用")
	case errors.Is(err, service.ErrTeamBillingOwnerInactive):
		AbortWithError(c, http.StatusForbidden, "TEAM_BILLING_OWNER_INACTIVE", "团队付款所有者已停用")
	default:
		return false
	}
	return true
}
