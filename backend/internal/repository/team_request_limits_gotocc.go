package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/LuckyKuang/sub2api-plus/internal/service"
)

// GetConcurrency reads persisted capacity without current membership or soft-
// deletion filters. New requests already pass team lifecycle authentication;
// accepted durable tasks must still recover results and settle after dissolution.
func (r *teamRepository) GetConcurrency(ctx context.Context, teamID int64) (int, error) {
	var limit int
	err := r.db.QueryRowContext(ctx, `SELECT concurrency FROM teams WHERE id = $1`, teamID).Scan(&limit)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, service.ErrTeamNotFound
	}
	return limit, err
}
