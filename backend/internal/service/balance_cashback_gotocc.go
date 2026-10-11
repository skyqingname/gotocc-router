package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	dbent "github.com/LuckyKuang/sub2api-plus/ent"
	"github.com/LuckyKuang/sub2api-plus/ent/setting"
	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

const (
	SettingKeyBalanceCashback     = "balance_cashback"
	AdjustmentTypeBalanceCashback = "balance_cashback"
)

type BalanceCashbackSettings struct {
	Enabled       bool    `json:"enabled"`
	MinimumAmount float64 `json:"minimum_amount"`
	Percent       float64 `json:"percent"`
}

func (s BalanceCashbackSettings) validate() error {
	if math.IsNaN(s.MinimumAmount) || math.IsInf(s.MinimumAmount, 0) || s.MinimumAmount < 0 ||
		math.IsNaN(s.Percent) || math.IsInf(s.Percent, 0) || s.Percent < 0 {
		return infraerrors.BadRequest("INVALID_BALANCE_CASHBACK", "cashback minimum amount and percent must be non-negative numbers")
	}
	return nil
}

func parseBalanceCashbackSettings(raw string) (BalanceCashbackSettings, error) {
	var result BalanceCashbackSettings
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return result, fmt.Errorf("parse balance cashback settings: %w", err)
	}
	return result, result.validate()
}

func addBalanceCashbackSetting(updates map[string]string, value BalanceCashbackSettings) error {
	if err := value.validate(); err != nil {
		return err
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	updates[SettingKeyBalanceCashback] = string(raw)
	return nil
}

// creditBalanceCashback runs only inside the original credit transaction. It creates
// a second balance increment and a separate used ledger record, not another recharge.
func creditBalanceCashback(ctx context.Context, client *dbent.Client, userID int64, source *RedeemCode) (float64, error) {
	entry, err := client.Setting.Query().Where(setting.KeyEQ(SettingKeyBalanceCashback)).Only(ctx)
	if err != nil && !dbent.IsNotFound(err) {
		return 0, fmt.Errorf("read balance cashback settings: %w", err)
	}
	// Existing installations have no row until the administrator saves the setting.
	raw := balanceCashbackDefaultsJSON
	if entry != nil {
		raw = entry.Value
	}
	config, err := parseBalanceCashbackSettings(raw)
	if err != nil {
		return 0, err
	}
	amount := decimal.NewFromFloat(source.Value).Round(balanceCashbackDecimalPlaces)
	if !config.Enabled || config.Percent == 0 || !amount.GreaterThan(decimal.NewFromFloat(config.MinimumAmount)) {
		return 0, nil
	}
	bonus := amount.Mul(decimal.NewFromFloat(config.Percent)).Div(decimal.NewFromInt(100)).Round(balanceCashbackDecimalPlaces)
	if !bonus.IsPositive() {
		return 0, nil
	}
	bonusAmount, _ := bonus.Float64()
	code, err := GenerateRedeemCode()
	if err != nil {
		return 0, err
	}
	notes := fmt.Sprintf("source_record_id=%d; source_type=%s; base_amount=%s; percent=%s", source.ID, source.Type, amount.String(), decimal.NewFromFloat(config.Percent).String())
	if err := client.User.UpdateOneID(userID).AddBalance(bonusAmount).Exec(ctx); err != nil {
		return 0, fmt.Errorf("credit balance cashback: %w", err)
	}
	if err := client.RedeemCode.Create().SetCode(code).SetType(AdjustmentTypeBalanceCashback).
		SetValue(bonusAmount).SetStatus(StatusUsed).SetUsedBy(userID).SetUsedAt(time.Now()).
		SetValidityDays(0).SetNotes(notes).Exec(ctx); err != nil {
		return 0, fmt.Errorf("record balance cashback: %w", err)
	}
	return bonusAmount, nil
}

func (s *adminServiceImpl) creditAdminBalanceCashback(ctx context.Context, client *dbent.Client, userID int64, operation string, source *RedeemCode) error {
	if operation != "add" || source.Value <= 0 {
		return nil
	}
	customer, err := s.billingCacheService.resellerRepo.CustomerAccount(ctx, userID)
	if err != nil {
		return err
	}
	if customer != nil {
		return nil
	}
	_, err = creditBalanceCashback(ctx, client, userID, source)
	return err
}
