package repository

import (
	"context"
	"database/sql"
	"errors"

	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
)

func (r *resellerRepository) SetInitialCredit(ctx context.Context, ownerID int64, amount float64) error {
	var id int64
	err := scanSingleRow(ctx, r.executor(ctx), `UPDATE reseller_profiles SET initial_credit=$2,updated_at=NOW()
		WHERE user_id=$1 AND enabled RETURNING user_id`, []any{ownerID, amount}, &id)
	if errors.Is(err, sql.ErrNoRows) {
		return infraerrors.Forbidden("RESELLER_DISABLED", "站长中心未开放")
	}
	return err
}
