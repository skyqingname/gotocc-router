package repository

import (
	"context"
	"database/sql"
	"errors"
	"math"

	"github.com/LuckyKuang/sub2api-plus/internal/service"
)

// AdjustWalletBalance changes available team credits under the team row lock and
// records the administrator in the same transaction, so concurrent usage cannot
// interleave. Personal balances and existing task holds do not move.
func (r *teamRepository) AdjustWalletBalance(ctx context.Context, teamID, adminID int64, operationID string, operation service.TeamBalanceOperation, amount float64) (*service.TeamWallet, error) {
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
	kind := "admin_" + string(operation)
	var previousActor int64
	var previousKind string
	var recordedAmount, recordedBalance float64
	err = tx.QueryRowContext(ctx, `SELECT actor_user_id,kind,amount,balance_after FROM team_wallet_entries
		WHERE team_id=$1 AND operation_id=$2 AND api_key_id=0`, teamID, operationID).
		Scan(&previousActor, &previousKind, &recordedAmount, &recordedBalance)
	if err == nil {
		// A retry repeats the same request: set compares the target, add/subtract the amount.
		same := recordedBalance == amount
		if operation != service.TeamBalanceSet {
			same = math.Abs(recordedAmount) == amount
		}
		if previousActor != adminID || previousKind != kind || !same {
			return nil, service.ErrTeamBalanceUpdateConflict
		}
		return wallet, tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	target := amount
	switch operation {
	case service.TeamBalanceAdd:
		target = service.QuantizeUsageBillingAmount(wallet.Balance + amount)
	case service.TeamBalanceSubtract:
		target = service.QuantizeUsageBillingAmount(wallet.Balance - amount)
	}
	if target < 0 {
		return nil, service.ErrTeamBalanceNegative
	}
	previousBalance := wallet.Balance
	err = tx.QueryRowContext(ctx, `UPDATE teams SET balance=$2,updated_at=NOW() WHERE id=$1 RETURNING balance`, teamID, target).Scan(&wallet.Balance)
	if err != nil {
		return nil, err
	}
	if err = recordTeamWalletMovement(ctx, tx, teamID, operationID, kind, adminID, 0,
		service.QuantizeUsageBillingAmount(target-previousBalance), 0, wallet.Balance, wallet.FrozenBalance); err != nil {
		return nil, err
	}
	return wallet, tx.Commit()
}
