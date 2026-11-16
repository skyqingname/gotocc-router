package handler

import (
	"fmt"
	"net/http"
	"strings"

	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
	middleware2 "github.com/LuckyKuang/sub2api-plus/internal/server/middleware"
	"github.com/LuckyKuang/sub2api-plus/internal/service"

	"github.com/gin-gonic/gin"
)

// VideoTaskUserHandler serves the task center's video tab for the signed-in
// user. Tasks are scoped to the actor that created them across every API key.
type VideoTaskUserHandler struct {
	tasks *service.VideoTaskUserService
}

func NewVideoTaskUserHandler(tasks *service.VideoTaskUserService) *VideoTaskUserHandler {
	return &VideoTaskUserHandler{tasks: tasks}
}

func (h *VideoTaskUserHandler) List(c *gin.Context) {
	actorUserID, ok := videoTaskActor(c)
	if !ok {
		return
	}
	status := strings.TrimSpace(c.Query("status"))
	if status != "" && status != service.VideoTaskUserStatusProcessing && status != service.VideoTaskUserStatusCompleted && status != service.VideoTaskUserStatusFailed {
		imageTaskJSONError(c, http.StatusBadRequest, "INVALID_STATUS", "status must be processing, completed, or failed")
		return
	}
	model, since, until, ok := taskCenterRange(c)
	if !ok {
		return
	}
	limit, validLimit := queryTimestamp(c, "limit")
	offset, validOffset := queryTimestamp(c, "offset")
	if !validLimit || !validOffset || (c.Query("limit") != "" && limit == 0) || limit > service.VideoTaskUserMaxLimit {
		imageTaskJSONError(c, http.StatusBadRequest, "INVALID_PAGINATION", fmt.Sprintf("limit must be between 1 and %d and offset must be a non-negative integer", service.VideoTaskUserMaxLimit))
		return
	}
	if limit == 0 {
		limit = service.VideoTaskUserDefaultLimit
	}
	result, err := h.tasks.List(c.Request.Context(), actorUserID, service.VideoTaskUserFilter{
		Status: status, Model: model, Since: since, Until: until, Limit: int(limit), Offset: int(offset),
	})
	if err != nil {
		videoTaskError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, result)
}

func (h *VideoTaskUserHandler) Filters(c *gin.Context) {
	actorUserID, ok := videoTaskActor(c)
	if !ok {
		return
	}
	filters, err := h.tasks.Filters(c.Request.Context(), actorUserID)
	if err != nil {
		videoTaskError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, filters)
}

func (h *VideoTaskUserHandler) Summary(c *gin.Context) {
	actorUserID, ok := videoTaskActor(c)
	if !ok {
		return
	}
	model, since, until, ok := taskCenterRange(c)
	if !ok {
		return
	}
	summary, err := h.tasks.Summary(c.Request.Context(), actorUserID, service.VideoTaskUserFilter{Model: model, Since: since, Until: until})
	if err != nil {
		videoTaskError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, summary)
}

// Content streams the video of a completed task. Once the upstream response
// has started, a later failure can only end the stream.
func (h *VideoTaskUserHandler) Content(c *gin.Context) {
	actorUserID, ok := videoTaskActor(c)
	if !ok {
		return
	}
	err := h.tasks.ServeContent(c.Request.Context(), c, actorUserID, strings.TrimSpace(c.Param("task_id")))
	if err != nil && !c.Writer.Written() {
		videoTaskError(c, err)
	}
}

func videoTaskActor(c *gin.Context) (int64, bool) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		imageTaskJSONError(c, http.StatusUnauthorized, "VIDEO_TASK_UNAUTHORIZED", "authentication is required")
		return 0, false
	}
	return subject.UserID, true
}

// taskCenterRange parses the model and [since, until) unix-second range shared
// by the task center lists and summaries.
func taskCenterRange(c *gin.Context) (string, int64, int64, bool) {
	since, ok := queryTimestamp(c, "since")
	if !ok {
		imageTaskJSONError(c, http.StatusBadRequest, "INVALID_TIME_RANGE", "since must be a unix timestamp")
		return "", 0, 0, false
	}
	until, ok := queryTimestamp(c, "until")
	if !ok {
		imageTaskJSONError(c, http.StatusBadRequest, "INVALID_TIME_RANGE", "until must be a unix timestamp")
		return "", 0, 0, false
	}
	if since > 0 && until > 0 && since >= until {
		imageTaskJSONError(c, http.StatusBadRequest, "INVALID_TIME_RANGE", "since must be earlier than until")
		return "", 0, 0, false
	}
	return strings.TrimSpace(c.Query("model")), since, until, true
}

func videoTaskError(c *gin.Context, err error) {
	imageTaskJSONError(c, infraerrors.Code(err), infraerrors.Reason(err), infraerrors.Message(err))
}
