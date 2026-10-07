package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/LuckyKuang/sub2api-plus/internal/service"
)

// SetWalletBalance adjusts available team credits and records the administrator
// in the same transaction. Personal balances and existing task holds do not move.
func (r *teamRepository) SetWalletBalance(ctx context.Context, teamID, adminID int64, operationID string, balance float64) (*service.TeamWallet, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err = lockTeamRow(ctx, tx, teamID); err != nil {
		return nil, err
	}
	wallet := &service.TeamWallet{}
	err = tx.QueryRowContext(ctx, `SELECT balance,frozen_balance,reseller_owner_id FROM teams WHERE id=$1`, teamID).
		Scan(&wallet.Balance, &wallet.FrozenBalance, &wallet.ResellerOwnerID)
	if err != nil {
		return nil, err
	}
	var previousActor int64
	var previousKind string
	var recordedBalance float64
	err = tx.QueryRowContext(ctx, `SELECT actor_user_id,kind,balance_after FROM team_wallet_entries
		WHERE team_id=$1 AND operation_id=$2 AND api_key_id=0`, teamID, operationID).
		Scan(&previousActor, &previousKind, &recordedBalance)
	if err == nil {
		if previousActor != adminID || previousKind != "admin_set" || recordedBalance != balance {
			return nil, service.ErrTeamBalanceUpdateConflict
		}
		return wallet, tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	previousBalance := wallet.Balance
	err = tx.QueryRowContext(ctx, `UPDATE teams SET balance=$2,updated_at=NOW() WHERE id=$1 RETURNING balance`, teamID, balance).Scan(&wallet.Balance)
	if err != nil {
		return nil, err
	}
	if err = recordTeamWalletMovement(ctx, tx, teamID, operationID, "admin_set", adminID, 0,
		service.QuantizeUsageBillingAmount(balance-previousBalance), 0, wallet.Balance, wallet.FrozenBalance); err != nil {
		return nil, err
	}
	return wallet, tx.Commit()
}
