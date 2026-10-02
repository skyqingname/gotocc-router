package routes

import (
	"github.com/LuckyKuang/sub2api-plus/internal/handler"
	"github.com/gin-gonic/gin"
)

func registerAdminSupportGotoccRoutes(support *gin.RouterGroup, h *handler.Handlers) {
	support.GET("/user/agent", h.Agent.Overview)
	support.GET("/keys/:id/routing-capabilities", h.Gateway.UserRoutingCapabilities)
	support.GET("/groups/routing-priorities", h.Gateway.UserRoutingPriorities)
	support.GET("/team", h.Team.GetCurrent)
	support.GET("/team/keys", h.Team.ListTeamKeys)
	support.GET("/team/members", h.Team.ListMembers)
	support.GET("/team/usage/members", h.Team.ListMemberUsageSeries)
}
