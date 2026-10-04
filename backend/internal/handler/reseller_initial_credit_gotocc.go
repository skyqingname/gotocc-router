package handler

import (
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

func (h *ResellerHandler) InitialCredit(c *gin.Context) {
	ownerID, ok := h.owner(c)
	if !ok {
		return
	}
	var input struct {
		InitialCredit *float64 `json:"initial_credit" binding:"required,gte=0"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "请填写有效的初始额度")
		return
	}
	profile, err := h.service.SetInitialCredit(c.Request.Context(), ownerID, *input.InitialCredit)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, profile)
}
