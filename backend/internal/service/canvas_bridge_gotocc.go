package service

import (
	"context"
	"errors"
	"strings"
	"time"

	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/logger"
)

// 影策画布桥接：画布前置的桥接服务凭管理员 API Key 调用这些接口。GoToCC 是账号与钱包的
// 唯一来源，画布积分不足时从付款人余额单向划入画布；团队成员由团队负责人付款并计入成员额度。
const (
	AdjustmentTypeCanvasTransfer         = "canvas_transfer"
	AdjustmentTypeCanvasTransferReversal = "canvas_transfer_reversal"

	canvasTransferCodePrefix         = "CANVAS-"
	canvasTransferReversalCodePrefix = "CANVASR-"
)

var ErrCanvasTransferNotFound = infraerrors.NotFound("CANVAS_TRANSFER_NOT_FOUND", "canvas transfer not found")

type CanvasBridgeVerifyInput struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type CanvasBridgeTransferInput struct {
	TransferID string  `json:"transfer_id" binding:"required,len=24,hexadecimal"`
	UserID     int64   `json:"user_id" binding:"required"`
	Amount     float64 `json:"amount" binding:"gt=0"`
	Notes      string  `json:"notes"`
}

type CanvasBridgeReverseInput struct {
	UserID int64 `json:"user_id" binding:"required"`
}

// CanvasBridgeAccount 是画布建号与绑定所需的 GoToCC 账号快照。
type CanvasBridgeAccount struct {
	UserID           int64  `json:"user_id"`
	Email            string `json:"email"`
	Username         string `json:"username"`
	Status           string `json:"status"`
	ResellerCustomer bool   `json:"reseller_customer"`
	PayerUserID      int64  `json:"payer_user_id"`
	TeamID           *int64 `json:"team_id,omitempty"`
}

type CanvasBridgeTransfer struct {
	TransferID   string  `json:"transfer_id"`
	UserID       int64   `json:"user_id"`
	PayerUserID  int64   `json:"payer_user_id"`
	Amount       float64 `json:"amount"`
	PayerBalance float64 `json:"payer_balance"`
	Reversed     bool    `json:"reversed"`
}

type CanvasBridgeTransferCommand struct {
	TransferID  string
	UserID      int64
	PayerUserID int64
	TeamID      *int64
	Amount      float64
	Notes       string
	At          time.Time
}

type CanvasBridgeRepository interface {
	Transfer(ctx context.Context, cmd *CanvasBridgeTransferCommand) (*CanvasBridgeTransfer, error)
	Reverse(ctx context.Context, transferID string, userID int64, teamID *int64) (*CanvasBridgeTransfer, error)
}

type CanvasBridgeService struct {
	users        UserRepository
	teams        TeamRepository
	resellers    ResellerRepository
	repo         CanvasBridgeRepository
	authCache    APIKeyAuthCacheInvalidator
	billingCache BillingCache
}

func NewCanvasBridgeService(users UserRepository, teams TeamRepository, resellers ResellerRepository, repo CanvasBridgeRepository, authCache APIKeyAuthCacheInvalidator, billingCache BillingCache) *CanvasBridgeService {
	return &CanvasBridgeService{users: users, teams: teams, resellers: resellers, repo: repo, authCache: authCache, billingCache: billingCache}
}

// CanvasTransferCode 以划转号作为余额记录唯一码，同一划转重复提交只入账一次。
func CanvasTransferCode(transferID string) string {
	return canvasTransferCodePrefix + strings.ToLower(transferID)
}

func CanvasTransferReversalCode(transferID string) string {
	return canvasTransferReversalCodePrefix + strings.ToLower(transferID)
}

func (s *CanvasBridgeService) Verify(ctx context.Context, in CanvasBridgeVerifyInput) (*CanvasBridgeAccount, error) {
	user, err := s.users.GetByEmail(ctx, strings.TrimSpace(in.Email))
	if errors.Is(err, ErrUserNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	if !user.CheckPassword(in.Password) {
		return nil, ErrInvalidCredentials
	}
	customer, err := s.resellers.CustomerAccount(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	payer, err := s.payer(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	return &CanvasBridgeAccount{
		UserID: user.ID, Email: user.Email, Username: user.Username, Status: user.Status,
		ResellerCustomer: customer != nil, PayerUserID: payer.userID, TeamID: payer.teamID,
	}, nil
}

type canvasBridgePayer struct {
	userID     int64
	teamID     *int64
	teamActive bool
}

// payer 团队成员由负责人付款并计入成员额度；负责人与未入团用户由本人付款。
func (s *CanvasBridgeService) payer(ctx context.Context, userID int64) (canvasBridgePayer, error) {
	teamCtx, err := s.teams.GetContextByUserID(ctx, userID)
	if errors.Is(err, ErrTeamNotFound) {
		return canvasBridgePayer{userID: userID, teamActive: true}, nil
	}
	if err != nil {
		return canvasBridgePayer{}, err
	}
	if teamCtx.Membership.Role == TeamRoleOwner {
		return canvasBridgePayer{userID: userID, teamActive: true}, nil
	}
	teamID := teamCtx.Team.ID
	return canvasBridgePayer{userID: teamCtx.Owner.UserID, teamID: &teamID, teamActive: teamCtx.Team.Status == TeamStatusActive}, nil
}

func (s *CanvasBridgeService) Transfer(ctx context.Context, in CanvasBridgeTransferInput) (*CanvasBridgeTransfer, error) {
	user, err := s.users.GetByID(ctx, in.UserID)
	if err != nil {
		return nil, err
	}
	if user.Status != StatusActive {
		return nil, ErrUserNotActive
	}
	payer, err := s.payer(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	if !payer.teamActive {
		return nil, ErrTeamSuspended
	}
	result, err := s.repo.Transfer(ctx, &CanvasBridgeTransferCommand{
		TransferID: in.TransferID, UserID: user.ID, PayerUserID: payer.userID, TeamID: payer.teamID,
		Amount: QuantizeUsageBillingAmount(in.Amount), Notes: strings.TrimSpace(in.Notes), At: time.Now(),
	})
	if err != nil {
		return nil, err
	}
	s.invalidate(ctx, user.ID, result.PayerUserID)
	return result, nil
}

func (s *CanvasBridgeService) Reverse(ctx context.Context, transferID string, in CanvasBridgeReverseInput) (*CanvasBridgeTransfer, error) {
	payer, err := s.payer(ctx, in.UserID)
	if err != nil {
		return nil, err
	}
	result, err := s.repo.Reverse(ctx, transferID, in.UserID, payer.teamID)
	if err != nil {
		return nil, err
	}
	s.invalidate(ctx, in.UserID, result.PayerUserID)
	return result, nil
}

func (s *CanvasBridgeService) invalidate(ctx context.Context, userID, payerUserID int64) {
	s.authCache.InvalidateAuthCacheByUserID(ctx, userID)
	s.authCache.InvalidateAuthCacheByUserID(ctx, payerUserID)
	if err := s.billingCache.InvalidateUserBalance(ctx, payerUserID); err != nil {
		logger.LegacyPrintf("service.canvas_bridge", "invalidate user balance cache failed: user_id=%d err=%v", payerUserID, err)
	}
}
