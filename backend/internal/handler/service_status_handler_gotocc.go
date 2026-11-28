package handler

import (
	"strconv"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/response"
	"github.com/LuckyKuang/sub2api-plus/internal/server/middleware"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
)

// ServiceStatusHandler 是用户端服务状态：按分组的被动检测和分组可用模型。
// 读的是当前用户（或管理员代看时的目标用户）能用的分组；只看真实请求，不依赖渠道监控开关。
type ServiceStatusHandler struct {
	status *service.ServiceStatusService
}

func NewServiceStatusHandler(status *service.ServiceStatusService) *ServiceStatusHandler {
	return &ServiceStatusHandler{status: status}
}

func (h *ServiceStatusHandler) subject(c *gin.Context) (int64, bool) {
	subject, ok := middleware.GetReadSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return 0, false
	}
	return subject.UserID, true
}

// Snapshot GET /api/v1/service-status
func (h *ServiceStatusHandler) Snapshot(c *gin.Context) {
	userID, ok := h.subject(c)
	if !ok {
		return
	}
	result, err := h.status.Snapshot(c.Request.Context(), userID, c.GetHeader("Accept-Language"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// Models GET /api/v1/service-status/groups/:id/models
func (h *ServiceStatusHandler) Models(c *gin.Context) {
	userID, ok := h.subject(c)
	if !ok {
		return
	}
	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || groupID <= 0 {
		response.BadRequest(c, "invalid group id")
		return
	}
	result, err := h.status.Models(c.Request.Context(), userID, groupID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}
