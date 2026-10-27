package service

import (
	"fmt"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/apicompat"
	"github.com/gin-gonic/gin"
)

// newChatResponseTiming samples the existing forwarding clock immediately
// before the final client response or finish chunk is serialized. It includes
// upstream waiting/transfer and local conversion, not client download time.
func newChatResponseTiming(startedAt time.Time, response *apicompat.ResponsesResponse) *apicompat.ChatResponseTiming {
	timing := &apicompat.ChatResponseTiming{
		ForwardDurationMs: time.Since(startedAt).Milliseconds(),
	}
	if response != nil && response.Status == "completed" && response.CreatedAt > 0 &&
		response.CompletedAt != nil && *response.CompletedAt >= float64(response.CreatedAt) {
		timing.UpstreamResponse = &apicompat.ReportedResponseTiming{
			CreatedAt:   response.CreatedAt,
			CompletedAt: *response.CompletedAt,
			DurationMs:  (*response.CompletedAt - float64(response.CreatedAt)) * 1000,
		}
	}
	return timing
}

func writeChatResponseTimingHeader(c *gin.Context, timing *apicompat.ChatResponseTiming) {
	value := fmt.Sprintf(`gotocc_forward;dur=%d;desc="GoToCC forwarding wall time"`, timing.ForwardDurationMs)
	if upstream := timing.UpstreamResponse; upstream != nil {
		value += fmt.Sprintf(`, upstream_response;dur=%g;desc="Upstream reported response lifetime"`, upstream.DurationMs)
	}
	c.Writer.Header().Add("Server-Timing", value)
}

func attachChatFinishMetadata(chunks []apicompat.ChatCompletionsChunk, startedAt time.Time, response *apicompat.ResponsesResponse, route *apicompat.ChatResponseRoute) {
	for i := range chunks {
		for _, choice := range chunks[i].Choices {
			if choice.FinishReason != nil {
				chunks[i].GoToCCTiming = newChatResponseTiming(startedAt, response)
				chunks[i].GoToCCRoute = route
				break
			}
		}
	}
}
