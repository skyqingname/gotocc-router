package service

import (
	"context"

	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
)

// Aggregate membership follows the same actor, current billing owner and wallet
// as single-group and smart keys. Refreshing grants must not replace a team
// wallet or a reseller customer account with the creator's personal balance.
func (s *APIKeyService) aggregateRequestIdentity(ctx context.Context, key *APIKey) (*APIKey, error) {
	actor, err := s.userRepo.GetByID(ctx, key.UserID)
	if err != nil {
		return nil, err
	}
	if !actor.IsActive() {
		return nil, ErrInsufficientPerms
	}
	out := *key
	out.User, out.ActorUser = actor, actor
	out.ResellerPrices = nil
	if _, err := s.hydrateTeamAPIKey(ctx, &out, nil); err != nil {
		return nil, err
	}
	if err := s.attachResellerCustomer(ctx, &out); err != nil {
		return nil, err
	}
	if err := s.ValidateTeamKeyLifecycle(&out); err != nil {
		return nil, err
	}
	if !out.User.IsActive() {
		return nil, ErrInsufficientPerms
	}
	return &out, nil
}

func validateAggregateRoutingMode(bindingMode, routingMode string) error {
	if NormalizeGroupBindingMode(bindingMode) != GroupBindingSingle && routingMode == APIKeyRoutingAuto {
		return infraerrors.BadRequest("API_KEY_GROUP_BINDING_INVALID", "Aggregate and smart routing are separate key modes")
	}
	return nil
}
