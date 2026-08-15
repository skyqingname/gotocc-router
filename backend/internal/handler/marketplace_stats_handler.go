package handler

import (
	"github.com/LuckyKuang/sub2api-plus/internal/handler/dto"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/response"
	"github.com/LuckyKuang/sub2api-plus/internal/service"

	"github.com/gin-gonic/gin"
)

// MarketplaceStatsHandler serves only the anonymous aggregate counters used by the local homepage.
type MarketplaceStatsHandler struct {
	dashboardService *service.DashboardService
}

func NewMarketplaceStatsHandler(dashboardService *service.DashboardService) *MarketplaceStatsHandler {
	return &MarketplaceStatsHandler{dashboardService: dashboardService}
}

// GetPublicStats handles GET /api/v1/marketplace/stats.
func (h *MarketplaceStatsHandler) GetPublicStats(c *gin.Context) {
	stats, err := h.dashboardService.GetPublicDashboardStats(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.MarketplaceStatsFromService(stats))
}
