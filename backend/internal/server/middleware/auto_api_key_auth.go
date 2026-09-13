package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/LuckyKuang/sub2api-plus/internal/config"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/ctxkey"
	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/ip"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/logger"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func handleAutoAPIKeyAuth(c *gin.Context, keys *service.APIKeyService, cfg *config.Config, key *service.APIKey, google bool) bool {
	if key.EffectiveRoutingMode() == service.APIKeyRoutingFixed {
		return false
	}
	if !key.IsAutoRouting() || key.GroupID != nil || key.Group != nil {
		writeAutoAuthError(c, http.StatusUnauthorized, "INVALID_API_KEY", "invalid API key routing configuration", google)
		return true
	}
	if cfg != nil && cfg.RunMode == config.RunModeSimple {
		writeAutoAuthError(c, http.StatusForbidden, "AUTO_ROUTING_UNSUPPORTED_RUN_MODE", "automatic routing requires standard run mode", google)
		return true
	}
	if err := keys.ValidateTeamKeyLifecycle(key); err != nil {
		WriteAutoRoutingError(c, err)
		return true
	}
	if !isAPIKeyNonConsumingRequest(c.Request.Method, c.Request.URL.Path) {
		if err := keys.CheckTeamMemberLimits(key); err != nil {
			WriteAutoRoutingError(c, err)
			return true
		}
		if key.IsExpired() || key.Status == service.StatusAPIKeyExpired {
			writeAutoAuthError(c, http.StatusForbidden, "API_KEY_EXPIRED", "API key has expired", google)
			return true
		}
		if key.IsQuotaExhausted() || key.Status == service.StatusAPIKeyQuotaExhausted {
			writeAutoAuthError(c, http.StatusTooManyRequests, "API_KEY_QUOTA_EXHAUSTED", "API key quota exhausted", google)
			return true
		}
	}
	ctx := service.WithAutoRouteLock(c.Request.Context(), key.ID, 0)
	trustForwarded := false
	if cfg != nil {
		trustForwarded = cfg.TrustForwardedIPForAPIKeyACL()
	}
	ctx = service.WithAutoRouteClientIP(ctx, ip.GetSecurityClientIP(c, trustForwarded))
	c.Request = c.Request.WithContext(ctx)
	BindAPIKeyContext(c, key, nil)
	c.Next()
	return true
}

// BindAPIKeyContext publishes one coherent request-local identity and group.
func BindAPIKeyContext(c *gin.Context, key *service.APIKey, subscription *service.UserSubscription) {
	if key.IsAutoRouting() {
		if route, ok := service.AutoRouteDecisionFromContext(c.Request.Context()); ok && route.Key != nil && route.Key.ID == key.ID && key.GroupID != nil {
			updated := *route
			updated.Key = key
			c.Request = c.Request.WithContext(updated.RequestContext(c.Request.Context()))
		}
		id := int64(0)
		if subscription != nil {
			id = subscription.ID
		}
		c.Request = c.Request.WithContext(service.WithAutoRouteSubscription(c.Request.Context(), id))
	}
	c.Set(string(ContextKeyAPIKey), key)
	c.Set(string(ContextKeyUser), AuthSubject{UserID: key.User.ID, Concurrency: key.User.Concurrency})
	c.Set(string(ContextKeyUserRole), key.User.Role)
	if subscription == nil {
		c.Set(string(ContextKeySubscription), nil)
	} else {
		c.Set(string(ContextKeySubscription), subscription)
	}
	c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ctxkey.UserID, key.User.ID))
	setGroupContext(c, key.Group)
	SetOpsFallbackAPIKey(c, key)
}

func WriteAutoRoutingError(c *gin.Context, err error) {
	status, detail := infraerrors.ToHTTP(err)
	if isSubscriptionUsageLimitError(err) {
		status = http.StatusTooManyRequests
		detail.Reason = subscriptionQuotaResponseCode(err)
		applySubscriptionQuotaResetHeaders(c, err)
	}
	if status >= 500 {
		logger.FromContext(c.Request.Context()).Error("automatic routing failed",
			zap.String("stage", "auto_route"), zap.String("error_code", detail.Reason),
			zap.String("endpoint", c.FullPath()))
	}
	google := strings.Contains(c.FullPath(), "/v1beta/") || strings.HasSuffix(c.FullPath(), "/v1beta/models")
	writeAutoAuthError(c, status, detail.Reason, detail.Message, google)
}

func writeAutoAuthError(c *gin.Context, status int, reason, message string, google bool) {
	if google {
		abortWithGoogleError(c, status, message)
		return
	}
	if reason == "" {
		reason = "api_error"
	}
	AbortWithError(c, status, reason, message)
}
