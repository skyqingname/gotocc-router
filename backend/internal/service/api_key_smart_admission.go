package service

import (
	"context"
	"errors"

	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
)

type AutoRouteAdmission struct {
	Key          *APIKey
	Subscription *UserSubscription
}

// Admit must run after the selected group's security audit. The existing
// handler remains responsible for its one billing/RPM/concurrency admission.
func (s *AutoGroupResolver) Admit(ctx context.Context, bound *APIKey) (*AutoRouteAdmission, error) {
	groupID, locked := AutoRouteGroupID(ctx)
	if !locked || bound == nil || bound.GroupID == nil || bound.Group == nil || bound.User == nil || *bound.GroupID != groupID {
		return nil, ErrAutoRouteContext
	}
	key, err := s.FreshKey(ctx, bound)
	if err != nil {
		return nil, err
	}
	if key.TeamID != nil {
		key, err = s.keys.hydrateTeamAPIKey(ctx, key, nil)
		if err != nil {
			return nil, err
		}
	}
	if key.User == nil || key.User.ID != bound.User.ID {
		return nil, ErrAutoRouteContext
	}
	if err := s.keys.CheckTeamMemberLimits(key); err != nil {
		return nil, err
	}
	currentGroup, err := s.keys.groupRepo.GetByIDLite(ctx, groupID)
	if errors.Is(err, ErrGroupNotFound) {
		return nil, ErrAutoRouteNoAccess
	}
	if err != nil {
		return nil, ErrAutoRouteUnavailable.WithCause(err)
	}
	if currentGroup == nil || currentGroup.ID != groupID || !currentGroup.IsActive() || currentGroup.Platform != bound.Group.Platform || currentGroup.SubscriptionType != bound.Group.SubscriptionType {
		return nil, ErrAutoRouteNoAccess
	}
	if route, ok := AutoRouteDecisionFromContext(ctx); ok {
		if (route.Endpoint != "" && !autoRouteEndpointSupportsGroup(currentGroup, route.Platform, route.Endpoint)) || (route.ImageGeneration && !currentGroup.AllowImageGeneration) {
			return nil, ErrAutoRouteNoAccess
		}
	}
	payer, err := s.keys.userRepo.GetByID(ctx, bound.User.ID)
	if err != nil {
		return nil, ErrAutoRouteUnavailable.WithCause(err)
	}
	if payer == nil || payer.ID != bound.User.ID || !payer.IsActive() {
		return nil, ErrAutoRouteNoAccess
	}
	if !currentGroup.IsSubscriptionType() && !payer.CanBindGroup(groupID, currentGroup.IsExclusive) {
		return nil, ErrAutoRouteNoAccess
	}
	copyUser, copyGroup := *payer, *currentGroup
	copyUser.UserGroupRPMOverride = nil
	key.User, key.Group, key.GroupID = &copyUser, &copyGroup, &groupID
	if s.keys.userGroupRateRepo != nil {
		override, err := s.keys.userGroupRateRepo.GetRPMOverrideByUserAndGroup(ctx, payer.ID, groupID)
		if err != nil {
			return nil, ErrAutoRouteUnavailable.WithCause(err)
		}
		if override != nil {
			value := *override
			copyUser.UserGroupRPMOverride = &value
		}
	}
	admission := &AutoRouteAdmission{Key: key}
	if currentGroup.IsSubscriptionType() {
		if s.subscriptions == nil || s.keys.userSubRepo == nil {
			return nil, ErrAutoRouteUnavailable
		}
		sub, err := s.keys.userSubRepo.GetActiveByUserIDAndGroupID(ctx, payer.ID, groupID)
		if errors.Is(err, ErrSubscriptionNotFound) {
			return nil, infraerrors.Forbidden("SUBSCRIPTION_NOT_FOUND", "no active subscription found for this group")
		}
		if err != nil {
			return nil, ErrAutoRouteUnavailable.WithCause(err)
		}
		if sub == nil || sub.UserID != payer.ID || sub.GroupID != groupID {
			return nil, ErrAutoRouteNoAccess
		}
		copySub := *sub
		sub = &copySub
		maintenance, validationErr := s.subscriptions.ValidateAndCheckLimits(sub, key.Group)
		if maintenance {
			sub, err = s.subscriptions.EnsureWindowMaintenance(ctx, sub)
			if err != nil {
				return nil, ErrAutoRouteUnavailable.WithCause(err)
			}
			_, validationErr = s.subscriptions.ValidateAndCheckLimits(sub, key.Group)
		}
		if validationErr != nil {
			return nil, validationErr
		}
		admission.Subscription = sub
	} else if payer.Balance <= 0 {
		return nil, ErrInsufficientBalance
	}
	_ = s.keys.TouchLastUsed(ctx, key.ID)
	return admission, nil
}
