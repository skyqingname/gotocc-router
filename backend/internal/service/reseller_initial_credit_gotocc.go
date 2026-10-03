package service

import (
	"context"
	"math"

	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
)

func (s *ResellerService) SetInitialCredit(ctx context.Context, ownerID int64, amount float64) (*ResellerProfile, error) {
	if _, err := s.RequireEnabled(ctx, ownerID); err != nil {
		return nil, err
	}
	if math.IsNaN(amount) || math.IsInf(amount, 0) || amount < 0 {
		return nil, infraerrors.BadRequest("INVALID_INITIAL_CREDIT", "初始额度必须为非负数")
	}
	if err := s.Repo.SetInitialCredit(ctx, ownerID, QuantizeUsageBillingAmount(amount)); err != nil {
		return nil, err
	}
	return s.Repo.Profile(ctx, ownerID)
}
