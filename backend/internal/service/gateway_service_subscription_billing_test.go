//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
)

// TestBuildUsageBillingCommand_SubscriptionAppliesRateMultiplier locks in the fix
// that subscription-mode billing honours the group (and any user-specific) rate
// multiplier — i.e. cmd.SubscriptionCost tracks ActualCost (= TotalCost *
// RateMultiplier), not raw TotalCost.
func TestBuildUsageBillingCommand_SubscriptionAppliesRateMultiplier(t *testing.T) {
	t.Parallel()

	groupID := int64(7)
	subID := int64(42)

	tests := []struct {
		name           string
		totalCost      float64
		actualCost     float64
		isSubscription bool
		wantSub        float64
		wantBalance    float64
	}{
		{
			name:           "subscription with 2x multiplier consumes 2x quota",
			totalCost:      1.0,
			actualCost:     2.0,
			isSubscription: true,
			wantSub:        2.0,
			wantBalance:    0,
		},
		{
			name:           "subscription with 0.5x multiplier consumes 0.5x quota",
			totalCost:      1.0,
			actualCost:     0.5,
			isSubscription: true,
			wantSub:        0.5,
			wantBalance:    0,
		},
		{
			name:           "free subscription (multiplier 0) consumes no quota",
			totalCost:      1.0,
			actualCost:     0,
			isSubscription: true,
			wantSub:        0,
			wantBalance:    0,
		},
		{
			name:           "balance billing keeps using ActualCost (regression)",
			totalCost:      1.0,
			actualCost:     2.0,
			isSubscription: false,
			wantSub:        0,
			wantBalance:    2.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := &postUsageBillingParams{
				Cost:               &CostBreakdown{TotalCost: tt.totalCost, ActualCost: tt.actualCost},
				User:               &User{ID: 1},
				APIKey:             &APIKey{ID: 2, GroupID: &groupID},
				Account:            &Account{ID: 3},
				Subscription:       &UserSubscription{ID: subID},
				IsSubscriptionBill: tt.isSubscription,
			}

			cmd := buildUsageBillingCommand("req-1", nil, p)
			if cmd == nil {
				t.Fatal("buildUsageBillingCommand returned nil")
			}
			if cmd.SubscriptionCost != tt.wantSub {
				t.Errorf("SubscriptionCost = %v, want %v", cmd.SubscriptionCost, tt.wantSub)
			}
			if cmd.BalanceCost != tt.wantBalance {
				t.Errorf("BalanceCost = %v, want %v", cmd.BalanceCost, tt.wantBalance)
			}
		})
	}
}

func TestBuildUsageBillingCommand_PreservesTeamActorAndBillingOwner(t *testing.T) {
	teamID := int64(7)
	owner := &User{ID: 101}
	actor := &User{ID: 202}
	p := &postUsageBillingParams{
		Cost:    &CostBreakdown{TotalCost: 1, ActualCost: 1},
		User:    owner,
		APIKey:  &APIKey{ID: 303, UserID: actor.ID, TeamID: &teamID, User: owner, ActorUser: actor},
		Account: &Account{ID: 404},
	}

	cmd := buildUsageBillingCommand("req-team-attribution", nil, p)
	if cmd == nil {
		t.Fatal("buildUsageBillingCommand returned nil")
	}
	if cmd.UserID != owner.ID {
		t.Fatalf("UserID = %d, want billing owner %d", cmd.UserID, owner.ID)
	}
	if cmd.ActorUserID != actor.ID {
		t.Fatalf("ActorUserID = %d, want actor %d", cmd.ActorUserID, actor.ID)
	}
	if cmd.TeamID == nil || *cmd.TeamID != teamID {
		t.Fatalf("TeamID = %v, want %d", cmd.TeamID, teamID)
	}
}

func TestApplyUsageBilling_TeamKeyFailsClosedWithoutAtomicRepository(t *testing.T) {
	teamID := int64(7)
	p := &postUsageBillingParams{
		Cost:    &CostBreakdown{TotalCost: 1, ActualCost: 1},
		User:    &User{ID: 101},
		APIKey:  &APIKey{ID: 303, UserID: 202, TeamID: &teamID, ActorUser: &User{ID: 202}},
		Account: &Account{ID: 404},
	}

	applied, err := applyUsageBilling(context.Background(), "req-team-no-repo", nil, p, &billingDeps{}, nil)
	if applied {
		t.Fatal("team billing must not report applied without the atomic repository")
	}
	if !errors.Is(err, ErrTeamBillingUnavailable) {
		t.Fatalf("error = %v, want %v", err, ErrTeamBillingUnavailable)
	}
}
