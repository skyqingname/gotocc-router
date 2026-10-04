package service

import (
	"context"
	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
)

var ErrResellerFundingViaOwner = infraerrors.Forbidden("RESELLER_FUNDING_VIA_OWNER", "请联系所属站长购买额度或申请退款")

func (s *RedeemService) requirePlatformFunding(ctx context.Context, userID int64) error {
	account, err := s.billingCacheService.resellerRepo.CustomerAccount(ctx, userID)
	if err != nil {
		return err
	}
	if account != nil {
		return ErrResellerFundingViaOwner
	}
	return nil
}
