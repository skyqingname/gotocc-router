package service

import (
	"context"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/logger"
)

// LC-024 agent enrollment states. An absent row means "never applied", which is
// why AgentStatusNone exists alongside the persisted values.
const (
	AgentStatusNone     = ""
	AgentStatusApproved = "approved"
)

const (
	AgentSourceApplied       = "applied"
	AgentSourceGrandfathered = "grandfathered"
)

// SettingKeyAgentEnrollmentCutoff stores the moment self-service enrollment
// became effective. Users registered before it are grandfathered agents.
const SettingKeyAgentEnrollmentCutoff = "agent_enrollment_cutoff"

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
// asks only whether a beneficiary has an active membership.
type AgentEligibility interface {
	EligibleAmong(ctx context.Context, userIDs []int64) (map[int64]bool, error)
}

type AgentRepository interface {
	AgentEligibility
	Overview(ctx context.Context, userID int64) (*AgentProfile, error)
	Apply(ctx context.Context, userID int64) (bool, error)
	List(ctx context.Context, search string, page, size int) ([]AgentApplication, int64, error)
	EnsureCutoff(ctx context.Context, sentinel string) (bool, error)
}

type AgentService struct {
	repo     AgentRepository
	settings *SettingService
}

func NewAgentService(repo AgentRepository, settings *SettingService) *AgentService {
	return &AgentService{repo: repo, settings: settings}
}

// Initialize eligibility before any affiliate/payment worker can process credits.
func ProvideAgentEligibility(repo AgentRepository) (AgentEligibility, error) {
	ctx, cancel := context.WithTimeout(context.Background(), AgentEnrollmentTimeout)
	defer cancel()
	applied, err := repo.EnsureCutoff(ctx, AgentEnrollmentCutoffSentinel)
	if err != nil {
		return nil, err
	}
	if applied {
		logger.LegacyPrintf("service.agent", "Agent enrollment cutoff initialized; existing users retained")
	}
	return repo, nil
}

func (s *AgentService) Overview(ctx context.Context, userID int64) (*AgentProfile, error) {
	return s.repo.Overview(ctx, userID)
}

// Apply records the user's intent and activates membership in one write.
// Repeated requests retain the original effective time; membership has no exit.
func (s *AgentService) Apply(ctx context.Context, userID int64) (*AgentProfile, error) {
	if !s.settings.IsAffiliateEnabled(ctx) {
		return nil, ErrAffiliateDisabled
	}
	if _, err := s.repo.Apply(ctx, userID); err != nil {
		return nil, err
	}
	return s.repo.Overview(ctx, userID)
}

func (s *AgentService) List(ctx context.Context, search string, page, size int) ([]AgentApplication, int64, error) {
	return s.repo.List(ctx, search, page, size)
}
