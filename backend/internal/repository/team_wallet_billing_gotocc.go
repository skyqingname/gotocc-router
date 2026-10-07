package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/timezone"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
)

// A completed request retains the admitted membership even when its actor was
// promoted or left while it ran. Settlement must not drop an already-used charge.
func recordTeamWalletMemberUsage(ctx context.Context, tx *sql.Tx, membershipID int64, amount float64, now time.Time) error {
	result, err := tx.ExecContext(ctx, `UPDATE team_memberships SET
		daily_usage_usd=CASE WHEN daily_window_start IS NULL OR daily_window_start < $3 THEN $2 ELSE daily_usage_usd+$2 END,
		weekly_usage_usd=CASE WHEN weekly_window_start IS NULL OR weekly_window_start < $4 THEN $2 ELSE weekly_usage_usd+$2 END,
		monthly_usage_usd=CASE WHEN monthly_window_start IS NULL OR monthly_window_start < $5 THEN $2 ELSE monthly_usage_usd+$2 END,
		daily_window_start=$3,weekly_window_start=$4,monthly_window_start=$5,updated_at=$6
		WHERE id=$1`, membershipID, amount, timezone.StartOfDay(now), timezone.StartOfWeek(now), timezone.StartOfMonth(now), now)
	return requireTeamAffected(result, err)
}

func recordTeamWalletMovement(ctx context.Context, tx *sql.Tx, teamID int64, requestID, kind string, actorID, keyID int64, amount, frozen, balanceAfter, frozenAfter float64) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO team_wallet_entries
		(team_id,operation_id,kind,actor_user_id,api_key_id,amount,frozen_amount,balance_after,frozen_after)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, teamID, requestID, kind, actorID, keyID, amount, frozen, balanceAfter, frozenAfter)
	return err
}

func deductTeamWallet(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand) (float64, bool, error) {
	amount := service.QuantizeUsageBillingAmount(cmd.BalanceCost + cmd.SubscriptionCost)
	sufficient := true
	if managedReseller(cmd.ResellerSnapshot) {
		_, ok, err := deductUsageBillingBalance(ctx, tx, cmd.ResellerSnapshot.OwnerID, resellerPlatformCost(cmd.ResellerSnapshot, amount))
		if err != nil {
			return 0, false, err
		}
		sufficient = ok
	}
	var balance, frozen float64
	err := tx.QueryRowContext(ctx, `UPDATE teams SET balance=balance-$2,updated_at=NOW() WHERE id=$1 RETURNING balance,frozen_balance`, *cmd.TeamID, amount).Scan(&balance, &frozen)
	if err != nil {
		return 0, false, err
	}
	err = recordTeamWalletMovement(ctx, tx, *cmd.TeamID, cmd.RequestID, "consume", cmd.ActorUserID, cmd.APIKeyID, -amount, 0, balance, frozen)
	return balance, sufficient && balance >= 0, err
}

func reserveTeamWallet(ctx context.Context, tx *sql.Tx, cmd *service.BatchImageBalanceHoldCommand) (*service.BatchImageBalanceHoldResult, error) {
	if managedReseller(cmd.ResellerSnapshot) {
		if _, err := reserveUsageBillingBatchImageBalance(ctx, tx, resellerOwnerHoldCommand(cmd)); err != nil {
			return nil, err
		}
	}
	var balance, frozen float64
	err := tx.QueryRowContext(ctx, `UPDATE teams SET balance=balance-$2,frozen_balance=frozen_balance+$2,updated_at=NOW()
		WHERE id=$1 AND status='active' AND deleted_at IS NULL AND balance >= $2 RETURNING balance,frozen_balance`, *cmd.TeamID, cmd.HoldAmount).Scan(&balance, &frozen)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrTeamBalanceInsufficient
	}
	if err != nil {
		return nil, err
	}
	err = recordTeamWalletMovement(ctx, tx, *cmd.TeamID, cmd.RequestID, "reserve", cmd.ActorUserID, cmd.APIKeyID, -cmd.HoldAmount, cmd.HoldAmount, balance, frozen)
	return &service.BatchImageBalanceHoldResult{NewBalance: &balance, FrozenBalance: &frozen}, err
}

func captureTeamWallet(ctx context.Context, tx *sql.Tx, cmd *service.BatchImageBalanceHoldCommand) (*service.BatchImageBalanceHoldResult, error) {
	if managedReseller(cmd.ResellerSnapshot) {
		if _, err := captureUsageBillingBatchImageBalance(ctx, tx, resellerOwnerHoldCommand(cmd)); err != nil {
			return nil, err
		}
	}
	var balance, frozen float64
	err := tx.QueryRowContext(ctx, `UPDATE teams SET balance=balance+$2-$3,frozen_balance=frozen_balance-$2,updated_at=NOW()
		WHERE id=$1 AND frozen_balance >= $2 RETURNING balance,frozen_balance`, *cmd.TeamID, cmd.HoldAmount, cmd.ActualAmount).Scan(&balance, &frozen)
	if err != nil {
		return nil, err
	}
	err = recordTeamWalletMovement(ctx, tx, *cmd.TeamID, cmd.RequestID, "capture", cmd.ActorUserID, cmd.APIKeyID, cmd.HoldAmount-cmd.ActualAmount, -cmd.HoldAmount, balance, frozen)
	return &service.BatchImageBalanceHoldResult{NewBalance: &balance, FrozenBalance: &frozen}, err
}

// Both task families verify that their original hold exists before calling this.
func releaseTeamWallet(ctx context.Context, tx *sql.Tx, cmd *service.BatchImageBalanceHoldCommand) (*service.BatchImageBalanceHoldResult, error) {
	if managedReseller(cmd.ResellerSnapshot) {
		owner := resellerOwnerHoldCommand(cmd)
		var id int64
		if err := tx.QueryRowContext(ctx, `UPDATE users SET balance=balance+$2,frozen_balance=frozen_balance-$2,updated_at=NOW()
			WHERE id=$1 AND frozen_balance >= $2 RETURNING id`, owner.UserID, owner.HoldAmount).Scan(&id); err != nil {
			return nil, err
		}
	}
	var balance, frozen float64
	err := tx.QueryRowContext(ctx, `UPDATE teams SET balance=balance+$2,frozen_balance=frozen_balance-$2,updated_at=NOW()
		WHERE id=$1 AND frozen_balance >= $2 RETURNING balance,frozen_balance`, *cmd.TeamID, cmd.HoldAmount).Scan(&balance, &frozen)
	if err != nil {
		return nil, err
	}
	err = recordTeamWalletMovement(ctx, tx, *cmd.TeamID, cmd.RequestID, "release", cmd.ActorUserID, cmd.APIKeyID, cmd.HoldAmount, -cmd.HoldAmount, balance, frozen)
	return &service.BatchImageBalanceHoldResult{NewBalance: &balance, FrozenBalance: &frozen}, err
}
