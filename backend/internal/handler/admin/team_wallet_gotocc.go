package admin

import (
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/response"
	"github.com/LuckyKuang/sub2api-plus/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

func (h *TeamHandler) SetBalance(c *gin.Context) {
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
		Balance     *float64 `json:"balance" binding:"required,gte=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	wallet, err := h.service.AdminSetWalletBalance(c.Request.Context(), teamID, subject.UserID, req.OperationID, *req.Balance)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, wallet)
}
