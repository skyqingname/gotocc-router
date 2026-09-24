package handler

import (
	"strconv"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/response"
	middleware2 "github.com/LuckyKuang/sub2api-plus/internal/server/middleware"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
)

// LC-024 agent center: a self-service application the user submits, and an admin
// review queue. Nothing is collected from the applicant beyond the submission.
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
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	items, total, err := h.service.List(c.Request.Context(), c.Query("status"), c.Query("search"), page, size)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, size)
}

// Review records the admin verdict. approve=false rejects, which the applicant
// may undo by re-applying; a rejection is not permanent.
func (h *AgentHandler) Review(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || userID <= 0 {
		response.BadRequest(c, "Invalid user")
		return
	}
	adminID, ok := h.subject(c)
	if !ok {
		return
	}
	var in struct {
		Approve bool `json:"approve"`
	}
	if c.ShouldBindJSON(&in) != nil {
		response.BadRequest(c, "Invalid review payload")
		return
	}
	if err := h.service.Review(c.Request.Context(), userID, adminID, in.Approve); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"user_id": userID, "status": agentStatusLabel(in.Approve)})
}

func agentStatusLabel(approve bool) string {
	if approve {
		return service.AgentStatusApproved
	}
	return service.AgentStatusRejected
}
