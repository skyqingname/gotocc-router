package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/service"
)

type serviceStatusRepository struct {
	db *sql.DB
}

func NewServiceStatusRepository(db *sql.DB) service.ServiceStatusRepository {
	return &serviceStatusRepository{db: db}
}

// serviceStatusEventsSQL 把窗口内每个请求的最终结果排成一列：
//   - 用量记录是请求的最终结果：正常完成算成功，中途断流（incomplete）算失败，
//     客户端主动断开和被上游内容策略拦下的不计入；
//   - 错误记录只取平台或上游原因（error_owner 为 provider/platform）且客户端收到错误的：
//     业务限制（余额、并发、参数、鉴权等）不算，唯独「没有可用账号」虽被运维标成业务限制
//     （error_phase=routing）仍是平台原因，算失败；
//   - 同一请求号已有用量记录的错误是换号前的中间失败，以最终结果为准；同一请求号的多条错误只算一次。
const serviceStatusEventsSQL = `
WITH usage_events AS (
  SELECT ul.group_id, ul.created_at,
         NOT (ul.completion_status = 'incomplete' OR (ul.completion_status = 'unknown' AND ul.is_complete IS FALSE)) AS ok,
         CASE WHEN ul.timing_version = 1 AND ul.first_output_kind IS NOT NULL THEN ul.first_token_ms END AS ttft,
         ul.input_tokens, ul.cache_read_tokens, ul.cache_creation_tokens
  FROM usage_logs ul
  WHERE ul.created_at >= $1 AND ul.created_at < $2 AND ul.group_id IS NOT NULL
    AND ul.completion_status <> 'client_disconnected' AND COALESCE(ul.request_type, 0) <> 4
), error_events AS (
  SELECT DISTINCT ON (COALESCE(NULLIF(e.request_id, ''), 'error:' || e.id::text))
         e.group_id, e.created_at
  FROM ops_error_logs e
  WHERE e.created_at >= $1 AND e.created_at < $2 AND e.group_id IS NOT NULL
    AND e.status_code >= 400 AND NOT e.is_count_tokens AND COALESCE(e.request_type, 0) <> 4
    AND e.error_owner IN ('provider', 'platform')
    AND (NOT e.is_business_limited OR e.error_phase = 'routing')
    AND NOT EXISTS (
      SELECT 1 FROM usage_logs u
      WHERE NULLIF(e.request_id, '') IS NOT NULL AND u.request_id = e.request_id
        AND u.created_at >= $1 - INTERVAL '90 minutes'
    )
  ORDER BY COALESCE(NULLIF(e.request_id, ''), 'error:' || e.id::text), e.created_at DESC
), events AS (
  SELECT group_id, created_at, ok FROM usage_events
  UNION ALL
  SELECT group_id, created_at, FALSE FROM error_events
), ranked AS (
  SELECT group_id, ok,
         LEAST(FLOOR(EXTRACT(EPOCH FROM created_at - $1) / $3)::int, $4 - 1) AS bucket,
         ROW_NUMBER() OVER (PARTITION BY group_id ORDER BY created_at DESC) AS rn
  FROM events
)
SELECT group_id, bucket, COUNT(*), COUNT(*) FILTER (WHERE ok),
       COUNT(*) FILTER (WHERE rn <= $5), COUNT(*) FILTER (WHERE rn <= $5 AND ok)
FROM ranked
GROUP BY group_id, bucket`

// 首字中位数只看成功请求；缓存命中率 = 缓存读 ÷（新输入 + 缓存读 + 缓存写）。
const serviceStatusUsageSQL = `
SELECT ul.group_id,
       percentile_cont(0.5) WITHIN GROUP (ORDER BY ul.first_token_ms)
         FILTER (WHERE ul.timing_version = 1 AND ul.first_output_kind IS NOT NULL AND ul.first_token_ms IS NOT NULL),
       COALESCE(SUM(ul.input_tokens), 0), COALESCE(SUM(ul.cache_read_tokens), 0), COALESCE(SUM(ul.cache_creation_tokens), 0)
FROM usage_logs ul
WHERE ul.created_at >= $1 AND ul.created_at < $2 AND ul.group_id IS NOT NULL
  AND ul.completion_status <> 'client_disconnected' AND COALESCE(ul.request_type, 0) <> 4
  AND NOT (ul.completion_status = 'incomplete' OR (ul.completion_status = 'unknown' AND ul.is_complete IS FALSE))
GROUP BY ul.group_id`

func (r *serviceStatusRepository) Read(ctx context.Context, start, end time.Time, bucket time.Duration, buckets, recent int) (map[int64]*service.ServiceStatusGroupStats, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	stats := map[int64]*service.ServiceStatusGroupStats{}
	group := func(id int64) *service.ServiceStatusGroupStats {
		if stats[id] == nil {
			stats[id] = &service.ServiceStatusGroupStats{Buckets: make([]service.ServiceStatusCounts, buckets)}
		}
		return stats[id]
	}

	rows, err := tx.QueryContext(ctx, serviceStatusEventsSQL, start, end, bucket.Seconds(), buckets, recent)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id int64
		var index, total, success, recentTotal, recentSuccess int
		if err := rows.Scan(&id, &index, &total, &success, &recentTotal, &recentSuccess); err != nil {
			_ = rows.Close()
			return nil, err
		}
		item := group(id)
		item.Buckets[index] = service.ServiceStatusCounts{Total: total, Success: success}
		item.Recent.Total += recentTotal
		item.Recent.Success += recentSuccess
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	rows, err = tx.QueryContext(ctx, serviceStatusUsageSQL, start, end)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id int64
		var ttft sql.NullFloat64
		var input, cacheRead, cacheWrite int64
		if err := rows.Scan(&id, &ttft, &input, &cacheRead, &cacheWrite); err != nil {
			return nil, err
		}
		item := group(id)
		if ttft.Valid {
			value := ttft.Float64
			item.TTFTP50Ms = &value
		}
		item.InputTokens, item.CacheReadTokens, item.CacheWriteTokens = input, cacheRead, cacheWrite
	}
	return stats, rows.Err()
}
