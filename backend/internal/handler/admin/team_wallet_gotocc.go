package admin

import (
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/response"
	"github.com/LuckyKuang/sub2api-plus/internal/server/middleware"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
)

// AdjustBalance adds to, subtracts from, or sets the available team balance.
func (h *TeamHandler) AdjustBalance(c *gin.Context) {
	teamID, ok := adminTeamID(c)
	if !ok {
		return
	}
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req struct {
		OperationID string   `json:"operation_id" binding:"required"`
		Operation   string   `json:"operation" binding:"required,oneof=add subtract set"`
		Amount      *float64 `json:"amount" binding:"required,gte=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	wallet, err := h.service.AdminAdjustWalletBalance(c.Request.Context(), teamID, subject.UserID, req.OperationID, service.TeamBalanceOperation(req.Operation), *req.Amount)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, wallet)
}
