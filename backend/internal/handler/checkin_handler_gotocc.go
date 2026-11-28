package handler

import (
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/response"
	"github.com/LuckyKuang/sub2api-plus/internal/server/middleware"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
)

// CheckinHandler 是签到中心：查看某月的签到日历，以及签今天。
type CheckinHandler struct {
	checkins *service.DailyCheckinService
}

func NewCheckinHandler(checkins *service.DailyCheckinService) *CheckinHandler {
	return &CheckinHandler{checkins: checkins}
}

// Overview GET /api/v1/checkin?month=YYYY-MM
func (h *CheckinHandler) Overview(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	result, err := h.checkins.Overview(c.Request.Context(), subject.UserID, c.Query("month"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// CheckIn POST /api/v1/checkin
func (h *CheckinHandler) CheckIn(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	result, err := h.checkins.CheckIn(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}
