package service

import (
	"context"
	"time"

	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/pagination"
)

const (
	ReusableInvitationCodeStatusActive   = "active"
	ReusableInvitationCodeStatusDisabled = "disabled"
)

var (
	ErrReusableInvitationCodeNotFound  = infraerrors.NotFound("REUSABLE_INVITATION_CODE_NOT_FOUND", "reusable invitation code not found")
	ErrReusableInvitationCodeInvalid   = infraerrors.BadRequest("REUSABLE_INVITATION_CODE_INVALID", "invalid reusable invitation code")
	ErrReusableInvitationCodeExhausted = infraerrors.Conflict("REUSABLE_INVITATION_CODE_EXHAUSTED", "reusable invitation code exhausted")
)

type ReusableInvitationCode struct {
	ID        int64
	Code      string
	Status    string
	MaxUses   int
	UsedCount int
	ExpiresAt *time.Time
	Notes     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (c *ReusableInvitationCode) IsUsableAt(now time.Time) bool {
	if c == nil || c.Status != ReusableInvitationCodeStatusActive {
		return false
	}
	if c.ExpiresAt != nil && !c.ExpiresAt.After(now) {
		return false
	}
	return c.MaxUses == 0 || c.UsedCount < c.MaxUses
}

type ReusableInvitationCodeUse struct {
	ID         int64
	CodeID     int64
	UserID     int64
	Email      string
	AuthSource string
	UsedAt     time.Time
}

type ReusableInvitationCodeRepository interface {
	Create(ctx context.Context, code *ReusableInvitationCode) error
	GetByID(ctx context.Context, id int64) (*ReusableInvitationCode, error)
	GetByCode(ctx context.Context, code string) (*ReusableInvitationCode, error)
	GetUsableByCode(ctx context.Context, code string) (*ReusableInvitationCode, error)
	List(ctx context.Context, params pagination.PaginationParams) ([]ReusableInvitationCode, *pagination.PaginationResult, error)
	Disable(ctx context.Context, id int64) (*ReusableInvitationCode, error)
	Use(ctx context.Context, id, userID int64, email, authSource string) error
	Release(ctx context.Context, id, userID int64) error
	ListUsesByCodeID(ctx context.Context, codeID int64, limit int) ([]ReusableInvitationCodeUse, error)
}
