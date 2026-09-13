//go:build unit

package handler

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/LuckyKuang/sub2api-plus/internal/securityaudit"
	"github.com/LuckyKuang/sub2api-plus/internal/server/middleware"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type autoIngressResolverStub struct {
	input    service.AutoRouteRequest
	decision *service.AutoRouteDecision
	err      error
	calls    int
}

func (r *autoIngressResolverStub) Resolve(_ context.Context, _ *service.APIKey, input service.AutoRouteRequest) (*service.AutoRouteDecision, error) {
	r.calls++
	r.input = input
	return r.decision, r.err
}

func TestAutoIngressBindsBeforeDispatchAndPreservesBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	groupID := int64(22)
	key := &service.APIKey{ID: 8, RoutingMode: service.APIKeyRoutingAuto, User: &service.User{ID: 4}}
	bound := *key
	bound.GroupID, bound.Group = &groupID, &service.Group{ID: groupID, Platform: service.PlatformOpenAI}
	resolver := &autoIngressResolverStub{decision: &service.AutoRouteDecision{Key: &bound, Platform: service.PlatformOpenAI, PublicModel: "public", UpstreamModel: "gpt-5"}}
	body := `{"model":"public","input":"audit me","future":{"value":1}}`
	router := gin.New()
	router.POST("/v1/responses", func(c *gin.Context) {
		middleware.BindAPIKeyContext(c, key, nil)
		routeAutoAPIKey(c, resolver)
	}, func(c *gin.Context) {
		got, _ := middleware.GetAPIKeyFromContext(c)
		require.Equal(t, int64(22), *got.GroupID)
		require.Nil(t, key.GroupID)
		data, err := io.ReadAll(c.Request.Body)
		require.NoError(t, err)
		require.Equal(t, body, string(data))
		decision, ok := service.AutoRouteDecisionFromContext(c.Request.Context())
		require.True(t, ok)
		require.Equal(t, "public", decision.PublicModel)
		c.Status(http.StatusNoContent)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)
	require.Equal(t, service.CompositeRouteEndpointResponses, resolver.input.Endpoint)
	require.Equal(t, 1, resolver.calls)
}

func TestAutoIngressRejectsAmbiguousAndUnknownRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tt := range []struct {
		path, body string
		status     int
	}{
		{"/v1/responses", `{"model":"a","model":"b"}`, 400},
		{"/v1/responses", `{"input":"no model"}`, 400},
		{"/v1/custom-voices", `{"model":"grok"}`, 409},
	} {
		t.Run(tt.path+tt.body, func(t *testing.T) {
			r := gin.New()
			resolver := &autoIngressResolverStub{}
			r.POST(tt.path, func(c *gin.Context) {
				middleware.BindAPIKeyContext(c, &service.APIKey{ID: 8, RoutingMode: service.APIKeyRoutingAuto, User: &service.User{ID: 4}}, nil)
				routeAutoAPIKey(c, resolver)
			}, func(c *gin.Context) { t.Error("unsafe request reached dispatch") })
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(tt.body)))
			require.Equal(t, tt.status, w.Code)
			require.Zero(t, resolver.calls)
		})
	}
}

type autoIngressScopedPrompt struct{ handlerPromptEngine }

func (e *autoIngressScopedPrompt) Evaluate(ctx context.Context, req securityaudit.Request) (*securityaudit.PromptDecision, error) {
	if req.GroupID == nil || *req.GroupID != 22 || req.Model != "public-alias" {
		return &securityaudit.PromptDecision{Kind: securityaudit.DecisionAllow, AllowNextStage: true}, nil
	}
	return e.handlerPromptEngine.Evaluate(ctx, req)
}

func TestAutoIngressRealMessagesAuditsSelectedGroupBeforeAdmission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	groupID := int64(22)
	key := &service.APIKey{ID: 8, UserID: 4, RoutingMode: service.APIKeyRoutingAuto, User: &service.User{ID: 4}}
	bound := *key
	bound.GroupID, bound.Group = &groupID, &service.Group{ID: groupID, Platform: service.PlatformComposite}
	decision := &service.AutoRouteDecision{
		Key: &bound, Platform: service.PlatformAnthropic, PublicModel: "public-alias", UpstreamModel: "claude-sonnet-4",
		Composite: &service.CompositeRouteDecision{Matched: true, GroupID: groupID, PublicModel: "public-alias", UpstreamModel: "claude-sonnet-4", TargetPlatform: service.PlatformAnthropic},
	}
	engine := &autoIngressScopedPrompt{handlerPromptEngine: *blockingHandlerPromptEngine()}
	h := &GatewayHandler{securityAuditCoordinator: securityaudit.NewCoordinator(nil, engine)}
	body := `{"model":"public-alias","max_tokens":20,"messages":[{"role":"user","content":"audit original"}],"future":{"value":1}}`
	r := gin.New()
	r.POST("/v1/messages", func(c *gin.Context) {
		middleware.BindAPIKeyContext(c, key, nil)
		routeAutoAPIKey(c, &autoIngressResolverStub{decision: decision})
	}, h.Messages)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(body)))
	require.Equal(t, http.StatusForbidden, w.Code, "a selected-group policy must block before the absent admission service is reached")
	evaluated, _, requests := engine.snapshot()
	require.Equal(t, 1, evaluated)
	require.Len(t, requests, 1)
	require.JSONEq(t, body, string(requests[0].Body))
	require.Nil(t, key.GroupID)
}
