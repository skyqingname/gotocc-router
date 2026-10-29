package handler

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// A canceled downstream request can close its route attempt before the first
// upstream output arrives. Keep that cancellation on the existing 499 path.
func handleCanceledOpenAISessionRoute(c *gin.Context, err error) bool {
	if !failoverClientGone(c) {
		return false
	}
	requestLogger(c, "handler.openai_gateway.session_route").Info("openai.session_route_client_disconnected",
		zap.NamedError("route_error", err),
		zap.NamedError("client_error", c.Request.Context().Err()),
	)
	return true
}
