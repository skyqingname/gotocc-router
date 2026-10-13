package service

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
	"github.com/google/uuid"
)

var (
	ErrCanvasCardAmount    = infraerrors.BadRequest("CANVAS_CARD_AMOUNT_INVALID", "请选择可兑换的卡密面额")
	ErrCanvasCardStock     = infraerrors.Conflict("CANVAS_CARD_OUT_OF_STOCK", "该面额卡密暂时缺货，请选择其他面额")
	ErrCanvasCardConflict  = infraerrors.Conflict("CANVAS_CARD_ORDER_CONFLICT", "该兑换单已提交，面额和扣款来源不能变更")
	ErrCanvasCardDuplicate = infraerrors.Conflict("CANVAS_CARD_DUPLICATE", "导入的卡密有重复，请检查后重新导入；本批未导入")
)

type CanvasCardStock struct {
	Denomination int `json:"denomination"`
	Available    int `json:"available"`
	Issued       int `json:"issued"`
}

type CanvasCardOrder struct {
	ID            int64     `json:"id"`
	OperationID   string    `json:"operation_id"`
	UserID        int64     `json:"user_id"`
	Denomination  int       `json:"denomination"`
	FundingSource string    `json:"funding_source"`
	TeamID        *int64    `json:"team_id"`
	BalanceAfter  float64   `json:"balance_after"`
	Code          string    `json:"code"`
	CreatedAt     time.Time `json:"created_at"`
}

type CanvasCardExchangeInput struct {
	OperationID   string `json:"operation_id" binding:"required,uuid"`
	Denomination  int    `json:"denomination" binding:"required"`
	FundingSource string `json:"funding_source" binding:"required,oneof=personal team"`
	TeamID        *int64 `json:"team_id"`
}

type CanvasCardImportInput struct {
	Denomination int      `json:"denomination" binding:"required"`
	Codes        []string `json:"codes" binding:"required,min=1"`
}

type CanvasCardInventoryItem struct {
	ID            int64      `json:"id"`
	Denomination  int        `json:"denomination"`
	Code          string     `json:"code"`
	CreatedAt     time.Time  `json:"created_at"`
	UserID        *int64     `json:"user_id"`
	FundingSource *string    `json:"funding_source"`
	TeamID        *int64     `json:"team_id"`
	IssuedAt      *time.Time `json:"issued_at"`
}

type CanvasCardOverview struct {
	Stock           []CanvasCardStock `json:"stock"`
	PersonalBalance float64           `json:"personal_balance"`
	Team            *Team             `json:"team"`
	TeamEnabled     bool              `json:"team_enabled"`
}

type CanvasCardRepository interface {
	Stock(context.Context) ([]CanvasCardStock, error)
	Import(context.Context, int64, int, []string) error
	Exchange(context.Context, int64, CanvasCardExchangeInput) (*CanvasCardOrder, error)
	History(context.Context, int64, int) ([]CanvasCardOrder, int, error)
	Inventory(context.Context, int, string, int) ([]CanvasCardInventoryItem, int, error)
}

type CanvasCardService struct {
	repo         CanvasCardRepository
	users        UserRepository
	teams        *TeamService
	redeem       *RedeemService
	balanceCache BillingCache
	authCache    APIKeyAuthCacheInvalidator
}

func NewCanvasCardService(repo CanvasCardRepository, users UserRepository, teams *TeamService, redeem *RedeemService, balanceCache BillingCache, authCache APIKeyAuthCacheInvalidator) *CanvasCardService {
	return &CanvasCardService{repo: repo, users: users, teams: teams, redeem: redeem, balanceCache: balanceCache, authCache: authCache}
}

func (s *CanvasCardService) Stock(ctx context.Context) ([]CanvasCardStock, error) {
	counts, err := s.repo.Stock(ctx)
	if err != nil {
		return nil, err
	}
	stock := make([]CanvasCardStock, 0, len(canvasCardDenominations))
	for _, amount := range canvasCardDenominations {
		item := CanvasCardStock{Denomination: amount}
		for _, count := range counts {
			if count.Denomination == amount {
				item = count
				break
			}
		}
		stock = append(stock, item)
	}
	return stock, nil
}

