package handler

import (
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
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

const opsLocalTakeoverTimeoutKey = "gotocc_local_takeover_timeout"

func (h *OpenAIGatewayHandler) writeTakeoverTimeout(c *gin.Context, failure *service.UpstreamFailoverError, streamStarted bool) bool {
	if failure == nil || failure.Reason != service.OpenAITakeoverTimeoutCode {
		return false
	}
	c.Set(opsLocalTakeoverTimeoutKey, true)
	if GetInboundEndpoint(c) == EndpointMessages {
		body := gin.H{"type": "error", "error": gin.H{"type": "api_error", "code": service.OpenAITakeoverTimeoutCode, "message": failure.ClientMessage}}
		if streamStarted {
			c.SSEvent("error", body)
			c.Writer.Flush()
		} else {
			c.JSON(http.StatusServiceUnavailable, body)
		}
	} else {
		h.handleStreamingAwareErrorWithCode(c, http.StatusServiceUnavailable, "service_unavailable_error", service.OpenAITakeoverTimeoutCode, failure.ClientMessage, streamStarted, true)
	}
	return true
}
