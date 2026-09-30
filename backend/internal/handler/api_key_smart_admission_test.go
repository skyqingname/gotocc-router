//go:build unit

package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/LuckyKuang/sub2api-plus/internal/server/middleware"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type autoAdmissionStub struct {
	result *service.AutoRouteAdmission
	err    error
}

func (s autoAdmissionStub) Admit(context.Context, *service.APIKey) (*service.AutoRouteAdmission, error) {
	return s.result, s.err
}

func TestAutoHTTPAdmissionRefreshesSubscriptionAndPreservesOriginalKey(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	groupID := int64(12)
	key := &service.APIKey{ID: 1, RoutingMode: service.APIKeyRoutingAuto, GroupID: &groupID, Group: &service.Group{ID: groupID}, User: &service.User{ID: 3}}
	original := key
	fresh := *key
	fresh.User = &service.User{ID: 3, Concurrency: 7}
	sub := &service.UserSubscription{ID: 8, GroupID: groupID, UserID: 3}
	require.True(t, admitAutoHTTPRoute(c, autoAdmissionStub{result: &service.AutoRouteAdmission{Key: &fresh, Subscription: sub}}, &key))
	require.Equal(t, 7, key.User.Concurrency)
	require.Zero(t, original.User.Concurrency)
	got, _ := middleware.GetSubscriptionFromContext(c)
	require.Same(t, sub, got)
}

func TestAutoHTTPAdmissionFailsClosed(t *testing.T) {
	c, w := gin.CreateTestContext(httptest.NewRecorder())
	_ = w
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	key := &service.APIKey{ID: 1, RoutingMode: service.APIKeyRoutingAuto}
	require.False(t, admitAutoHTTPRoute(c, autoAdmissionStub{err: service.ErrAutoRouteNoAccess}, &key))
	require.True(t, c.IsAborted())
	require.Equal(t, http.StatusForbidden, c.Writer.Status())
}

func TestAutoHTTPModelRewritePreservesPublicDecision(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(""))
	groupID := int64(12)
	d := &service.AutoRouteDecision{Key: &service.APIKey{ID: 1, GroupID: &groupID}, PublicModel: "alias", UpstreamModel: "gpt-5", Composite: &service.CompositeRouteDecision{Matched: true}}
	c.Request = c.Request.WithContext(d.RequestContext(c.Request.Context()))
	body, model := []byte(`{"model":"alias","input":"keep me","unknown":123}`), "alias"
	require.True(t, applyAutoHTTPModel(c, &body, &model))
	require.Equal(t, "gpt-5", model)
	require.JSONEq(t, `{"model":"gpt-5","input":"keep me","unknown":123}`, string(body))
	require.Equal(t, "alias", d.PublicModel)
}
