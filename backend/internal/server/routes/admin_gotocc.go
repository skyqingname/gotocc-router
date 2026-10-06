package routes

import (
	"github.com/LuckyKuang/sub2api-plus/internal/handler"
	"github.com/LuckyKuang/sub2api-plus/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

func registerTeamRoutes(admin *gin.RouterGroup, h *handler.Handlers, stepUpAuth middleware.StepUpAuthMiddleware) {
	teams := admin.Group("/teams")
	{
		teams.GET("", h.Admin.Team.List)
		teams.POST("", h.Admin.Team.Create)
		teams.GET("/:id", h.Admin.Team.Get)
		teams.GET("/:id/members", h.Admin.Team.ListMembers)
		teams.GET("/:id/usage", h.Admin.Team.GetUsage)
		teams.PATCH("/:id", h.Admin.Team.Update)
		teams.POST("/:id/force-transfer", gin.HandlerFunc(stepUpAuth), h.Admin.Team.ForceTransfer)
		teams.DELETE("/:id", gin.HandlerFunc(stepUpAuth), h.Admin.Team.Dissolve)
	}
}

func registerReusableInvitationCodeRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	if h.Admin == nil || h.Admin.ReusableInvitationCode == nil {
		return
	}
	codes := admin.Group("/reusable-invitation-codes")
	{
		codes.GET("", h.Admin.ReusableInvitationCode.List)
		codes.POST("", h.Admin.ReusableInvitationCode.Create)
		codes.PUT("/:id/owner", h.Admin.ReusableInvitationCode.SetOwner)
		codes.POST("/:id/disable", h.Admin.ReusableInvitationCode.Disable)
		codes.GET("/:id/uses", h.Admin.ReusableInvitationCode.ListUses)
	}
}

// registerAgentRoutes 注册 LC-024 代理中心管理端路由（只读代理名单）
func registerAgentRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	agents := admin.Group("/agents")
	{
		agents.GET("", h.Agent.List)
	}
}

// registerModelPlazaOverrideRoutes 注册模型广场按模型展示信息（简介、厂商、用途）的管理路由。
func registerModelPlazaOverrideRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	admin.GET("/model-plaza/overrides", h.Admin.Setting.GetModelPlazaOverrides)
	admin.PUT("/model-plaza/overrides", h.Admin.Setting.UpdateModelPlazaOverrides)
}

// registerCanvasBridgeRoutes 注册影策画布桥接接口：校验 GoToCC 账号、把余额单向划入画布及退回。
func registerCanvasBridgeRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	bridge := admin.Group("/canvas-bridge")
	{
		bridge.POST("/verify", h.Admin.CanvasBridge.Verify)
		bridge.POST("/transfers", h.Admin.CanvasBridge.Transfer)
		bridge.POST("/transfers/:id/reverse", h.Admin.CanvasBridge.Reverse)
	}
}
