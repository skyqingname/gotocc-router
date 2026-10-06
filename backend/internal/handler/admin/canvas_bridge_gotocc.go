package admin

import (
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/response"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
)

// CanvasBridgeHandler 供影策画布前置桥接服务以管理员 API Key 调用。
type CanvasBridgeHandler struct {
	service *service.CanvasBridgeService
}

func NewCanvasBridgeHandler(canvasBridgeService *service.CanvasBridgeService) *CanvasBridgeHandler {
	return &CanvasBridgeHandler{service: canvasBridgeService}
}

func (h *CanvasBridgeHandler) Verify(c *gin.Context) {
	var in service.CanvasBridgeVerifyInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	account, err := h.service.Verify(c.Request.Context(), in)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, account)
}

func (h *CanvasBridgeHandler) Transfer(c *gin.Context) {
	var in service.CanvasBridgeTransferInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	transfer, err := h.service.Transfer(c.Request.Context(), in)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, transfer)
}

func (h *CanvasBridgeHandler) Reverse(c *gin.Context) {
	var in service.CanvasBridgeReverseInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	transfer, err := h.service.Reverse(c.Request.Context(), c.Param("id"), in)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, transfer)
}
