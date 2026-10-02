package routes

import (
	"github.com/LuckyKuang/sub2api-plus/internal/handler"
	"github.com/LuckyKuang/sub2api-plus/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

func registerUserTeamRoutes(authenticated *gin.RouterGroup, h *handler.Handlers, stepUpAuth middleware.StepUpAuthMiddleware, panelRateLimiter *middleware.PanelRateLimiter) {
	// 团队管理沿用 Plus 的认证、面板限流和审计链；敏感生命周期操作额外要求 step-up。
	team := authenticated.Group("/team")
	{
		team.GET("", h.Team.GetCurrent)
		team.POST("", h.Team.Create)
		team.PATCH("", h.Team.Update)
		team.PATCH("/default-member-limits", h.Team.UpdateDefaultMemberLimits)
		team.POST("/status", gin.HandlerFunc(stepUpAuth), h.Team.SetStatus)
		team.DELETE("", gin.HandlerFunc(stepUpAuth), h.Team.Dissolve)
		team.GET("/members", h.Team.ListMembers)
		team.GET("/usage", panelRateLimiter.Heavy(), h.Team.GetUsageSummary)
		team.GET("/usage/members", panelRateLimiter.Heavy(), h.Team.ListMemberUsageSeries)
		team.GET("/usage/logs", panelRateLimiter.Heavy(), h.Team.ListUsageLogs)
		team.GET("/keys", h.Team.ListTeamKeys)
		team.POST("/keys/:id/disable", h.Team.DisableTeamKey)
		team.POST("/keys/:id/enable", h.Team.EnableTeamKey)
		team.DELETE("/keys/:id", h.Team.DeleteTeamKey)
		team.DELETE("/members/:user_id", h.Team.RemoveMember)
		team.PATCH("/members/:user_id/limits", h.Team.UpdateMemberLimits)
		team.POST("/members/:user_id/usage/reset", h.Team.ResetMemberUsage)
		team.POST("/leave", h.Team.Leave)
		team.GET("/invitations", h.Team.ListInvitations)
		team.POST("/invitations", h.Team.Invite)
		team.POST("/invitations/preview", h.Team.PreviewInvitation)
		team.POST("/invitations/resolve", h.Team.ResolveInvitation)
		team.POST("/invitations/:id/reissue", h.Team.ReissueInvitation)
		team.DELETE("/invitations/:id", h.Team.RevokeInvitation)
		team.POST("/ownership-transfer", gin.HandlerFunc(stepUpAuth), h.Team.StartOwnershipTransfer)
		team.POST("/ownership-transfer/resolve", gin.HandlerFunc(stepUpAuth), h.Team.ResolveOwnershipTransfer)
		team.DELETE("/ownership-transfer", gin.HandlerFunc(stepUpAuth), h.Team.CancelOwnershipTransfer)
	}
}
