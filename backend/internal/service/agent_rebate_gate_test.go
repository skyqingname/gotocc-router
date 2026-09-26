package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// agentGateAffiliateRepoStub implements only the calls the accrual path makes.
// The embedded interface panics on anything else, which is what we want: the
// test should fail loudly if accrual starts touching other methods.
type agentGateAffiliateRepoStub struct {
	AffiliateRepository
	chain   []int64
	accrued map[int64]float64
	created []int64
}

func (s *agentGateAffiliateRepoStub) LockInviterBindings(context.Context) error { return nil }

func (s *agentGateAffiliateRepoStub) GetInviterChain(context.Context, int64, int) ([]int64, error) {
	return s.chain, nil
}

func (s *agentGateAffiliateRepoStub) EnsureUserAffiliate(_ context.Context, userID int64) (*AffiliateSummary, error) {
	s.created = append(s.created, userID)
	return &AffiliateSummary{UserID: userID}, nil
}

func (s *agentGateAffiliateRepoStub) AccrueQuota(_ context.Context, inviterID, _ int64, amount float64, _ int, _ *int64, _ ...AffiliateRebateSnapshot) (bool, error) {
	if s.accrued == nil {
		s.accrued = map[int64]float64{}
	}
	s.accrued[inviterID] += amount
	return true, nil
}

type agentGateSettingRepoStub struct{ values map[string]string }

func (s *agentGateSettingRepoStub) Get(context.Context, string) (*Setting, error) {
	return nil, ErrSettingNotFound
}

func (s *agentGateSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	value, ok := s.values[key]
	if !ok {
		return "", ErrSettingNotFound
	}
	return value, nil
}

func (s *agentGateSettingRepoStub) Set(context.Context, string, string) error { return nil }

func (s *agentGateSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	values := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			values[key] = value
		}
	}
	return values, nil
}

func (s *agentGateSettingRepoStub) SetMultiple(context.Context, map[string]string) error { return nil }
func (s *agentGateSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	return map[string]string{}, nil
}
func (s *agentGateSettingRepoStub) Delete(context.Context, string) error { return nil }

// agentEligibilityStub answers with a fixed approved set.
type agentEligibilityStub struct{ approved map[int64]bool }

func (s *agentEligibilityStub) EligibleAmong(_ context.Context, userIDs []int64) (map[int64]bool, error) {
	out := make(map[int64]bool, len(userIDs))
	for _, id := range userIDs {
		if s.approved[id] {
			out[id] = true
		}
	}
	return out, nil
}

func newAgentGateService(repo AffiliateRepository, agents AgentEligibility) *AffiliateService {
	settings := &SettingService{settingRepo: &agentGateSettingRepoStub{values: map[string]string{
		SettingKeyAffiliateEnabled: "true",
	}}}
	svc := NewAffiliateService(repo, settings, nil, nil)
	svc.agents = agents
	return svc
}

// A pending or never-applied beneficiary gets nothing at that level, and the
// level below it is still paid: the gate is per generation, not per chain.
func TestAccrueInviteRebateSkipsUnapprovedGenerationAndKeepsTheRest(t *testing.T) {
	repo := &agentGateAffiliateRepoStub{chain: []int64{11, 22, 33}}
	svc := newAgentGateService(repo, &agentEligibilityStub{approved: map[int64]bool{22: true}})

	if _, err := svc.accrueInviteRebate(context.Background(), 99, 100, nil, "payment"); err != nil {
		t.Fatalf("accrue: %v", err)
	}

	require.NotContains(t, repo.accrued, int64(11), "unapproved first generation must not be paid")
	require.Equal(t, 10.0, repo.accrued[22], "approved second generation keeps 10%%")
	require.NotContains(t, repo.accrued, int64(33), "unapproved third generation must not be paid")
}

// Without an eligibility source the historical behaviour is preserved, which is
// what a nil-injection (tests, partial wiring) must never silently break.
func TestAccrueInviteRebateWithoutEligibilitySourcePaysEveryGeneration(t *testing.T) {
	repo := &agentGateAffiliateRepoStub{chain: []int64{11, 22, 33}}
	svc := newAgentGateService(repo, nil)

	if _, err := svc.accrueInviteRebate(context.Background(), 99, 100, nil, "payment"); err != nil {
		t.Fatalf("accrue: %v", err)
	}

	require.Equal(t, 20.0, repo.accrued[11])
	require.Equal(t, 10.0, repo.accrued[22])
	require.Equal(t, 5.0, repo.accrued[33])
}

// An approved agent is paid the global rate regardless of any reseller profile:
// LC-024 removed the per-reseller rebate schedule.
func TestAccrueInviteRebateUsesGlobalRateForApprovedAgent(t *testing.T) {
	repo := &agentGateAffiliateRepoStub{chain: []int64{11}}
	svc := newAgentGateService(repo, &agentEligibilityStub{approved: map[int64]bool{11: true}})

	total, err := svc.accrueInviteRebate(context.Background(), 99, 1000, nil, "payment")
	require.NoError(t, err)
	require.Equal(t, 200.0, total, "first generation uses the global 20%% rate")
}
