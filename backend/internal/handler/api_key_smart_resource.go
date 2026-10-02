package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/LuckyKuang/sub2api-plus/internal/server/middleware"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *GatewayHandler) restoreAutoResource(c *gin.Context, key *service.APIKey) bool {
	if c.Request.Method != http.MethodGet || (c.Param("call_id") == "" && c.Param("request_id") == "") {
		return false
	}
	if h.autoGroupResolver == nil || h.openAIGatewayService == nil {
		middleware.WriteAutoRoutingError(c, service.ErrAutoRouteUnavailable)
		return true
	}
	var groupID int64
	var subscriptionID int64
	platform := service.PlatformOpenAI
	if id := c.Param("call_id"); id != "" {
		record, err := h.openAIGatewayService.GetLiveCallForAutoKey(c.Request.Context(), id, key)
		if err != nil {
			if errors.Is(err, service.ErrLiveCallNotFound) || errors.Is(err, service.ErrLiveIdentityMismatch) {
				middleware.WriteAutoRoutingError(c, service.ErrAutoRouteContext)
			} else {
				middleware.WriteAutoRoutingError(c, service.ErrAutoRouteUnavailable)
			}
			return true
		}
		groupID = record.GroupID
		subscriptionID = record.SubscriptionID
	} else if strings.Contains(c.FullPath(), "/videos/") {
		var task *service.OpenAIVideoTask
		var account *service.Account
		var err error
		if h.openAIGatewayService.OpenAIVideoTaskStorageAvailable() {
			task, account, err = h.openAIGatewayService.GetOpenAIVideoTaskForAPIKey(c.Request.Context(), c.Param("request_id"), key.ID)
			if err != nil && !errors.Is(err, service.ErrOpenAIVideoTaskNotFound) {
				middleware.WriteAutoRoutingError(c, service.ErrAutoRouteUnavailable)
				return true
			}
		}
		if task != nil && account != nil {
			if task.ActorUserID != key.UserID || task.BillingUserID != key.User.ID || valueOrZeroInt64(task.TeamID) != valueOrZeroInt64(key.TeamID) {
				middleware.WriteAutoRoutingError(c, service.ErrAutoRouteNoAccess)
				return true
			}
			groupID, platform, subscriptionID = task.GroupID, account.Platform, valueOrZeroInt64(task.SubscriptionID)
		} else {
			binding, err := h.openAIGatewayService.LookupAutoResponseAffinity(c.Request.Context(), key, "video:"+c.Param("request_id"))
			if err != nil {
				middleware.WriteAutoRoutingError(c, err)
				return true
			}
			if binding.Platform != service.PlatformGrok {
				middleware.WriteAutoRoutingError(c, service.ErrAutoRouteContext)
				return true
			}
			groupID, platform, subscriptionID = binding.GroupID, binding.Platform, binding.SubscriptionID
		}
	} else {
		return false
	}
	decision, err := h.autoGroupResolver.RestoreGroup(c.Request.Context(), key, groupID, platform)
	if err != nil {
		middleware.WriteAutoRoutingError(c, err)
		return true
	}
	if decision.Key.User.ID != key.User.ID {
		middleware.WriteAutoRoutingError(c, service.ErrAutoRouteNoAccess)
		return true
	}
	if c.Param("call_id") != "" {
		decision.Endpoint = service.AutoRouteEndpointLive
	}
	subscription, err := h.autoGroupResolver.RestoreResourceSubscription(c.Request.Context(), decision.Key, subscriptionID)
	if err != nil {
		middleware.WriteAutoRoutingError(c, err)
		return true
	}
	c.Request = c.Request.WithContext(service.WithResolvedTargetPlatform(decision.RequestContext(c.Request.Context()), platform))
	middleware.BindAPIKeyContext(c, decision.Key, subscription)
	c.Next()
	return true
}
