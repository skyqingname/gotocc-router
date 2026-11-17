package handler

import (
	"net/http"
	"strings"

	"github.com/LuckyKuang/sub2api-plus/internal/service"

	"github.com/gin-gonic/gin"
)

// Summary counts the signed-in user's image tasks with the console list
// filters (status, model, since, until).
func (h *ImageTaskUserHandler) Summary(c *gin.Context) {
	owner, ok := h.owner(c)
	if !ok {
		return
	}
	status := strings.TrimSpace(c.Query("status"))
	if status != "" && status != service.ImageTaskStatusProcessing && status != service.ImageTaskStatusCompleted && status != service.ImageTaskStatusFailed {
		imageTaskJSONError(c, http.StatusBadRequest, "INVALID_STATUS", "status must be processing, completed, or failed")
		return
	}
	model, since, until, ok := taskCenterRange(c)
	if !ok {
		return
	}
	summary, err := h.tasks.Summary(c.Request.Context(), owner, service.ImageTaskHistoryFilter{Status: status, Model: model, Since: since, Until: until})
	if err != nil {
		imageTaskError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, summary)
}
