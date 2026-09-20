package dto

import "github.com/LuckyKuang/sub2api-plus/internal/service"

// MarketplaceStats is the narrow public aggregate used by the GotoCC homepage.
type MarketplaceStats struct {
	TodayTokens int64 `json:"today_tokens"`
	TotalTokens int64 `json:"total_tokens"`
	TotalUsers  int64 `json:"total_users"`
}

func MarketplaceStatsFromService(stats *service.DashboardPublicStats) *MarketplaceStats {
	if stats == nil {
		return nil
	}
	return &MarketplaceStats{
		TodayTokens: stats.TodayTokens,
		TotalTokens: stats.TotalTokens,
		TotalUsers:  stats.TotalUsers,
	}
}