func (s *CanvasCardService) Overview(ctx context.Context, userID int64) (*CanvasCardOverview, error) {
	if err := s.redeem.requirePlatformFunding(ctx, userID); err != nil {
		return nil, err
	}
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	stock, err := s.Stock(ctx)
	if err != nil {
		return nil, err
	}
	out := &CanvasCardOverview{Stock: stock, PersonalBalance: user.Balance, TeamEnabled: s.teams.ensureEnabled() == nil}
	team, err := s.teams.repo.GetContextByUserID(ctx, userID)
	if errors.Is(err, ErrTeamNotFound) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	out.Team = team.Team
	return out, nil
}

func (s *CanvasCardService) Exchange(ctx context.Context, userID int64, in CanvasCardExchangeInput) (*CanvasCardOrder, error) {
	if err := s.redeem.requirePlatformFunding(ctx, userID); err != nil {
		return nil, err
	}
	if !slices.Contains(canvasCardDenominations, in.Denomination) {
		return nil, ErrCanvasCardAmount
	}
	operationID, err := uuid.Parse(in.OperationID)
	if err != nil {
		return nil, infraerrors.BadRequest("CANVAS_CARD_OPERATION_INVALID", "兑换单号无效")
	}
	in.OperationID = operationID.String()
	switch in.FundingSource {
	case "personal":
		if in.TeamID != nil {
			return nil, infraerrors.BadRequest("CANVAS_CARD_SOURCE_INVALID", "个人余额兑换不应指定团队")
		}
	case "team":
		if err := s.teams.ensureEnabled(); err != nil {
			return nil, err
		}
		if in.TeamID == nil || *in.TeamID <= 0 {
			return nil, ErrTeamMembershipRequired
		}
	default:
		return nil, infraerrors.BadRequest("CANVAS_CARD_SOURCE_INVALID", "请选择个人余额或团队余额")
	}
	out, err := s.repo.Exchange(ctx, userID, in)
	if err != nil {
		return nil, err
	}
	if out.TeamID != nil {
		s.teams.invalidateTeamKeys(ctx, *out.TeamID)
	} else {
		_ = s.balanceCache.InvalidateUserBalance(ctx, userID)
		s.authCache.InvalidateAuthCacheByUserID(ctx, userID)
	}
	return out, nil
}

func (s *CanvasCardService) Import(ctx context.Context, adminID int64, in CanvasCardImportInput) error {
	if !slices.Contains(canvasCardDenominations, in.Denomination) {
		return ErrCanvasCardAmount
	}
	if len(in.Codes) == 0 || len(in.Codes) > canvasCardImportLimit {
		return infraerrors.BadRequest("CANVAS_CARD_IMPORT_SIZE", "卡密数量超出单批导入范围")
	}
	seen := make(map[string]bool, len(in.Codes))
	for i, code := range in.Codes {
		code = strings.TrimSpace(code)
		if code == "" {
			return infraerrors.BadRequest("CANVAS_CARD_EMPTY", "卡密不能为空")
		}
		if seen[code] {
			return ErrCanvasCardDuplicate
		}
		seen[code] = true
		in.Codes[i] = code
	}
	return s.repo.Import(ctx, adminID, in.Denomination, in.Codes)
}

func (s *CanvasCardService) History(ctx context.Context, userID int64, page int) ([]CanvasCardOrder, int, error) {
	return s.repo.History(ctx, userID, page)
}

func (s *CanvasCardService) Inventory(ctx context.Context, denomination int, status string, page int) ([]CanvasCardInventoryItem, int, error) {
	if denomination != 0 && !slices.Contains(canvasCardDenominations, denomination) {
		return nil, 0, ErrCanvasCardAmount
	}
	if status != "" && status != "available" && status != "issued" {
		return nil, 0, infraerrors.BadRequest("CANVAS_CARD_STATUS_INVALID", "库存状态无效")
	}
	return s.repo.Inventory(ctx, denomination, status, page)
}
