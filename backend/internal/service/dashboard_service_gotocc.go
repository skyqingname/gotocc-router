package service

import (
	"context"
	"fmt"
	"time"
)

type dashboardPublicStatsFetcher interface {
	GetDashboardPublicStats(ctx context.Context, start, end time.Time, useAggregates bool) (*DashboardPublicStats, error)
}

// DashboardPublicStats contains only the aggregate fields exposed on the public homepage.
type DashboardPublicStats struct {
	TodayTokens int64
	TotalTokens int64
	TotalUsers  int64
}

// GetPublicDashboardStats avoids the full administrator dashboard query when the repository supports it.
func (s *DashboardService) GetPublicDashboardStats(ctx context.Context) (*DashboardPublicStats, error) {
	if fetcher, ok := s.usageRepo.(dashboardPublicStatsFetcher); ok {
		now := time.Now().UTC()
		start := truncateToDayUTC(now.AddDate(0, 0, -s.aggUsageDays))
		stats, err := fetcher.GetDashboardPublicStats(ctx, start, now, s.aggEnabled)
		if err != nil {
			return nil, fmt.Errorf("get public dashboard stats: %w", err)
		}
		return stats, nil
	}

	stats, err := s.GetDashboardStats(ctx)
	if err != nil {
		return nil, err
	}
	return &DashboardPublicStats{
		TodayTokens: stats.TodayTokens,
		TotalTokens: stats.TotalTokens,
		TotalUsers:  stats.TotalUsers,
	}, nil
}
