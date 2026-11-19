package handler

import (
	"net/http"

	"github.com/LuckyKuang/sub2api-plus/internal/service"
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

const opsLocalTakeoverTimeoutKey = "gotocc_local_takeover_timeout"

func isLocalTakeoverTimeout(c *gin.Context, turn int) bool {
	if c == nil {
		return false
	}
	recordedTurn, ok := c.Get(opsLocalTakeoverTimeoutKey)
	return ok && recordedTurn == turn
}

func writeAnthropicCodedError(c *gin.Context, status int, errType, code, message string, streamStarted bool) {
	body := gin.H{"type": "error", "error": gin.H{"type": errType, "code": code, "message": message}}
	if streamStarted {
		c.SSEvent("error", body)
		c.Writer.Flush()
		return
	}
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.JSON(status, body)
}

func (h *OpenAIGatewayHandler) writeTakeoverTimeout(c *gin.Context, failure *service.UpstreamFailoverError, streamStarted bool) bool {
	if failure == nil || failure.Reason != service.OpenAITakeoverTimeoutCode {
		return false
	}
	c.Set(opsLocalTakeoverTimeoutKey, c.GetInt(service.OpsStreamTurnKey))
	if GetInboundEndpoint(c) == EndpointMessages {
		if streamStarted {
			service.MarkOpsStreamFailure(c, "api_error", service.OpenAITakeoverTimeoutCode, failure.ClientMessage, http.StatusServiceUnavailable)
		}
		writeAnthropicCodedError(c, http.StatusServiceUnavailable, "api_error", service.OpenAITakeoverTimeoutCode, failure.ClientMessage, streamStarted)
	} else {
		h.handleStreamingAwareErrorWithCode(c, http.StatusServiceUnavailable, "service_unavailable_error", service.OpenAITakeoverTimeoutCode, failure.ClientMessage, streamStarted, true)
	}
	return true
}
