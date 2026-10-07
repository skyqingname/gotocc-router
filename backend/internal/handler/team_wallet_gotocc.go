package handler

import (
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

func (h *TeamHandler) FundWallet(c *gin.Context) {
	subject, ok := teamSubject(c)
	if !ok {
		return
	}
	var req struct {
		OperationID string  `json:"operation_id" binding:"required"`
		Amount      float64 `json:"amount" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	result, err := h.service.FundWallet(c.Request.Context(), subject.UserID, req.OperationID, req.Amount)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}
