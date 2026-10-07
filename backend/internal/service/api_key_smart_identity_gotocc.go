package service

import (
	"context"
	"errors"

	"github.com/LuckyKuang/sub2api-plus/internal/config"
	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/ip"
)

// A completed resource stays readable when its creation exhausts the key.
func autoKeyAllowsResourceRead(key *APIKey) bool {
	return key != nil && !key.TeamOwnerDisabled &&
		(key.Status == StatusAPIKeyActive || key.Status == StatusAPIKeyExpired || key.Status == StatusAPIKeyQuotaExhausted)
}

func (s *AutoGroupResolver) freshAutoKey(ctx context.Context, authenticated *APIKey, enforceBilling bool) (*APIKey, error) {
	if s == nil || s.keys == nil || s.keys.apiKeyRepo == nil {
		return nil, ErrAutoRouteUnavailable
	}
	if s.cfg != nil && s.cfg.RunMode == config.RunModeSimple {
		return nil, infraerrors.Forbidden("AUTO_ROUTING_UNSUPPORTED_RUN_MODE", "automatic routing requires standard run mode")
	}
	if authenticated == nil || authenticated.ID <= 0 || !authenticated.IsAutoRouting() {
		return nil, ErrAutoRouteNoAccess
	}
	key, err := s.keys.apiKeyRepo.GetByID(ctx, authenticated.ID)
	if errors.Is(err, ErrAPIKeyNotFound) {
		return nil, ErrAutoRouteNoAccess
	}
	if err != nil {
		return nil, ErrAutoRouteUnavailable.WithCause(err)
	}
	if key == nil || key.User == nil || key.Key != authenticated.Key || key.UserID != authenticated.UserID ||
		!key.IsAutoRouting() || key.GroupID != nil ||
		(key.TeamID == nil) != (authenticated.TeamID == nil) ||
		(key.TeamID != nil && *key.TeamID != *authenticated.TeamID) {
		return nil, ErrAutoRouteContext
	}
	if enforceBilling {
		if key.IsExpired() || key.Status == StatusAPIKeyExpired {
			return nil, ErrAPIKeyExpired
		}
		if key.IsQuotaExhausted() || key.Status == StatusAPIKeyQuotaExhausted {
			return nil, ErrAPIKeyQuotaExhausted
		}
	}
	if !autoKeyAllowsResourceRead(key) {
		return nil, ErrAutoRouteNoAccess
	}
	copyKey := *key
	if _, err := s.keys.hydrateTeamAPIKey(ctx, &copyKey, nil); err != nil {
		return nil, err
	}
	if err := s.keys.ValidateTeamKeyLifecycle(&copyKey); err != nil {
		return nil, err
	}
	s.keys.compileAPIKeyIPRules(&copyKey)
	if err := s.keys.attachResellerCustomer(ctx, &copyKey); err != nil {
		return nil, err
	}
	if clientIP, ok := ctx.Value(autoRouteClientIPKey{}).(string); ok {
		if allowed, _ := ip.CheckIPRestrictionWithCompiledRules(clientIP, copyKey.CompiledIPWhitelist, copyKey.CompiledIPBlacklist); !allowed {
			return nil, ErrAutoRouteNoAccess
		}
	}
	return &copyKey, nil
}
