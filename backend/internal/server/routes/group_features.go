package routes

import (
	"github.com/LuckyKuang/sub2api-plus/internal/handler"
	"github.com/LuckyKuang/sub2api-plus/internal/server/middleware"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
)

func RegisterGroupFeatureRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	adminAuth middleware.AdminAuthMiddleware,
	auditLog middleware.AuditLogMiddleware,
	settings *service.SettingService,
	panelRateLimiter *middleware.PanelRateLimiter,
) {
	admin := v1.Group("/admin/group-features")
	admin.Use(gin.HandlerFunc(adminAuth))
	admin.Use(panelRateLimiter.Global())
	admin.Use(gin.HandlerFunc(auditLog))
	admin.Use(middleware.AdminComplianceGuard(settings))
	rates := h.Admin.Group.RateScheduleHandler(settings)
	admin.GET("/:id/rate-schedule", rates)
	admin.PUT("/:id/rate-schedule", rates)
}
