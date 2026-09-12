package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LuckyKuang/sub2api-plus/internal/config"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAutoAPIKeyAuthDefersGroupBilling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	key := &service.APIKey{ID: 1, UserID: 7, Status: service.StatusActive,
		RoutingMode: service.APIKeyRoutingAuto, User: &service.User{ID: 7, Status: service.StatusActive}}
	r := gin.New()
	r.Use(func(c *gin.Context) {
		require.True(t, handleAutoAPIKeyAuth(c, &service.APIKeyService{}, &config.Config{}, key, false))
	})
	r.POST("/v1/messages", func(c *gin.Context) {
		actual, ok := GetAPIKeyFromContext(c)
		require.True(t, ok)
		require.Nil(t, actual.GroupID)
		require.True(t, service.IsAutoRoutingRequest(c.Request.Context()))
		c.Status(http.StatusNoContent)
	})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/messages", nil))
	require.Equal(t, http.StatusNoContent, rec.Code)
}

func TestAutoAPIKeyCannotUseUngroupedSchedulingFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(ContextKeyAPIKey), &service.APIKey{ID: 1, RoutingMode: service.APIKeyRoutingAuto})
		c.Next()
	})
	r.Use(RequireGroupAssignment(nil, AnthropicErrorWriter))
	r.POST("/v1/messages", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	rec := httptest.NewRecorder()
	require.NotPanics(t, func() { r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/messages", nil)) })
	require.Equal(t, http.StatusForbidden, rec.Code)
}
