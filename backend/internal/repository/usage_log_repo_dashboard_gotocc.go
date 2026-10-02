package repository

import (
	"context"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/timezone"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
)

// GetDashboardPublicStats reads only aggregate homepage counters and never returns billing details.
func (r *usageLogRepository) GetDashboardPublicStats(ctx context.Context, start, end time.Time, useAggregates bool) (*service.DashboardPublicStats, error) {
	stats := &service.DashboardPublicStats{}
	if err := r.fillDashboardPublicUserStats(ctx, stats); err != nil {
		return nil, err
	}
	if useAggregates {
		if err := r.fillDashboardPublicTokenStatsAggregated(ctx, stats, timezone.Today()); err != nil {
			return nil, err
		}
		return stats, nil
	}
	if err := r.fillDashboardPublicTokenStatsFromUsageLogs(ctx, stats, start, end, timezone.Today()); err != nil {
		return nil, err
	}
	return stats, nil
}

func (r *usageLogRepository) fillDashboardPublicTokenStatsAggregated(ctx context.Context, stats *service.DashboardPublicStats, todayStart time.Time) error {
	query := `
		SELECT
			COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens) FILTER (WHERE bucket_date = $1::date), 0) AS today_tokens,
			COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens), 0) AS total_tokens
		FROM usage_dashboard_daily
	`
	return scanSingleRow(ctx, r.sql, query, []any{todayStart}, &stats.TodayTokens, &stats.TotalTokens)
}

func (r *usageLogRepository) fillDashboardPublicTokenStatsFromUsageLogs(ctx context.Context, stats *service.DashboardPublicStats, start, end, todayStart time.Time) error {
	startUTC := start.UTC()
	endUTC := end.UTC()
	todayStartUTC := todayStart.UTC()
	todayEndUTC := todayStartUTC.Add(24 * time.Hour)
	query := `
		WITH scoped AS (
			SELECT created_at, input_tokens, output_tokens, cache_creation_tokens, cache_read_tokens
			FROM usage_logs
			WHERE created_at >= LEAST($1::timestamptz, $3::timestamptz)
				AND created_at < GREATEST($2::timestamptz, $4::timestamptz)
		)
		SELECT
			COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens) FILTER (WHERE created_at >= $3::timestamptz AND created_at < $4::timestamptz), 0) AS today_tokens,
			COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens) FILTER (WHERE created_at >= $1::timestamptz AND created_at < $2::timestamptz), 0) AS total_tokens
		FROM scoped
	`
	return scanSingleRow(ctx, r.sql, query, []any{startUTC, endUTC, todayStartUTC, todayEndUTC}, &stats.TodayTokens, &stats.TotalTokens)
}

func (r *usageLogRepository) fillDashboardPublicUserStats(ctx context.Context, stats *service.DashboardPublicStats) error {
	return scanSingleRow(ctx, r.sql, "SELECT COUNT(*) FROM users WHERE deleted_at IS NULL", nil, &stats.TotalUsers)
}
