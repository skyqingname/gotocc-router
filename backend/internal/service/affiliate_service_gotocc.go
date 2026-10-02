package service

import (
	"context"
	"math"
)

type AffiliateRebateSnapshot struct {
	SourceType  string
	Level       int
	RatePercent float64
	BaseAmount  float64
}

// AccrueInviteRebateForRedeem runs inside the transaction that consumes a
// standalone balance code. Payment-backed codes retain order fulfillment's rebate.
func (s *AffiliateService) AccrueInviteRebateForRedeem(ctx context.Context, inviteeUserID int64, baseRechargeAmount float64) (float64, error) {
	return s.accrueInviteRebate(ctx, inviteeUserID, baseRechargeAmount, nil, "redeem_code")
}

func (s *AffiliateService) accrueInviteRebate(ctx context.Context, inviteeUserID int64, baseRechargeAmount float64, sourceOrderID *int64, sourceType string) (float64, error) {
	if s == nil || s.repo == nil {
		return 0, nil
	}
	if inviteeUserID <= 0 || baseRechargeAmount <= 0 || math.IsNaN(baseRechargeAmount) || math.IsInf(baseRechargeAmount, 0) {
		return 0, nil
	}
	// 总开关关闭时，新充值不再产生返利
	if !s.IsEnabled(ctx) {
		return 0, nil
	}

	if err := s.repo.LockInviterBindings(ctx); err != nil {
		return 0, err
	}
	rates, err := s.settingService.GetAffiliateRebateRates(ctx)
	if err != nil {
		return 0, err
	}
	var inviters []int64
	if sourceOrderID != nil {
		inviters, err = s.repo.GetPaymentInviters(ctx, *sourceOrderID)
	} else {
		inviters, err = s.repo.GetInviterChain(ctx, inviteeUserID, AffiliateRebateGenerations)
	}
	if err != nil {
		return 0, err
	}
	freezeHours := s.settingService.GetAffiliateRebateFreezeHours(ctx)

	// LC-024: each generation is judged on its own. A beneficiary without an
	// approved agent identity simply receives nothing at this level; the rest of
	// the chain is unaffected and nothing is redistributed to another level.
	var eligible map[int64]bool
	if len(inviters) > 0 {
		eligible, err = s.agents.EligibleAmong(ctx, inviters)
		if err != nil {
			return 0, err
		}
	}

	total := 0.0
	for i, rate := range rates {
		if i >= len(inviters) {
			break
		}
		inviterID := inviters[i]
		if !eligible[inviterID] {
			continue
		}
		if _, err := s.repo.EnsureUserAffiliate(ctx, inviterID); err != nil {
			return 0, err
		}
		rebate := roundTo(baseRechargeAmount*rate/100, 8)
		if rebate <= 0 {
			continue
		}
		applied, err := s.repo.AccrueQuota(ctx, inviterID, inviteeUserID, rebate, freezeHours, sourceOrderID,
			AffiliateRebateSnapshot{SourceType: sourceType, Level: i + 1, RatePercent: rate, BaseAmount: baseRechargeAmount})
		if err != nil {
			return 0, err
		}
		if applied {
			total = roundTo(total+rebate, 8)
		}
	}
	return total, nil
}
