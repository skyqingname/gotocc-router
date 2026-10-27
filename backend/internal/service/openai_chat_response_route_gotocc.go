package service

import (
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/apicompat"
	"github.com/gin-gonic/gin"
)

func newChatResponseRoute(c *gin.Context, upstreamModel string) *apicompat.ChatResponseRoute {
	return &apicompat.ChatResponseRoute{
		InboundEndpoint:  c.Request.URL.Path,
		UpstreamEndpoint: GetActualOpenAIUpstreamEndpoint(c),
		UpstreamModel:    upstreamModel,
	}
}
