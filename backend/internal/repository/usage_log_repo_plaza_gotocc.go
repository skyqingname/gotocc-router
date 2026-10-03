package repository

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/service"
)

// GetPlazaModelStats 汇总 since 之后各模型的成功请求数、平均首字耗时，以及按运维 SLA 口径计的失败数
// （状态码 ≥ 400 且不是业务限额拒绝），供模型广场展示；只返回聚合数字，不含账号或用户信息。
func (r *usageLogRepository) GetPlazaModelStats(ctx context.Context, since time.Time) (map[string]service.PlazaModelStats, error) {
	rows, err := r.sql.QueryContext(ctx, `
		WITH ok AS (
			SELECT LOWER(model) AS model, COUNT(*) AS requests, AVG(first_token_ms) FILTER (WHERE first_token_ms > 0) AS first_token_ms
			FROM usage_logs
			WHERE created_at >= $1
			GROUP BY LOWER(model)
		), failed AS (
			SELECT LOWER(model) AS model, COUNT(*) AS errors
			FROM ops_error_logs
			WHERE created_at >= $1 AND COALESCE(status_code, 0) >= 400 AND NOT is_business_limited AND COALESCE(model, '') <> ''
			GROUP BY LOWER(model)
		)
		SELECT COALESCE(ok.model, failed.model), COALESCE(ok.requests, 0), COALESCE(failed.errors, 0), ok.first_token_ms
		FROM ok FULL OUTER JOIN failed ON failed.model = ok.model`, since.UTC())
	if err != nil {
		return nil, err
	}
	// 查询结束时关闭结果集，读取阶段的错误统一通过 rows.Err 返回。
	defer func() { _ = rows.Close() }()
	out := make(map[string]service.PlazaModelStats)
	for rows.Next() {
		var model string
		var requests, errors int64
		var firstToken sql.NullFloat64
		if err := rows.Scan(&model, &requests, &errors, &firstToken); err != nil {
			return nil, err
		}
		stats := service.PlazaModelStats{Requests: requests, Errors: errors}
		rate := float64(requests) / float64(requests+errors) * 100
		stats.SuccessRate = &rate
		if firstToken.Valid {
			value := firstToken.Float64
			stats.AvgFirstTokenMs = &value
		}
		out[strings.ToLower(model)] = stats
	}
	return out, rows.Err()
}
