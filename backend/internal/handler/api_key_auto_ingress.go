package handler

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"

	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
	pkghttputil "github.com/LuckyKuang/sub2api-plus/internal/pkg/httputil"
	"github.com/LuckyKuang/sub2api-plus/internal/server/middleware"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

type autoIngressResolver interface {
	Resolve(context.Context, *service.APIKey, service.AutoRouteRequest) (*service.AutoRouteDecision, error)
}

type autoHTTPRouteResolver struct {
	groups *service.AutoGroupResolver
	openAI *service.OpenAIGatewayService
}

func (r autoHTTPRouteResolver) Resolve(ctx context.Context, key *service.APIKey, input service.AutoRouteRequest) (*service.AutoRouteDecision, error) {
	if r.groups == nil {
		return nil, service.ErrAutoRouteUnavailable
	}
	return r.groups.Resolve(ctx, key, input)
}

func (r autoHTTPRouteResolver) LookupAutoResponseAffinity(ctx context.Context, key *service.APIKey, id string) (*service.AutoResponseAffinity, error) {
	return r.openAI.LookupAutoResponseAffinity(ctx, key, id)
}

func (h *GatewayHandler) RouteAutoAPIKey(c *gin.Context) {
	if key, ok := middleware.GetAPIKeyFromContext(c); ok && key != nil && key.IsAutoRouting() {
		if h.restoreAutoResource(c, key) {
			return
		}
		if _, deferred := autoRouteOperation(c); deferred == service.AutoRouteDeferredModels {
			serveAutoModels(c, h.autoGroupResolver)
			return
		}
	}
	routeAutoAPIKey(c, autoHTTPRouteResolver{groups: h.autoGroupResolver, openAI: h.openAIGatewayService})
}

func routeAutoAPIKey(c *gin.Context, resolver autoIngressResolver) {
	key, ok := middleware.GetAPIKeyFromContext(c)
	if !ok || key == nil || !key.IsAutoRouting() {
		c.Next()
		return
	}
	endpoint, deferred := autoRouteOperation(c)
	if deferred != "" {
		c.Request = c.Request.WithContext(service.WithAutoRouteDeferred(c.Request.Context(), key.ID, deferred))
		c.Next()
		return
	}
	if endpoint == "" {
		middleware.WriteAutoRoutingError(c, service.ErrAutoRouteContext)
		return
	}
	body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil {
		status := http.StatusBadRequest
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			status = http.StatusRequestEntityTooLarge
		}
		middleware.AbortWithError(c, status, "invalid_request_error", "Failed to read request body")
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	model := ""
	switch endpoint {
	case service.AutoRouteEndpointWebSearch:
		model = resolveGrokStandaloneSearchModel()
		if !gjson.ValidBytes(body) {
			err = errors.New("invalid search JSON")
		}
	case service.AutoRouteEndpointVoice:
		model = "grok-4.5"
	case service.CompositeRouteEndpointGemini:
		model = strings.TrimPrefix(c.Param("modelAction"), "/")
		if index := strings.LastIndex(model, ":"); index >= 0 {
			model = model[:index]
		}
		if model == "" || len(model) > 1024 || !gjson.ValidBytes(body) {
			err = errors.New("invalid Gemini model or JSON")
		}
	default:
		model, err = pkghttputil.ExtractGatewayRoutingModel(c.GetHeader("Content-Type"), body, endpoint == service.AutoRouteEndpointLive)
	}
	if err != nil {
		middleware.WriteAutoRoutingError(c, infraerrors.BadRequest("invalid_request_error", "Request must contain one unambiguous model string"))
		return
	}
	var affinity *service.AutoResponseAffinity
	previousID := ""
	if endpoint == service.CompositeRouteEndpointResponses {
		previousID, err = pkghttputil.ExtractGatewayPreviousResponseID(body)
		if err != nil {
			middleware.AbortWithError(c, http.StatusBadRequest, "invalid_request_error", "Invalid previous_response_id")
			return
		}
	}
	if previousID != "" {
		lookup, ok := resolver.(interface {
			LookupAutoResponseAffinity(context.Context, *service.APIKey, string) (*service.AutoResponseAffinity, error)
		})
		if !ok {
			middleware.WriteAutoRoutingError(c, service.ErrAutoRouteContext)
			return
		}
		affinity, err = lookup.LookupAutoResponseAffinity(c.Request.Context(), key, previousID)
		if err != nil {
			middleware.WriteAutoRoutingError(c, err)
			return
		}
	}
	SetClaudeCodeClientContext(c, body, nil)
	forced, _ := middleware.GetForcePlatformFromContext(c)
	input := service.AutoRouteRequest{
		Model: model, Endpoint: endpoint, ForcePlatform: forced,
		ClaudeCodeClient: service.IsClaudeCodeClient(c.Request.Context()),
		Provider:         gjson.GetBytes(body, "provider").String(),
		ImageGeneration:  service.IsExplicitImageGenerationIntent(c.Request.URL.Path, model, body),
	}
	if affinity != nil {
		input.RequiredGroupID = &affinity.GroupID
		input.RequiredAccountID = affinity.AccountID
	}
	if resolver == nil {
		middleware.WriteAutoRoutingError(c, service.ErrAutoRouteUnavailable)
		return
	}
	decision, err := resolver.Resolve(c.Request.Context(), key, input)
	if err != nil {
		middleware.WriteAutoRoutingError(c, err)
		return
	}
	if decision == nil || decision.Key == nil || decision.Key.GroupID == nil || decision.Key.Group == nil || decision.Key.User == nil {
		middleware.WriteAutoRoutingError(c, service.ErrAutoRouteUnavailable)
		return
	}
	if affinity != nil && (decision.Key.User.ID != affinity.BillingUserID || *decision.Key.GroupID != affinity.GroupID) {
		middleware.WriteAutoRoutingError(c, service.ErrAutoRouteContext)
		return
	}
	c.Request = c.Request.WithContext(decision.RequestContext(c.Request.Context()))
	middleware.BindAPIKeyContext(c, decision.Key, nil)
	c.Next()
}

