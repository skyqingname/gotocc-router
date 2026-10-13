package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/service"
)

type canvasCardRepository struct{ db *sql.DB }

func NewCanvasCardRepository(db *sql.DB) service.CanvasCardRepository {
	return &canvasCardRepository{db: db}
}

func (r *canvasCardRepository) Stock(ctx context.Context) ([]service.CanvasCardStock, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT denomination,
		COUNT(*) FILTER (WHERE issued_order_id IS NULL), COUNT(*) FILTER (WHERE issued_order_id IS NOT NULL)
		FROM canvas_card_inventory GROUP BY denomination`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []service.CanvasCardStock{}
	for rows.Next() {
		var item service.CanvasCardStock
		if err := rows.Scan(&item.Denomination, &item.Available, &item.Issued); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *canvasCardRepository) Import(ctx context.Context, adminID int64, denomination int, codes []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, code := range codes {
		var id int64
		err := tx.QueryRowContext(ctx, `INSERT INTO canvas_card_inventory(denomination,code,imported_by)
			VALUES($1,$2,$3) ON CONFLICT(code) DO NOTHING RETURNING id`, denomination, code, adminID).Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			return service.ErrCanvasCardDuplicate
		}
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *canvasCardRepository) Exchange(ctx context.Context, userID int64, in service.CanvasCardExchangeInput) (*service.CanvasCardOrder, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	out := &service.CanvasCardOrder{UserID: userID, OperationID: in.OperationID, Denomination: in.Denomination, FundingSource: in.FundingSource, TeamID: in.TeamID}
	// A duplicate waits for the first transaction and reads its issued card.
	err = tx.QueryRowContext(ctx, `INSERT INTO canvas_card_orders(user_id,operation_id,denomination,funding_source,team_id)
		VALUES($1,$2,$3,$4,$5) ON CONFLICT(user_id,operation_id) DO NOTHING RETURNING id,created_at`,
		userID, in.OperationID, in.Denomination, in.FundingSource, in.TeamID).Scan(&out.ID, &out.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		var previousTeam sql.NullInt64
		err = tx.QueryRowContext(ctx, `SELECT o.id,o.denomination,o.funding_source,o.team_id,o.balance_after,o.created_at,i.code
			FROM canvas_card_orders o JOIN canvas_card_inventory i ON i.issued_order_id=o.id
			WHERE o.user_id=$1 AND o.operation_id=$2`, userID, in.OperationID).
			Scan(&out.ID, &out.Denomination, &out.FundingSource, &previousTeam, &out.BalanceAfter, &out.CreatedAt, &out.Code)
		if err != nil {
			return nil, err
		}
		if out.Denomination != in.Denomination || out.FundingSource != in.FundingSource || previousTeam.Valid != (in.TeamID != nil) || (previousTeam.Valid && previousTeam.Int64 != *in.TeamID) {
			return nil, service.ErrCanvasCardConflict
		}
		return out, tx.Commit()
	}
	if err != nil {
		return nil, err
	}
	var cardID int64
	err = tx.QueryRowContext(ctx, `SELECT id,code FROM canvas_card_inventory WHERE denomination=$1 AND issued_order_id IS NULL
		ORDER BY id LIMIT 1 FOR UPDATE SKIP LOCKED`, in.Denomination).Scan(&cardID, &out.Code)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrCanvasCardStock
	}
	if err != nil {
		return nil, err
	}
	amount := float64(in.Denomination)
	if in.FundingSource == "team" {
		err = debitCanvasCardTeam(ctx, tx, userID, in, out, amount)
	} else {
		err = debitCanvasCardPersonal(ctx, tx, userID, out, amount)
	}
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE canvas_card_inventory SET issued_order_id=$2 WHERE id=$1`, cardID, out.ID); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE canvas_card_orders SET balance_after=$2 WHERE id=$1`, out.ID, out.BalanceAfter); err != nil {
		return nil, err
	}
	return out, tx.Commit()
}

func debitCanvasCardPersonal(ctx context.Context, tx *sql.Tx, userID int64, out *service.CanvasCardOrder, amount float64) error {
	err := tx.QueryRowContext(ctx, `UPDATE users SET balance=balance-$2,updated_at=NOW()
		WHERE id=$1 AND status='active' AND deleted_at IS NULL AND balance >= $2
		AND NOT EXISTS(SELECT 1 FROM reseller_customers WHERE user_id=$1) RETURNING balance`, userID, amount).Scan(&out.BalanceAfter)
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrInsufficientBalance
	}
	if err != nil {
		return err
	}
	code, err := service.GenerateRedeemCode()
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO redeem_codes(code,type,value,status,used_by,used_at,notes)
		VALUES($1,'canvas_card',$2,'used',$3,NOW(),$4)`, code, -amount, userID, fmt.Sprintf("画布卡密兑换 #%d · 面额 %d", out.ID, out.Denomination))
	return err
}

