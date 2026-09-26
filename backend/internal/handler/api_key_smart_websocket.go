package handler

import (
	"context"
	"errors"
	pkghttputil "github.com/LuckyKuang/sub2api-plus/internal/pkg/httputil"
	"github.com/LuckyKuang/sub2api-plus/internal/server/middleware"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

type autoWSTurnState struct {
	turn         int
	key          *service.APIKey
	subscription *service.UserSubscription
	subject      middleware.AuthSubject
	route        *service.AutoRouteDecision
	ctx          context.Context
}

func (h *OpenAIGatewayHandler) resolveAutoWSTurn(c *gin.Context, key *service.APIKey, model string, payload []byte, groupID *int64, accountID int64) (*service.AutoRouteDecision, error) {
	if h.autoGroupResolver == nil {
		return nil, service.ErrAutoRouteUnavailable
	}
	if gjson.GetBytes(payload, "model").Exists() {
		var err error
		model, err = pkghttputil.ExtractGatewayRoutingModel("application/json", payload, false)
		if err != nil {
			return nil, service.ErrAutoRouteContext
		}
	}
	input := service.AutoRouteRequest{
		Model: model, Endpoint: service.AutoRouteEndpointResponsesWS,
		RequiredGroupID: groupID, RequiredAccountID: accountID,
		ImageGeneration: service.IsExplicitImageGenerationIntent("/v1/responses", model, payload),
	}
	input.ForcePlatform, _ = middleware.GetForcePlatformFromContext(c)
	var affinity *service.AutoResponseAffinity
	previous, parseErr := pkghttputil.ExtractGatewayPreviousResponseID(payload)
	if parseErr != nil {
		return nil, service.ErrAutoRouteContext
	}
	if previous != "" {
		var err error
		affinity, err = h.gatewayService.LookupAutoResponseAffinity(c.Request.Context(), key, previous)
		if err != nil {
			return nil, err
		}
		if (groupID != nil && affinity.GroupID != *groupID) || (accountID > 0 && affinity.AccountID != accountID) {
			return nil, service.ErrAutoRouteContext
		}
		input.RequiredGroupID, input.RequiredAccountID = &affinity.GroupID, affinity.AccountID
	}
	decision, err := h.autoGroupResolver.Resolve(c.Request.Context(), key, input)
	if err != nil {
		return nil, err
	}
	if decision.Key.User.ID != key.User.ID || (affinity != nil && decision.Key.User.ID != affinity.BillingUserID) {
		return nil, service.ErrAutoRouteContext
	}
	return decision, nil
}

func autoWSCloseError(err error) error {
	if errors.Is(err, service.ErrAutoRouteUnavailable) {
		return service.NewOpenAIWSClientCloseError(coderws.StatusTryAgainLater, "automatic routing is temporarily unavailable", err)
	}
	return service.NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, "automatic routing context is no longer valid; reconnect", err)
}

func rewriteAutoWSModel(payload []byte, decision *service.AutoRouteDecision) ([]byte, error) {
	if decision == nil || decision.UpstreamModel == decision.PublicModel {
		return payload, nil
	}
	return pkghttputil.RewriteGatewayRoutingModel("application/json", payload, false, decision.UpstreamModel)
}
