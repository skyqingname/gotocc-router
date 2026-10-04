package service

import (
	"context"
	"strings"
)

func (s *APIKeyService) SetTeamRepository(repo TeamRepository) {
	s.teamRepo = repo
}

func (s *APIKeyService) ValidateTeamKeyLifecycle(apiKey *APIKey) error {
	if apiKey == nil || apiKey.TeamID == nil {
		return nil
	}
	if s != nil && s.cfg != nil && !s.cfg.Team.Enabled {
		return ErrTeamFeatureDisabled
	}
	if apiKey.Team == nil || apiKey.TeamMembership == nil || apiKey.Team.ID != *apiKey.TeamID || apiKey.TeamMembership.TeamID != *apiKey.TeamID {
		return ErrTeamMembershipRequired
	}
	if apiKey.TeamMembership.UserID != apiKey.UserID || apiKey.TeamMembership.JoinedAt.After(apiKey.CreatedAt) {
		return ErrTeamMembershipRequired
	}
	if apiKey.Team.Status != TeamStatusActive {
		return ErrTeamSuspended
	}
	if apiKey.ActorUser == nil || !apiKey.ActorUser.IsActive() {
		return ErrTeamActorInactive
	}
	if apiKey.User == nil || !apiKey.User.IsActive() {
		return ErrTeamBillingOwnerInactive
	}
	return nil
}

func (s *APIKeyService) CheckTeamMemberLimits(apiKey *APIKey) error {
	if err := s.ValidateTeamKeyLifecycle(apiKey); err != nil {
		return err
	}
	return checkTeamMemberLimitSnapshot(apiKey.TeamMembership)
}

func (s *APIKeyService) GetAvailableGroupsForScope(ctx context.Context, userID int64, scope string) ([]Group, error) {
	account, err := s.resellerRepo.CustomerAccount(ctx, userID)
	if err != nil {
		return nil, err
	}
	if account != nil {
		return s.resellerAvailableGroups(ctx, userID, account)
	}

	if strings.EqualFold(strings.TrimSpace(scope), "team") {
		if s.cfg != nil && !s.cfg.Team.Enabled {
			return nil, ErrTeamFeatureDisabled
		}
		if s.teamRepo == nil {
			return nil, ErrTeamFeatureDisabled
		}
		teamCtx, err := s.teamRepo.GetContextByUserID(ctx, userID)
		if err != nil || teamCtx == nil || teamCtx.Owner == nil {
			return nil, ErrTeamMembershipRequired
		}
		return s.GetAvailableGroups(ctx, teamCtx.Owner.UserID)
	}
	return s.GetAvailableGroups(ctx, userID)
}

func (s *APIKeyService) GetUserGroupRatesForScope(ctx context.Context, userID int64, scope string) (map[int64]float64, error) {
	if !strings.EqualFold(strings.TrimSpace(scope), "team") {
		return s.GetUserGroupRates(ctx, userID)
	}
	if s.cfg != nil && !s.cfg.Team.Enabled {
		return nil, ErrTeamFeatureDisabled
	}
	if s.teamRepo == nil {
		return nil, ErrTeamFeatureDisabled
	}
	teamCtx, err := s.teamRepo.GetContextByUserID(ctx, userID)
	if err != nil || teamCtx == nil || teamCtx.Owner == nil {
		return nil, ErrTeamMembershipRequired
	}
	return s.GetUserGroupRates(ctx, teamCtx.Owner.UserID)
}
