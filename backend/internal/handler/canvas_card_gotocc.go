package handler

import (
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/pagination"
	"strconv"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/response"
	"github.com/LuckyKuang/sub2api-plus/internal/server/middleware"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
)

type CanvasCardHandler struct{ service *service.CanvasCardService }

func NewCanvasCardHandler(s *service.CanvasCardService) *CanvasCardHandler {
	return &CanvasCardHandler{service: s}
}

func (h *CanvasCardHandler) Overview(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	out, err := h.service.Overview(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, out)
}

func (h *CanvasCardHandler) Exchange(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var in service.CanvasCardExchangeInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid exchange request")
		return
	}
	out, err := h.service.Exchange(c.Request.Context(), subject.UserID, in)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, out)
}

func canvasCardPage(c *gin.Context) (int, bool) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 || page-1 > int(^uint(0)>>1)/service.CanvasCardPageSize {
		response.BadRequest(c, "Invalid page")
		return 0, false
	}
	return page, true
}

func (h *CanvasCardHandler) Activity(c *gin.Context) {
	subject, ok := middleware.GetReadSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	page, pageSize := response.ParsePagination(c)
	if page-1 > int(^uint(0)>>1)/pageSize {
		response.BadRequest(c, "Invalid page")
		return
	}
	items, total, err := h.service.Activity(c.Request.Context(), subject.UserID, pagination.PaginationParams{Page: page, PageSize: pageSize})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

// Admin methods are registered only below the existing administrator middleware.
func (h *CanvasCardHandler) Stock(c *gin.Context) {
	out, err := h.service.Stock(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, out)
}

func (h *CanvasCardHandler) Import(c *gin.Context) {
	adminID, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var in service.CanvasCardImportInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid import request")
		return
	}
	if err := h.service.Import(c.Request.Context(), adminID.UserID, in); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"imported": len(in.Codes)})
}

func (h *CanvasCardHandler) Inventory(c *gin.Context) {
	page, ok := canvasCardPage(c)
	if !ok {
		return
	}
	denomination, err := strconv.Atoi(c.DefaultQuery("denomination", "0"))
	if err != nil {
		response.BadRequest(c, "Invalid denomination")
		return
	}
	items, total, err := h.service.Inventory(c.Request.Context(), denomination, c.Query("status"), page)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, int64(total), page, service.CanvasCardPageSize)
}