func autoRouteOperation(c *gin.Context) (string, service.AutoRouteDeferredKind) {
	path := c.FullPath()
	for _, prefix := range []string{"/antigravity", "/backend-api/codex", "/openai"} {
		path = strings.TrimPrefix(path, prefix)
	}
	if strings.HasPrefix(path, "/v1beta/models") {
		if c.Request.Method == http.MethodGet {
			return "", service.AutoRouteDeferredModels
		}
		if c.Request.Method == http.MethodPost && strings.HasSuffix(path, "/*modelAction") {
			return service.CompositeRouteEndpointGemini, ""
		}
	}
	path = strings.TrimPrefix(path, "/v1/")
	path = strings.TrimPrefix(path, "/")
	if path == "images/batches/models" && c.Request.Method == http.MethodGet {
		return "", service.AutoRouteDeferredModels
	}
	if path == "images/batches" || strings.HasPrefix(path, "images/batches/:id") {
		return "", service.AutoRouteDeferredResource
	}
	if c.Request.Method == http.MethodGet {
		switch path {
		case "models":
			return "", service.AutoRouteDeferredModels
		case "responses":
			return "", service.AutoRouteDeferredWS
		case "usage", "images/tasks", "images/tasks/:task_id", "images/tasks/:task_id/download", "images/objects/:object_id/url":
			return "", service.AutoRouteDeferredResource
		}
	}
	if c.Request.Method == http.MethodDelete && path == "images/tasks/:task_id" {
		return "", service.AutoRouteDeferredResource
	}
	if c.Request.Method != http.MethodPost {
		return "", ""
	}
	switch path {
	case "messages":
		return service.CompositeRouteEndpointMessages, ""
	case "messages/count_tokens":
		return service.CompositeRouteEndpointCountTokens, ""
	case "responses", "responses/*subpath":
		return service.CompositeRouteEndpointResponses, ""
	case "chat/completions":
		return service.CompositeRouteEndpointChatCompletions, ""
	case "embeddings":
		return service.CompositeRouteEndpointEmbeddings, ""
	case "images/generations", "images/edits", "images/generations/async", "images/edits/async":
		return service.CompositeRouteEndpointImages, ""
	case "live", "realtime/calls":
		return service.AutoRouteEndpointLive, ""
	case "videos", "videos/generations":
		return service.AutoRouteEndpointVideos, ""
	case "alpha/search":
		return service.AutoRouteEndpointAlphaSearch, ""
	case "web_search", "x_search":
		return service.AutoRouteEndpointWebSearch, ""
	case "tts", "stt":
		return service.AutoRouteEndpointVoice, ""
	default:
		return "", ""
	}
}
