//go:build unit

package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/LuckyKuang/sub2api-plus/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// Preferences are authenticated settings, not credential-authenticated gateway
// requests. Anonymous callers and unknown administrator fields must be rejected.
func TestUserKeyRoutingPreferenceRequiresAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &GatewayHandler{}
	router := gin.New()
	router.GET("/keys/:id/routing-policy", h.UserGetKeyRoutingPreference)
	router.PUT("/keys/:id/routing-policy", h.UserUpdateKeyRoutingPreference)
	for _, method := range []string{http.MethodGet, http.MethodPut} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(method, "/keys/1/routing-policy", strings.NewReader("null")))
		require.Equal(t, http.StatusUnauthorized, recorder.Code)
	}
}

func TestUserKeyRoutingPreferenceStrictPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &GatewayHandler{}
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 7}) })
	router.PUT("/keys/:id/routing-policy", h.UserUpdateKeyRoutingPreference)
	for _, body := range []string{`{"allow_user_override":true}`, `{"default_group_order":[1]} {}`, `[]`, `{"model_rules":`, ""} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPut, "/keys/1/routing-policy", strings.NewReader(body)))
		require.Equal(t, http.StatusBadRequest, recorder.Code, body)
	}
}
