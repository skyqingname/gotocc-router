package handler

import (
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/response"
	middleware2 "github.com/LuckyKuang/sub2api-plus/internal/server/middleware"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *ResellerHandler) PlatformFunding(c *gin.Context) {
	subject, _ := middleware2.GetAuthSubjectFromContext(c)
	account, err := h.service.Repo.CustomerAccount(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		c.Abort()
		return
	}
	if account != nil {
		response.ErrorFrom(c, service.ErrResellerFundingViaOwner)
		c.Abort()
		return
	}
	c.Next()
}
