package service

import (
	"context"
	"math"

	"github.com/LuckyKuang/sub2api-plus/internal/config"
	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
	"github.com/google/uuid"
)

var (
	ErrTeamBalanceInsufficient   = infraerrors.Forbidden("TEAM_BALANCE_INSUFFICIENT", "团队公共余额不足，请由现任负责人转入额度")
	ErrTeamFundingSource         = infraerrors.Conflict("TEAM_FUNDING_SOURCE_MISMATCH", "团队额度来源不一致，请使用相同来源的额度")
	ErrTeamFundingConflict       = infraerrors.Conflict("TEAM_FUNDING_CONFLICT", "这笔团队转入已使用不同内容提交")
	ErrTeamBalanceUpdateConflict = infraerrors.Conflict("TEAM_BALANCE_UPDATE_CONFLICT", "这笔团队余额调整已使用不同内容提交")
)

// TeamWallet belongs to the team; an owner change never moves these funds.
type TeamWallet struct {
	Balance         float64 `json:"balance"`
	FrozenBalance   float64 `json:"frozen_balance"`
	ResellerOwnerID *int64  `json:"-"`
}

type TeamFundingResult struct {
	OperationID     string  `json:"operation_id"`
	Amount          float64 `json:"amount"`
	PersonalBalance float64 `json:"personal_balance"`
	TeamBalance     float64 `json:"team_balance"`
}

type TeamWalletRepository interface {
	GetWallet(context.Context, int64) (*TeamWallet, error)
	FundWallet(context.Context, int64, int64, string, float64) (*TeamFundingResult, error)
	SetWalletBalance(context.Context, int64, int64, string, float64) (*TeamWallet, error)
}

func (s *TeamService) AdminSetWalletBalance(ctx context.Context, teamID, adminID int64, operationID string, balance float64) (*TeamWallet, error) {
	if _, err := uuid.Parse(operationID); err != nil {
		return nil, infraerrors.BadRequest("TEAM_BALANCE_OPERATION_ID_INVALID", "团队余额调整单号无效")
	}
	if math.IsNaN(balance) || math.IsInf(balance, 0) || balance < 0 {
		return nil, infraerrors.BadRequest("TEAM_BALANCE_INVALID", "团队余额必须为非负数")
	}
	wallet, err := s.walletRepo.SetWalletBalance(ctx, teamID, adminID, operationID, QuantizeUsageBillingAmount(balance))
	if err != nil {
		return nil, err
	}
	s.invalidateTeamKeys(ctx, teamID)
	return wallet, nil
}

func ProvideTeamService(repo TeamRepository, userRepo UserRepository, emailService *EmailService, apiKeyCache APIKeyCache, inviteLimiter TeamInvitationLimiter, settings *SettingService, cfg *config.Config, wallet TeamWalletRepository, balanceCache BillingCache) *TeamService {
	svc := NewTeamService(repo, userRepo, emailService, apiKeyCache, inviteLimiter, settings, cfg)
	svc.walletRepo = wallet
	svc.balanceCache = balanceCache
	return svc
}

func (s *TeamService) FundWallet(ctx context.Context, userID int64, operationID string, amount float64) (*TeamFundingResult, error) {
	teamCtx, err := s.requireOwner(ctx, userID)
	if err != nil {
		return nil, err
	}
	if _, err := uuid.Parse(operationID); err != nil {
		return nil, infraerrors.BadRequest("TEAM_FUNDING_ID_INVALID", "团队转入单号无效")
	}
	if math.IsNaN(amount) || math.IsInf(amount, 0) || amount <= 0 {
		return nil, infraerrors.BadRequest("TEAM_FUNDING_AMOUNT_INVALID", "转入额度必须大于 0")
	}
	amount = QuantizeUsageBillingAmount(amount)
	if amount <= 0 {
		return nil, infraerrors.BadRequest("TEAM_FUNDING_AMOUNT_INVALID", "转入额度小于计费精度")
	}
	result, err := s.walletRepo.FundWallet(ctx, teamCtx.Team.ID, userID, operationID, amount)
	if err != nil {
		return nil, err
	}
	// The transfer is committed; refresh the personal balance used by personal keys.
	_ = s.balanceCache.InvalidateUserBalance(ctx, userID)
	s.invalidateTeamKeys(ctx, teamCtx.Team.ID)
	return result, nil
}

type teamOwnerMutationContextKey struct{}

func WithTeamOwnerMutation(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, teamOwnerMutationContextKey{}, userID)
}

func TeamOwnerMutationUserID(ctx context.Context) int64 {
	userID, _ := ctx.Value(teamOwnerMutationContextKey{}).(int64)
	return userID
}
