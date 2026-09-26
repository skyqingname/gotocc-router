package handler

import (
	"context"

	pkghttputil "github.com/LuckyKuang/sub2api-plus/internal/pkg/httputil"
	"github.com/LuckyKuang/sub2api-plus/internal/server/middleware"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
)

type autoHTTPAdmission interface {
	Admit(context.Context, *service.APIKey) (*service.AutoRouteAdmission, error)
}

// Call only after the endpoint has audited its original payload.
func admitAutoHTTPRoute(c *gin.Context, resolver autoHTTPAdmission, key **service.APIKey) bool {
	if key == nil || *key == nil || !(*key).IsAutoRouting() {
		return true
	}
	if resolver == nil {
		middleware.WriteAutoRoutingError(c, service.ErrAutoRouteUnavailable)
		return false
	}
	admission, err := resolver.Admit(c.Request.Context(), *key)
	if err != nil {
		middleware.WriteAutoRoutingError(c, err)
		return false
	}
	if admission == nil || admission.Key == nil || admission.Key.User == nil {
		middleware.WriteAutoRoutingError(c, service.ErrAutoRouteUnavailable)
		return false
	}
	*key = admission.Key
	middleware.BindAPIKeyContext(c, admission.Key, admission.Subscription)
	return true
}

func applyAutoHTTPModel(c *gin.Context, body *[]byte, model *string) bool {
	decision, ok := service.AutoRouteDecisionFromContext(c.Request.Context())
	if !ok || decision.Composite == nil || decision.UpstreamModel == "" || decision.UpstreamModel == *model {
		return true
	}
	rewritten, err := pkghttputil.RewriteGatewayRoutingModel(c.GetHeader("Content-Type"), *body, false, decision.UpstreamModel)
	if err != nil {
		middleware.WriteAutoRoutingError(c, service.ErrAutoRouteUnavailable)
		return false
	}
	*body, *model = rewritten, decision.UpstreamModel
	return true
}
