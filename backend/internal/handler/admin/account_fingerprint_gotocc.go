package admin

import (
	"strconv"
	"strings"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

// Fingerprint samples the selected account without the connectivity probe's
// post-success recovery action. Each call is one independent numeric challenge.
func (h *AccountHandler) Fingerprint(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	var req struct {
		ModelID string `json:"model_id" binding:"required"`
		Prompt  string `json:"prompt" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "model_id and prompt are required")
		return
	}
	req.ModelID, req.Prompt = strings.TrimSpace(req.ModelID), strings.TrimSpace(req.Prompt)
	if req.ModelID == "" || req.Prompt == "" {
		response.BadRequest(c, "model_id and prompt are required")
		return
	}
	_ = h.accountTestService.TestAccountFingerprint(c, accountID, req.ModelID, req.Prompt)
}
