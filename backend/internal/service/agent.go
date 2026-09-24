package service

import (
	"context"
	"time"

	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
)

// LC-024 agent enrollment states. An absent row means "never applied", which is
// why AgentStatusNone exists alongside the persisted values.
const (
	AgentStatusNone     = ""
	AgentStatusPending  = "pending"
	AgentStatusApproved = "approved"
	AgentStatusRejected = "rejected"
)

const (
	AgentSourceApplied       = "applied"
	AgentSourceGrandfathered = "grandfathered"
)

// SettingKeyAgentEnrollmentCutoff stores the moment self-service enrollment
// became effective. Users registered before it are grandfathered agents.
const SettingKeyAgentEnrollmentCutoff = "agent_enrollment_cutoff"

var (
	ErrAgentAlreadyPending  = infraerrors.Conflict("AGENT_ALREADY_PENDING", "代理申请正在审核中")
	ErrAgentAlreadyApproved = infraerrors.Conflict("AGENT_ALREADY_APPROVED", "已是代理，无需重复申请")
	ErrAgentNotFound        = infraerrors.NotFound("AGENT_NOT_FOUND", "代理申请不存在")
	ErrAgentNotPending      = infraerrors.Conflict("AGENT_NOT_PENDING", "该申请已被处理")
)

type AgentProfile struct {
	UserID     int64      `json:"user_id"`
	Status     string     `json:"status"`
	Source     string     `json:"source,omitempty"`
	AppliedAt  *time.Time `json:"applied_at,omitempty"`
	ReviewedAt *time.Time `json:"reviewed_at,omitempty"`
}

type AgentApplication struct {
	UserID     int64      `json:"user_id"`
	Username   string     `json:"username"`
	Email      string     `json:"email"`
	CreatedAt  time.Time  `json:"created_at"`
	Status     string     `json:"status"`
	Source     string     `json:"source"`
	AppliedAt  *time.Time `json:"applied_at,omitempty"`
	ReviewedAt *time.Time `json:"reviewed_at,omitempty"`
}

// AgentEligibility is injected into AffiliateService so the rebate accrual point
// can ask one narrow question without depending on the review workflow.
type AgentEligibility interface {
	EligibleAmong(ctx context.Context, userIDs []int64) (map[int64]bool, error)
}

type AgentRepository interface {
	AgentEligibility
	Status(ctx context.Context, userID int64) (string, error)
	Overview(ctx context.Context, userID int64) (*AgentProfile, error)
	Apply(ctx context.Context, userID int64) (bool, error)
	Review(ctx context.Context, userID, adminID int64, approve bool) (bool, error)
	List(ctx context.Context, status, search string, page, size int) ([]AgentApplication, int64, error)
	EnsureCutoff(ctx context.Context, sentinel string) (bool, error)
}

type AgentService struct {
	repo AgentRepository
}

func NewAgentService(repo AgentRepository) *AgentService {
	return &AgentService{repo: repo}
}

// ProvideAgentEligibility narrows the repository to the only capability the
// rebate accrual path needs, so AffiliateService cannot reach the review
// workflow or the admin queue.
func ProvideAgentEligibility(repo AgentRepository) AgentEligibility {
	return repo
}

func (s *AgentService) Overview(ctx context.Context, userID int64) (*AgentProfile, error) {
	if s == nil || s.repo == nil {
		return nil, infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "agent service unavailable")
	}
	return s.repo.Overview(ctx, userID)
}

func (s *AgentService) Apply(ctx context.Context, userID int64) (*AgentProfile, error) {
	if s == nil || s.repo == nil {
		return nil, infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "agent service unavailable")
	}
	current, err := s.repo.Status(ctx, userID)
	if err != nil {
		return nil, err
	}
	switch current {
	case AgentStatusApproved:
		return nil, ErrAgentAlreadyApproved
	case AgentStatusPending:
		return nil, ErrAgentAlreadyPending
	}
	if _, err := s.repo.Apply(ctx, userID); err != nil {
		return nil, err
	}
	return s.repo.Overview(ctx, userID)
}

// Review is the admin decision. Only pending applications can be decided, so the
// verdict is terminal in both directions; a rejected user re-applies themselves.
func (s *AgentService) Review(ctx context.Context, userID, adminID int64, approve bool) error {
	if s == nil || s.repo == nil {
		return infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "agent service unavailable")
	}
	ok, err := s.repo.Review(ctx, userID, adminID, approve)
	if err != nil {
		return err
	}
	if !ok {
		return ErrAgentNotPending
	}
	return nil
}

func (s *AgentService) List(ctx context.Context, status, search string, page, size int) ([]AgentApplication, int64, error) {
	if s == nil || s.repo == nil {
		return nil, 0, infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "agent service unavailable")
	}
	return s.repo.List(ctx, status, search, page, size)
}

// EnsureCutoff initializes the enrollment boundary on first boot. It is
// idempotent and safe to call from every instance on every start.
func (s *AgentService) EnsureCutoff(ctx context.Context, sentinel string) (bool, error) {
	if s == nil || s.repo == nil {
		return false, nil
	}
	return s.repo.EnsureCutoff(ctx, sentinel)
}
