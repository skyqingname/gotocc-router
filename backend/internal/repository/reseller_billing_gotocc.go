package repository

import (
	"context"
	"database/sql"
	"errors"
	dbent "github.com/LuckyKuang/sub2api-plus/ent"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/logger"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/reseller"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
)

func managedReseller(snapshot *reseller.Snapshot) bool {
	return snapshot != nil && snapshot.ManagedCredits
}

func resellerPlatformCost(snapshot *reseller.Snapshot, customerAmount float64) float64 {
	return service.QuantizeUsageBillingAmount(customerAmount / snapshot.Multiplier)
}

func recordResellerCreditMovement(ctx context.Context, tx *sql.Tx, snapshot *reseller.Snapshot, requestID string, keyID, actorID int64, model, kind string, amount, frozen, balanceAfter, frozenAfter, cost float64) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO reseller_credit_entries
		(customer_user_id,owner_user_id,operation_id,kind,amount,frozen_amount,balance_after,frozen_after,platform_cost,api_key_id,model,actor_user_id)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,NULLIF($12,0))`,
		snapshot.UserID, snapshot.OwnerID, requestID, kind, amount, frozen, balanceAfter, frozenAfter, cost, keyID, model, actorID)
	return err
}

func deductResellerCustomerBalance(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand) (float64, bool, error) {
	snapshot := cmd.ResellerSnapshot
	charged := service.QuantizeUsageBillingAmount(cmd.BalanceCost + cmd.SubscriptionCost)
	cost := resellerPlatformCost(snapshot, charged)
	_, sufficient, err := deductUsageBillingBalance(ctx, tx, snapshot.OwnerID, cost)
	if err != nil {
		return 0, false, err
	}
	var balance, frozen float64
	err = tx.QueryRowContext(ctx, `UPDATE reseller_customers SET credit_balance=credit_balance-$2
		WHERE user_id=$1 RETURNING credit_balance::float8,frozen_credit::float8`, snapshot.UserID, charged).Scan(&balance, &frozen)
	if err != nil {
		return 0, false, err
	}
	err = recordResellerCreditMovement(ctx, tx, snapshot, cmd.RequestID, cmd.APIKeyID, cmd.ActorUserID, cmd.Model, "consume", -charged, 0, balance, frozen, cost)
	return balance, sufficient && balance >= 0, err
}

func resellerOwnerHoldCommand(cmd *service.BatchImageBalanceHoldCommand) *service.BatchImageBalanceHoldCommand {
	owner := *cmd
	owner.UserID = cmd.ResellerSnapshot.OwnerID
	owner.HoldAmount = resellerPlatformCost(cmd.ResellerSnapshot, cmd.HoldAmount)
	owner.ActualAmount = resellerPlatformCost(cmd.ResellerSnapshot, cmd.ActualAmount)
	owner.ResellerSnapshot = nil
	owner.TeamWallet = false
	return &owner
}

func reserveResellerCustomerBalance(ctx context.Context, tx *sql.Tx, cmd *service.BatchImageBalanceHoldCommand) (*service.BatchImageBalanceHoldResult, error) {
	if _, err := reserveUsageBillingBatchImageBalance(ctx, tx, resellerOwnerHoldCommand(cmd)); err != nil {
		return nil, err
	}
	amount := service.QuantizeUsageBillingAmount(cmd.HoldAmount)
	var balance, frozen float64
	err := tx.QueryRowContext(ctx, `UPDATE reseller_customers SET credit_balance=credit_balance-$2,frozen_credit=frozen_credit+$2
		WHERE user_id=$1 AND credit_balance >= $2 RETURNING credit_balance::float8,frozen_credit::float8`, cmd.ResellerSnapshot.UserID, amount).Scan(&balance, &frozen)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrResellerCreditInsufficient
	}
	if err != nil {
		return nil, err
	}
	if err = recordResellerCreditMovement(ctx, tx, cmd.ResellerSnapshot, cmd.RequestID, cmd.APIKeyID, cmd.ActorUserID, cmd.Model, "reserve", -amount, amount, balance, frozen, 0); err != nil {
		return nil, err
	}
	return &service.BatchImageBalanceHoldResult{NewBalance: &balance, FrozenBalance: &frozen}, nil
}

func captureResellerCustomerBalance(ctx context.Context, tx *sql.Tx, cmd *service.BatchImageBalanceHoldCommand) (*service.BatchImageBalanceHoldResult, error) {
	if _, err := captureUsageBillingBatchImageBalance(ctx, tx, resellerOwnerHoldCommand(cmd)); err != nil {
		return nil, err
	}
	hold, actual := service.QuantizeUsageBillingAmount(cmd.HoldAmount), service.QuantizeUsageBillingAmount(cmd.ActualAmount)
	var balance, frozen float64
	err := tx.QueryRowContext(ctx, `UPDATE reseller_customers SET credit_balance=credit_balance+$2-$3,frozen_credit=frozen_credit-$2
		WHERE user_id=$1 AND frozen_credit >= $2 RETURNING credit_balance::float8,frozen_credit::float8`, cmd.ResellerSnapshot.UserID, hold, actual).Scan(&balance, &frozen)
	if err != nil {
		return nil, err
	}
	if err = recordResellerCreditMovement(ctx, tx, cmd.ResellerSnapshot, cmd.RequestID, cmd.APIKeyID, cmd.ActorUserID, cmd.Model, "capture", hold-actual, -hold, balance, frozen, resellerPlatformCost(cmd.ResellerSnapshot, actual)); err != nil {
		return nil, err
	}
	return &service.BatchImageBalanceHoldResult{NewBalance: &balance, FrozenBalance: &frozen}, nil
}

// Both image and video callers establish the original hold before releasing it.
func releaseResellerCustomerBalance(ctx context.Context, tx *sql.Tx, cmd *service.BatchImageBalanceHoldCommand) (*service.BatchImageBalanceHoldResult, error) {
	owner := resellerOwnerHoldCommand(cmd)
	var ownerID int64
	if err := tx.QueryRowContext(ctx, `UPDATE users SET balance=balance+$2,frozen_balance=frozen_balance-$2,updated_at=NOW()
		WHERE id=$1 AND frozen_balance >= $2 RETURNING id`, owner.UserID, owner.HoldAmount).Scan(&ownerID); err != nil {
		return nil, err
	}
	hold := service.QuantizeUsageBillingAmount(cmd.HoldAmount)
	var balance, frozen float64
	err := tx.QueryRowContext(ctx, `UPDATE reseller_customers SET credit_balance=credit_balance+$2,frozen_credit=frozen_credit-$2
		WHERE user_id=$1 AND frozen_credit >= $2 RETURNING credit_balance::float8,frozen_credit::float8`, cmd.ResellerSnapshot.UserID, hold).Scan(&balance, &frozen)
	if err != nil {
		return nil, err
	}
	if err = recordResellerCreditMovement(ctx, tx, cmd.ResellerSnapshot, cmd.RequestID, cmd.APIKeyID, cmd.ActorUserID, cmd.Model, "release", hold, -hold, balance, frozen, 0); err != nil {
		return nil, err
	}
	return &service.BatchImageBalanceHoldResult{NewBalance: &balance, FrozenBalance: &frozen}, nil
}

func ProvideResellerUsageBillingRepository(_ *dbent.Client, db *sql.DB, cache service.BillingCache) service.UsageBillingRepository {
	return &usageBillingRepository{db: db, resellerBalanceCache: cache}
}
func ProvideResellerVideoBillingRepository(_ *dbent.Client, db *sql.DB, cache service.BillingCache) service.OpenAIVideoBillingRepository {
	return &usageBillingRepository{db: db, resellerBalanceCache: cache}
}
func (r *usageBillingRepository) invalidateResellerBalance(ctx context.Context, snapshot *reseller.Snapshot) {
	if !managedReseller(snapshot) {
		return
	}
	if err := r.resellerBalanceCache.InvalidateUserBalance(ctx, snapshot.OwnerID); err != nil {
		logger.LegacyPrintf("repository.reseller", "reseller balance cache invalidation failed owner=%d", snapshot.OwnerID)
	}
}
