package admin

import (
	"strconv"

	"github.com/LuckyKuang/sub2api-plus/internal/handler/dto"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/pagination"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// GetUsages lists the users who redeemed a shared redeem code.
// GET /api/v1/admin/redeem-codes/:id/usages
func (h *RedeemHandler) GetUsages(c *gin.Context) {
	codeID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid redeem code ID")
		return
	}

	page, pageSize := response.ParsePagination(c)
	usages, result, err := h.redeemService.ListUsages(c.Request.Context(), codeID, pagination.PaginationParams{Page: page, PageSize: pageSize})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]dto.RedeemCodeUsage, 0, len(usages))
	for i := range usages {
		out = append(out, dto.RedeemCodeUsageFromService(&usages[i]))
	}
	response.Paginated(c, out, result.Total, page, pageSize)
}
