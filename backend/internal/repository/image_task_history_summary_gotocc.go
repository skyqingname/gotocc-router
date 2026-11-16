package repository

import (
	"context"
	"strconv"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/service"
)

// Summary counts one user's async image tasks with the console list filters
// (status, model, [since, until)), using the user/created_at index of migration 338.
func (r *imageTaskHistoryRepository) Summary(ctx context.Context, userID int64, filter service.ImageTaskHistoryFilter) (*service.ImageTaskHistorySummary, error) {
	where := "user_id = $1"
	args := []any{userID}
	if filter.Status != "" {
		args = append(args, filter.Status)
		where += " AND status = $" + strconv.Itoa(len(args))
	}
	if filter.Model != "" {
		args = append(args, filter.Model)
		where += " AND model = $" + strconv.Itoa(len(args))
	}
	if filter.Since > 0 {
		args = append(args, time.Unix(filter.Since, 0).UTC())
		where += " AND created_at >= $" + strconv.Itoa(len(args))
	}
	if filter.Until > 0 {
		args = append(args, time.Unix(filter.Until, 0).UTC())
		where += " AND created_at < $" + strconv.Itoa(len(args))
	}
	summary := &service.ImageTaskHistorySummary{}
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*),
		COUNT(*) FILTER (WHERE status = $`+strconv.Itoa(len(args)+1)+`),
		COUNT(*) FILTER (WHERE status = $`+strconv.Itoa(len(args)+2)+`),
		COUNT(*) FILTER (WHERE status = $`+strconv.Itoa(len(args)+3)+`)
		FROM async_image_tasks WHERE `+where,
		append(args, service.ImageTaskStatusCompleted, service.ImageTaskStatusProcessing, service.ImageTaskStatusFailed)...).
		Scan(&summary.Total, &summary.Completed, &summary.Processing, &summary.Failed)
	if err != nil {
		return nil, err
	}
	return summary, nil
}
