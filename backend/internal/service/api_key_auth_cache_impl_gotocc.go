package service

import (
	"context"
	"errors"
)

func (s *APIKeyService) hydrateTeamAPIKey(ctx context.Context, apiKey *APIKey, err error) (*APIKey, error) {
	if err != nil || apiKey == nil || apiKey.TeamID == nil {
		return apiKey, err
	}
	if !apiKey.IsActive() && apiKey.Status != StatusAPIKeyExpired && apiKey.Status != StatusAPIKeyQuotaExhausted {
		return apiKey, nil
	}
	if s.cfg != nil && !s.cfg.Team.Enabled {
		return nil, ErrTeamFeatureDisabled
	}
	if s.teamRepo == nil {
		return nil, ErrTeamFeatureDisabled
	}
	teamCtx, err := s.teamRepo.GetContextByUserID(ctx, apiKey.UserID)
	if err != nil {
		if errors.Is(err, ErrTeamNotFound) {
			return nil, ErrTeamMembershipRequired
		}
		return nil, err
	}
	if teamCtx == nil || teamCtx.Team == nil || teamCtx.Owner == nil || teamCtx.Membership == nil || teamCtx.Team.ID != *apiKey.TeamID {
		return nil, ErrTeamMembershipRequired
	}
	if teamCtx.Membership.JoinedAt.After(apiKey.CreatedAt) {
		return nil, ErrTeamMembershipRequired
	}
	actor, err := s.userRepo.GetByID(ctx, apiKey.UserID)
	if err != nil {
		return nil, err
	}
	owner, err := s.userRepo.GetByID(ctx, teamCtx.Owner.UserID)
	if err != nil {
		return nil, err
	}
	apiKey.ActorUser = actor
	apiKey.User = owner
	apiKey.Team = teamCtx.Team
	apiKey.TeamMembership = teamCtx.Membership
	return apiKey, nil
}
