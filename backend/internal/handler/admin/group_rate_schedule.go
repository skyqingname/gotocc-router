package admin

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/rateschedule"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/response"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
)

// RateScheduleHandler is registered under the existing admin auth, compliance,
// rate-limit and audit middleware. It cannot be mounted on a public route.
func (h *GroupHandler) RateScheduleHandler(settings *service.SettingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || id <= 0 {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": "invalid_group_id", "message": "Invalid group ID"})
			return
		}
		group, err := h.adminService.GetGroup(c.Request.Context(), id)
		if err != nil { response.ErrorFrom(c, err); return }
		if c.Request.Method == http.MethodGet {
			view, err := settings.GetGroupRateSchedule(c.Request.Context(), group)
			if err != nil { response.ErrorFrom(c, err); return }
			response.Success(c, view)
			return
		}
		if c.Request.Method != http.MethodPut { c.AbortWithStatus(http.StatusMethodNotAllowed); return }
		var input struct {
			ExpectedVersion *int64 `json:"expected_version"`
			Config *rateschedule.Config `json:"config"`
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 128*1024)
		decoder := json.NewDecoder(c.Request.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil || input.ExpectedVersion == nil || input.Config == nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": "invalid_rate_schedule", "message": "expected_version and config are required"})
			return
		}
		if err := decoder.Decode(new(any)); err != io.EOF {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": "invalid_rate_schedule", "message": "One JSON object is required"})
			return
		}
		if _, err := settings.SaveGroupRateSchedule(c.Request.Context(), id, *input.ExpectedVersion, *input.Config); err != nil {
			response.ErrorFrom(c, err)
			return
		}
		view, err := settings.GetGroupRateSchedule(c.Request.Context(), group)
		if err != nil { response.ErrorFrom(c, err); return }
		response.Success(c, view)
	}
}
