package service

import (
	"context"
	"strings"
	"time"

	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
)

const (
	AffiliateCodeTypePermanent = "permanent"
	AffiliateCodeTypeAFF       = "aff"
)

var (
	ErrAffiliateInviterChanged     = infraerrors.Conflict("AFFILIATE_INVITER_CHANGED", "返佣归属已变化，请重新打开编辑窗口")
	ErrAffiliateCodeOwnerChanged   = infraerrors.Conflict("AFFILIATE_CODE_OWNER_CHANGED", "邀请码归属已变化，请重新解析邀请码")
	ErrAffiliateInviterCycle       = infraerrors.BadRequest("AFFILIATE_INVITER_CYCLE", "不能将用户归属给自己或自己的下级")
	ErrAffiliateCodeOwnerMissing   = infraerrors.BadRequest("AFFILIATE_CODE_OWNER_MISSING", "该邀请码尚未绑定归属用户")
	ErrAffiliateInviterUnavailable = infraerrors.BadRequest("AFFILIATE_INVITER_UNAVAILABLE", "邀请码归属用户不存在或已停用")
)

type AffiliateInviterUser struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

type AffiliateInviterState struct {
	UserID      int64                 `json:"user_id"`
	Inviter     *AffiliateInviterUser `json:"inviter"`
	CodeType    string                `json:"code_type"`
	Code        string                `json:"code"`
	Version     int64                 `json:"version"`
	EffectiveAt *time.Time            `json:"effective_at"`
}

type AffiliateInviterCode struct {
	CodeType string `json:"code_type" binding:"omitempty,oneof=permanent aff"`
	Code     string `json:"code" binding:"required"`
}

type AffiliateInviterChange struct {
	CodeType        string `json:"code_type" binding:"omitempty,oneof=permanent aff"`
	Code            string `json:"code" binding:"required"`
	ResolvedUserID  int64  `json:"resolved_user_id" binding:"required,min=1"`
	ExpectedVersion *int64 `json:"expected_version" binding:"required,min=0"`
	ActorUserID     *int64 `json:"-"`
	AuthMethod      string `json:"-"`
}

func (s *AffiliateService) GetInviter(ctx context.Context, userID int64) (*AffiliateInviterState, error) {
	return s.repo.GetInviter(ctx, userID)
}

func (s *AffiliateService) ResolveInviterCode(ctx context.Context, input AffiliateInviterCode) (*AffiliateInviterUser, error) {
	return s.repo.ResolveInviterCode(ctx, AffiliateCodeTypePermanent, strings.ToUpper(strings.TrimSpace(input.Code)))
}

func (s *AffiliateService) ChangeInviter(ctx context.Context, userID int64, input *AffiliateInviterChange) error {
	input.CodeType = AffiliateCodeTypePermanent
	input.Code = strings.ToUpper(strings.TrimSpace(input.Code))
	return s.repo.ChangeInviter(ctx, userID, input)
}

// LockInviterBindings is held by the caller's credit transaction, before any
// balance rows are changed. Parent changes and credit snapshots share this order.
func (s *AffiliateService) LockInviterBindings(ctx context.Context) error {
	return s.repo.LockInviterBindings(ctx)
}

func (s *AffiliateService) CapturePaymentInvitersForRedeem(ctx context.Context, code string, userID int64) error {
	return s.repo.CapturePaymentInvitersForRedeem(ctx, code, userID, AffiliateRebateGenerations)
}
