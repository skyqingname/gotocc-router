package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/LuckyKuang/sub2api-plus/ent/redeemcode"
	"github.com/LuckyKuang/sub2api-plus/ent/user"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/pagination"
	"github.com/LuckyKuang/sub2api-plus/internal/service"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/lib/pq"
)

// UseShared counts one redemption of a shared code inside the redeem
// transaction. The row update serializes concurrent redeemers; the usage row
// keeps each user to one redemption per code.
func (r *redeemCodeRepository) UseShared(ctx context.Context, id, userID int64) error {
	client := clientFromContext(ctx, r.client)
	result, err := client.ExecContext(ctx, `UPDATE redeem_codes
		SET used_count = used_count + 1,
			status = CASE WHEN max_uses > 0 AND used_count + 1 >= max_uses THEN $2 ELSE status END
		WHERE id = $1 AND status = $3 AND max_uses <> 1 AND (max_uses = 0 OR used_count < max_uses)`,
		id, service.StatusUsed, service.StatusUnused)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrRedeemCodeUsed
	}
	result, err = client.ExecContext(ctx, `INSERT INTO redeem_code_usages (redeem_code_id, user_id)
		VALUES ($1, $2) ON CONFLICT (redeem_code_id, user_id) DO NOTHING`, id, userID)
	if err != nil {
		return err
	}
	if affected, err = result.RowsAffected(); err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrRedeemCodeRedeemed
	}
	return nil
}

func (r *redeemCodeRepository) ListUsages(ctx context.Context, id int64, params pagination.PaginationParams) ([]service.RedeemCodeUsage, *pagination.PaginationResult, error) {
	client := clientFromContext(ctx, r.client)
	var total int64
	rows, err := client.QueryContext(ctx, `SELECT COUNT(*) FROM redeem_code_usages WHERE redeem_code_id = $1`, id)
	if err != nil {
		return nil, nil, err
	}
	rows.Next()
	err = rows.Scan(&total)
	_ = rows.Close()
	if err != nil {
		return nil, nil, err
	}

	rows, err = client.QueryContext(ctx, `SELECT u.id, u.redeem_code_id, u.user_id, u.used_at, us.email
		FROM redeem_code_usages u JOIN users us ON us.id = u.user_id
		WHERE u.redeem_code_id = $1
		ORDER BY u.used_at DESC, u.id DESC
		LIMIT $2 OFFSET $3`, id, params.Limit(), params.Offset())
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = rows.Close() }()
	usages := make([]service.RedeemCodeUsage, 0)
	for rows.Next() {
		usage := service.RedeemCodeUsage{User: &service.User{}}
		if err := rows.Scan(&usage.ID, &usage.RedeemCodeID, &usage.UserID, &usage.UsedAt, &usage.User.Email); err != nil {
			return nil, nil, err
		}
		usage.User.ID = usage.UserID
		usages = append(usages, usage)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return usages, paginationResultFromTotal(total, params), nil
}

// redeemCodeRedeemedBy matches single-use codes recorded in used_by and shared
// codes the user redeemed through redeem_code_usages.
func redeemCodeRedeemedBy(userID int64) func(*entsql.Selector) {
	return func(s *entsql.Selector) {
		usages := entsql.Table("redeem_code_usages")
		s.Where(entsql.Or(
			entsql.EQ(s.C(redeemcode.FieldUsedBy), userID),
			entsql.In(s.C(redeemcode.FieldID), entsql.Select(usages.C("redeem_code_id")).From(usages).Where(entsql.EQ(usages.C("user_id"), userID))),
		))
	}
}

// Shared codes keep their redeemers in the usage table rather than used_by.
func redeemCodeSharedUserEmailContains(email string) func(*entsql.Selector) {
	return func(s *entsql.Selector) {
		usages := entsql.Table("redeem_code_usages")
		users := entsql.Table(user.Table)
		s.Where(entsql.In(s.C(redeemcode.FieldID),
			entsql.Select(usages.C("redeem_code_id")).From(usages).
				Join(users).On(usages.C("user_id"), users.C(user.FieldID)).
				Where(entsql.ContainsFold(users.C(user.FieldEmail), email))))
	}
}

// redeemCodeRedeemedAtDesc orders a user's history by that user's redemption
// time; shared codes keep it in redeem_code_usages.
func redeemCodeRedeemedAtDesc(userID int64) redeemcode.OrderOption {
	return func(s *entsql.Selector) {
		s.OrderExpr(entsql.Expr(fmt.Sprintf(
			"COALESCE(%s, (SELECT u.used_at FROM redeem_code_usages u WHERE u.redeem_code_id = %s AND u.user_id = %d)) DESC",
			s.C(redeemcode.FieldUsedAt), s.C(redeemcode.FieldID), userID)))
	}
}

// presentSharedRedemptions shows shared codes in a user's history as that
// user's own redemption.
func (r *redeemCodeRepository) presentSharedRedemptions(ctx context.Context, userID int64, codes []service.RedeemCode) error {
	ids := make([]int64, 0)
	positions := make(map[int64]int)
	for i := range codes {
		if codes[i].UsedBy == nil {
			ids = append(ids, codes[i].ID)
			positions[codes[i].ID] = i
		}
	}
	if len(ids) == 0 {
		return nil
	}
	rows, err := r.client.QueryContext(ctx, `SELECT redeem_code_id, used_at FROM redeem_code_usages
		WHERE user_id = $1 AND redeem_code_id = ANY($2)`, userID, pq.Array(ids))
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var codeID int64
		var usedAt time.Time
		if err := rows.Scan(&codeID, &usedAt); err != nil {
			return err
		}
		target := &codes[positions[codeID]]
		target.UsedAt = &usedAt
		target.UsedBy = &userID
		target.Status = service.StatusUsed
	}
	return rows.Err()
}
