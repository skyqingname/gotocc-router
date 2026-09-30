package handler

import (
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/response"
	middleware2 "github.com/LuckyKuang/sub2api-plus/internal/server/middleware"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
)

// LC-024 agent center: a self-service application the user submits, and an admin
// read-only membership list. Applying takes effect immediately.
type AgentHandler struct{ service *service.AgentService }

func NewAgentHandler(s *service.AgentService) *AgentHandler { return &AgentHandler{service: s} }

func (h *AgentHandler) subject(c *gin.Context) (int64, bool) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return 0, false
	}
	return subject.UserID, true
}

// Overview returns the caller's own enrollment state. status is empty when the
// user has never applied.
func (h *AgentHandler) Overview(c *gin.Context) {
	userID, ok := h.subject(c)
	if !ok {
		return
	}
	profile, err := h.service.Overview(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, profile)
}

// Apply submits the application. It takes no body: the applicant's identity is
// the whole payload.
func (h *AgentHandler) Apply(c *gin.Context) {
	userID, ok := h.subject(c)
	if !ok {
		return
	}
	profile, err := h.service.Apply(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, profile)
}

func (h *AgentHandler) List(c *gin.Context) {
	page, size := response.ParsePagination(c)
	items, total, err := h.service.List(c.Request.Context(), c.Query("search"), page, size)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, size)
}