func debitCanvasCardTeam(ctx context.Context, tx *sql.Tx, userID int64, in service.CanvasCardExchangeInput, out *service.CanvasCardOrder, amount float64) error {
	teamID := *in.TeamID
	if err := lockTeamRow(ctx, tx, teamID); err != nil {
		return err
	}
	var status string
	var source sql.NullInt64
	if err := tx.QueryRowContext(ctx, `SELECT status,reseller_owner_id FROM teams WHERE id=$1`, teamID).Scan(&status, &source); err != nil {
		return err
	}
	if status != service.TeamStatusActive {
		return service.ErrTeamSuspended
	}
	if source.Valid {
		return service.ErrResellerFundingViaOwner
	}
	var role string
	err := tx.QueryRowContext(ctx, `SELECT m.role FROM team_memberships m JOIN users u ON u.id=m.user_id
		WHERE m.team_id=$1 AND m.user_id=$2 AND m.left_at IS NULL AND u.status='active' AND u.deleted_at IS NULL
		AND NOT EXISTS(SELECT 1 FROM reseller_customers WHERE user_id=$2) FOR UPDATE OF m`, teamID, userID).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrTeamMembershipRequired
	}
	if err != nil {
		return err
	}
	if role == service.TeamRoleMember {
		if err := incrementUsageBillingTeamMember(ctx, tx, teamID, userID, amount, time.Now()); err != nil {
			return err
		}
	} else if role != service.TeamRoleOwner {
		return service.ErrTeamMembershipRequired
	}
	var frozen float64
	err = tx.QueryRowContext(ctx, `UPDATE teams SET balance=balance-$2,updated_at=NOW()
		WHERE id=$1 AND balance >= $2 RETURNING balance,frozen_balance`, teamID, amount).Scan(&out.BalanceAfter, &frozen)
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrTeamBalanceInsufficient
	}
	if err != nil {
		return err
	}
	return recordTeamWalletMovement(ctx, tx, teamID, "canvas-card:"+in.OperationID, "canvas_card", userID, 0, -amount, 0, out.BalanceAfter, frozen)
}

func (r *canvasCardRepository) Inventory(ctx context.Context, denomination int, status string, page int) ([]service.CanvasCardInventoryItem, int, error) {
	where := ` WHERE ($1::integer=0 OR i.denomination=$1) AND ($2::text='' OR ($2='available' AND i.issued_order_id IS NULL) OR ($2='issued' AND i.issued_order_id IS NOT NULL))`
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM canvas_card_inventory i`+where, denomination, status).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT i.id,i.denomination,i.code,i.created_at,o.user_id,o.funding_source,o.team_id,o.created_at
		FROM canvas_card_inventory i LEFT JOIN canvas_card_orders o ON o.id=i.issued_order_id`+where+` ORDER BY i.id DESC LIMIT $3 OFFSET $4`, denomination, status, service.CanvasCardPageSize, (page-1)*service.CanvasCardPageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []service.CanvasCardInventoryItem{}
	for rows.Next() {
		var item service.CanvasCardInventoryItem
		if err := rows.Scan(&item.ID, &item.Denomination, &item.Code, &item.CreatedAt, &item.UserID, &item.FundingSource, &item.TeamID, &item.IssuedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, item)
	}
	return out, total, rows.Err()
}
