//go:build unit

package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LuckyKuang/sub2api-plus/internal/server/middleware"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type autoModelCatalogStub struct {
	models   []service.AutoRouteModel
	manifest []byte
	err      error
	calls    int
	input    service.AutoRouteRequest
}

func TestUserRoutingPrioritiesRequiresAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &GatewayHandler{}
	router := gin.New()
	router.GET("/groups/routing-priorities", h.UserRoutingPriorities)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/groups/routing-priorities?scope=personal", nil))
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func (s *autoModelCatalogStub) ListModels(_ context.Context, _ *service.APIKey, input service.AutoRouteRequest) ([]service.AutoRouteModel, error) {
	s.calls++
	s.input = input
	return s.models, s.err
}
func (s *autoModelCatalogStub) BuildCodexModelsManifest(context.Context, *service.APIKey) ([]byte, error) {
	s.calls++
	return s.manifest, s.err
}

func TestAutoModelsNeverFallsThroughToGlobalCatalog(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, fail := range []bool{false, true} {
		catalog := &autoModelCatalogStub{models: []service.AutoRouteModel{}}
		if fail {
			catalog.err = service.ErrAutoRouteUnavailable
		}
		r := gin.New()
		r.GET("/v1/models", func(c *gin.Context) {
			middleware.BindAPIKeyContext(c, &service.APIKey{ID: 1, RoutingMode: service.APIKeyRoutingAuto, User: &service.User{ID: 2}}, nil)
			serveAutoModels(c, catalog)
		}, func(c *gin.Context) { t.Error("auto model request escaped to global fallback") })
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v1/models", nil))
		if fail {
			require.Equal(t, 503, w.Code)
		} else {
			require.JSONEq(t, `{"object":"list","data":[]}`, w.Body.String())
		}
	}
}

func TestAutoCodexManifestReauthorizesBeforeNotModified(t *testing.T) {
	gin.SetMode(gin.TestMode)
	catalog := &autoModelCatalogStub{manifest: []byte(`{"models":[{"slug":"gpt-test"}]}`)}
	r := gin.New()
	r.GET("/v1/models", func(c *gin.Context) {
		middleware.BindAPIKeyContext(c, &service.APIKey{ID: 1, RoutingMode: service.APIKeyRoutingAuto, User: &service.User{ID: 2}}, nil)
		serveAutoModels(c, catalog)
	})
	first := httptest.NewRecorder()
	r.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/v1/models?client_version=1", nil))
	require.Equal(t, 200, first.Code)
	require.NotEmpty(t, first.Header().Get("ETag"))
	catalog.err = service.ErrAutoRouteNoAccess
	req := httptest.NewRequest(http.MethodGet, "/v1/models?client_version=1", nil)
	req.Header.Set("If-None-Match", first.Header().Get("ETag"))
	second := httptest.NewRecorder()
	r.ServeHTTP(second, req)
	require.Equal(t, 403, second.Code)
	require.Equal(t, 2, catalog.calls)
}
