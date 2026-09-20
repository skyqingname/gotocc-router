package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/ctxkey"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/googleapi"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
)

// ContextKey 定义上下文键类型
type ContextKey string

const (
	ContextKeyUser ContextKey = "user"
	ContextKeyUserRole ContextKey = "user_role"
	ContextKeyAPIKey ContextKey = "api_key"
	ContextKeySubscription ContextKey = "subscription"
	ContextKeyForcePlatform ContextKey = "force_platform"
	// Only for ingress diagnostics, never proof of successful authentication.
	ContextKeyOpsFallbackAPIKey ContextKey = "ops_fallback_api_key"
)

type groupRateAcceptedAtKey struct{}

// CaptureGroupRateRequestTime runs before authentication. Client headers cannot
// provide or override this trusted time. Configuration is read only after auth.
func CaptureGroupRateRequestTime() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), groupRateAcceptedAtKey{}, time.Now())
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func ForcePlatform(platform string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), ctxkey.ForcePlatform, platform)
		c.Request = c.Request.WithContext(ctx)
		c.Set(string(ContextKeyForcePlatform), platform)
		c.Next()
	}
}

func HasForcePlatform(c *gin.Context) bool {
	_, exists := c.Get(string(ContextKeyForcePlatform))
	return exists
}

func GetForcePlatformFromContext(c *gin.Context) (string, bool) {
	value, exists := c.Get(string(ContextKeyForcePlatform))
	if !exists { return "", false }
	platform, ok := value.(string)
	return platform, ok
}

type ErrorResponse struct {
	Code string `json:"code"`
	Message string `json:"message"`
}

func NewErrorResponse(code, message string) ErrorResponse { return ErrorResponse{Code: code, Message: message} }

func AbortWithError(c *gin.Context, statusCode int, code, message string) {
	c.JSON(statusCode, NewErrorResponse(code, message))
	c.Abort()
}

func abortWithOpenAIQuotaError(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, gin.H{"error": gin.H{
		"message": message, "type": "insufficient_quota", "param": nil, "code": "insufficient_quota",
	}})
	c.Abort()
}

type GatewayErrorWriter func(c *gin.Context, status int, message string)

func AnthropicErrorWriter(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"type": "error", "error": gin.H{"type": "permission_error", "message": message}})
}

func GoogleErrorWriter(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": gin.H{"code": status, "message": message, "status": googleapi.HTTPStatusToGoogleStatus(status)}})
}

// ApplyGroupRateSchedule is also available to deferred auto-route admission:
// call it after the final billing group has been assigned, before scheduling.
// This function performs no billing, quota, concurrency, or upstream write.
func ApplyGroupRateSchedule(c *gin.Context, settings *service.SettingService) bool {
	key, ok := GetAPIKeyFromContext(c)
	if !ok || key == nil || key.Group == nil || settings == nil { return true }
	if key.Group.RateScheduleRuntime != nil { return true }
	acceptedAt, _ := c.Request.Context().Value(groupRateAcceptedAtKey{}).(time.Time)
	if acceptedAt.IsZero() { acceptedAt = time.Now() }
	live := strings.EqualFold(c.GetHeader("Upgrade"), "websocket")
	resolved, err := settings.PrepareGroupRateSchedule(c.Request.Context(), key, acceptedAt, live)
	if err != nil {
		slog.Error("group_rate_schedule.load_failed", "request_id", c.GetString("request_id"),
			"endpoint", c.FullPath(), "protocol", "http", "stage", "group_configuration",
			"error_code", "group_rate_schedule_unavailable")
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{
			"type": "server_error", "code": "group_rate_schedule_unavailable", "message": "Group pricing configuration is temporarily unavailable",
		}})
		return false
	}
	if resolved != key {
		c.Set(string(ContextKeyAPIKey), resolved)
		current, _ := c.Request.Context().Value(ctxkey.Group).(*service.Group)
		if current == nil || current.ID == resolved.Group.ID {
			c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ctxkey.Group, resolved.Group))
		}
	}
	return true
}

func RequireGroupAssignment(settingService *service.SettingService, writeError GatewayErrorWriter) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey, ok := GetAPIKeyFromContext(c)
		if !ok || apiKey.GroupID != nil {
			if ok && !ApplyGroupRateSchedule(c, settingService) { return }
			c.Next()
			return
		}
		if apiKey.IsAutoRouting() {
			if service.CanDeferAutoRoute(c.Request.Context(), apiKey.ID) {
				c.Next()
				return
			}
			writeError(c, http.StatusForbidden, "automatic routing did not resolve a group for this request")
			c.Abort()
			return
		}
		if settingService.IsUngroupedKeySchedulingAllowed(c.Request.Context()) {
			c.Next()
			return
		}
		service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonAPIKeyGroupUnassigned)
		MarkIngressRejected(c, IngressRejectGroupUnassigned)
		writeError(c, http.StatusForbidden, "API Key is not assigned to any group and cannot be used. Please contact the administrator to assign it to a group.")
		c.Abort()
	}
}
