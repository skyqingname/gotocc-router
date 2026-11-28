package repository

import (
	"context"
	"database/sql"
	"time"
)

// fillUserDashboardFocus 补齐用户仪表盘重点卡片的统计：近 24 小时成功数、首字中位数、失败数和平台原因失败数，
// 以及近 7 天实际扣费。口径见 usagestats.UserDashboardFocus。
func (r *usageLogRepository) fillUserDashboardFocus(ctx context.Context, stats *UserDashboardStats, userID int64, now time.Time) error {
	dayAgo := now.Add(-24 * time.Hour)
	weekAgo := now.Add(-7 * 24 * time.Hour)
	var ttft sql.NullFloat64
	if err := scanSingleRow(ctx, r.sql, `
		SELECT
			COUNT(*) FILTER (WHERE created_at >= $2),
			percentile_cont(0.5) WITHIN GROUP (ORDER BY first_token_ms) FILTER (WHERE created_at >= $2 AND first_token_ms IS NOT NULL),
			COALESCE(SUM(actual_cost), 0)
		FROM usage_logs
		WHERE user_id = $1 AND created_at >= $3
	`, []any{userID, dayAgo, weekAgo}, &stats.Last24hSuccess, &ttft, &stats.Last7dActualCost); err != nil {
		return err
	}
	if ttft.Valid {
		stats.Last24hTTFTMedianMs = &ttft.Float64
	}
	return scanSingleRow(ctx, r.sql, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE NOT COALESCE(is_business_limited, false) AND error_owner IN ('provider', 'platform'))
		FROM ops_error_logs
		WHERE user_id = $1 AND created_at >= $2 AND COALESCE(status_code, 0) >= 400 AND is_count_tokens = FALSE
	`, []any{userID, dayAgo}, &stats.Last24hFailed, &stats.Last24hPlatformFailed)
}
