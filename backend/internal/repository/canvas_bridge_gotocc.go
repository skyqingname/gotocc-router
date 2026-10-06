package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/service"
)

type canvasBridgeRepository struct {
	db *sql.DB
}

// NewCanvasBridgeRepository 在一个事务里写余额记录、累加团队成员额度并扣付款人余额。
func NewCanvasBridgeRepository(db *sql.DB) service.CanvasBridgeRepository {
	return &canvasBridgeRepository{db: db}
}

func (r *canvasBridgeRepository) Transfer(ctx context.Context, cmd *service.CanvasBridgeTransferCommand) (*service.CanvasBridgeTransfer, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	var id int64
	err = tx.QueryRowContext(ctx, `INSERT INTO redeem_codes (code, type, value, status, used_by, used_at, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT (code) DO NOTHING RETURNING id`,
		service.CanvasTransferCode(cmd.TransferID), service.AdjustmentTypeCanvasTransfer, -cmd.Amount,
		service.StatusUsed, cmd.PayerUserID, cmd.At, cmd.Notes).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return r.recordedTransfer(ctx, cmd.TransferID, cmd.UserID)
	}
	if err != nil {
		return nil, err
	}
	if cmd.TeamID != nil {
		if err := incrementUsageBillingTeamMember(ctx, tx, *cmd.TeamID, cmd.UserID, cmd.Amount, cmd.At); err != nil {
			return nil, err
		}
	}
	var balance float64
	err = tx.QueryRowContext(ctx, `UPDATE users SET balance = balance - $1, updated_at = NOW()
		WHERE id = $2 AND status = $3 AND deleted_at IS NULL AND balance >= $1 RETURNING balance::float8`,
		cmd.Amount, cmd.PayerUserID, service.StatusActive).Scan(&balance)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, canvasBridgePayerUnavailable(ctx, tx, cmd.PayerUserID)
	}
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &service.CanvasBridgeTransfer{TransferID: cmd.TransferID, UserID: cmd.UserID, PayerUserID: cmd.PayerUserID, Amount: cmd.Amount, PayerBalance: balance}, nil
}

func canvasBridgePayerUnavailable(ctx context.Context, tx *sql.Tx, payerUserID int64) error {
	var active bool
	if err := tx.QueryRowContext(ctx, `SELECT status = $2 AND deleted_at IS NULL FROM users WHERE id = $1`, payerUserID, service.StatusActive).Scan(&active); err != nil {
		return err
	}
	if !active {
		return service.ErrUserNotActive
	}
	return service.ErrInsufficientBalance
}

// recordedTransfer 返回已入账划转的结果，供同一划转号的重复提交使用。
func (r *canvasBridgeRepository) recordedTransfer(ctx context.Context, transferID string, userID int64) (*service.CanvasBridgeTransfer, error) {
	result := &service.CanvasBridgeTransfer{TransferID: transferID, UserID: userID}
	err := r.db.QueryRowContext(ctx, `SELECT rc.used_by, (-rc.value)::float8, u.balance::float8,
			EXISTS (SELECT 1 FROM redeem_codes reversal WHERE reversal.code = $2)
		FROM redeem_codes rc JOIN users u ON u.id = rc.used_by
		WHERE rc.code = $1 AND rc.type = $3`,
		service.CanvasTransferCode(transferID), service.CanvasTransferReversalCode(transferID), service.AdjustmentTypeCanvasTransfer).
		Scan(&result.PayerUserID, &result.Amount, &result.PayerBalance, &result.Reversed)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrCanvasTransferNotFound
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Reverse 退回一笔划转：余额加回付款人，并释放该成员在划转所在窗口内的团队额度。
func (r *canvasBridgeRepository) Reverse(ctx context.Context, transferID string, userID int64, teamID *int64) (*service.CanvasBridgeTransfer, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	result := &service.CanvasBridgeTransfer{TransferID: transferID, UserID: userID, Reversed: true}
	var transferredAt time.Time
	err = tx.QueryRowContext(ctx, `SELECT used_by, (-value)::float8, used_at FROM redeem_codes WHERE code = $1 AND type = $2 FOR UPDATE`,
		service.CanvasTransferCode(transferID), service.AdjustmentTypeCanvasTransfer).Scan(&result.PayerUserID, &result.Amount, &transferredAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrCanvasTransferNotFound
	}
	if err != nil {
		return nil, err
	}
	var id int64
	err = tx.QueryRowContext(ctx, `INSERT INTO redeem_codes (code, type, value, status, used_by, used_at, notes)
		VALUES ($1, $2, $3, $4, $5, NOW(), $6) ON CONFLICT (code) DO NOTHING RETURNING id`,
		service.CanvasTransferReversalCode(transferID), service.AdjustmentTypeCanvasTransferReversal, result.Amount,
		service.StatusUsed, result.PayerUserID, "影策画布划转退回 "+transferID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return r.recordedTransfer(ctx, transferID, userID)
	}
	if err != nil {
		return nil, err
	}
	release := &service.BatchImageBalanceHoldCommand{TeamID: teamID, ActorUserID: userID, UserID: result.PayerUserID, ReservedAt: transferredAt}
	if err := releaseBatchImageMemberAllowance(ctx, tx, release, result.Amount); err != nil && !errors.Is(err, service.ErrTeamMembershipRequired) {
		return nil, err
	}
	if err := tx.QueryRowContext(ctx, `UPDATE users SET balance = balance + $1, updated_at = NOW() WHERE id = $2 RETURNING balance::float8`,
		result.Amount, result.PayerUserID).Scan(&result.PayerBalance); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}
