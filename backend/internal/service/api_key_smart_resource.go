package service

import "context"

type autoRouteSubscriptionKey struct{}

func (s *OpenAIGatewayService) OpenAIVideoTaskStorageAvailable() bool {
	return s != nil && s.openAIVideoTaskRepo != nil
}

func WithAutoRouteSubscription(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, autoRouteSubscriptionKey{}, id)
}

func (s *AutoGroupResolver) RestoreResourceSubscription(ctx context.Context, key *APIKey, id int64) (*UserSubscription, error) {
	if key == nil || key.Group == nil || key.User == nil {
		return nil, ErrAutoRouteContext
	}
	if !key.Group.IsSubscriptionType() {
		if id != 0 {
			return nil, ErrAutoRouteContext
		}
		return nil, nil
	}
	if id <= 0 || s == nil || s.keys == nil || s.keys.userSubRepo == nil {
		return nil, ErrAutoRouteContext
	}
	sub, err := s.keys.userSubRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if sub == nil || sub.UserID != key.User.ID || sub.GroupID != key.Group.ID {
		return nil, ErrAutoRouteNoAccess
	}
	return sub, nil
}

// RestoreGroup restores a verified resource owner binding without discovering
// another model, account, or payment source.
func (s *AutoGroupResolver) RestoreGroup(ctx context.Context, authenticated *APIKey, groupID int64, platform string) (*AutoRouteDecision, error) {
	key, err := s.FreshKey(ctx, authenticated)
	if err != nil {
		return nil, err
	}
	key, groups, err := s.eligibleGroups(ctx, key)
	if err != nil {
		return nil, err
	}
	for _, group := range groups {
		if group.ID != groupID {
			continue
		}
		if group.Platform != PlatformComposite && group.Platform != platform {
			return nil, ErrAutoRouteNoAccess
		}
		groupCopy, userCopy, keyCopy := group, *key.User, *key
		keyCopy.GroupID, keyCopy.Group, keyCopy.User = &groupID, &groupCopy, &userCopy
		return &AutoRouteDecision{Key: &keyCopy, Platform: platform}, nil
	}
	return nil, ErrAutoRouteNoAccess
}

func (s *OpenAIGatewayService) GetLiveCallForAutoKey(ctx context.Context, callID string, key *APIKey) (*LiveCallRecord, error) {
	if key == nil || !key.IsAutoRouting() || key.User == nil {
		return nil, ErrLiveIdentityMismatch
	}
	store, err := s.liveStore()
	if err != nil {
		return nil, err
	}
	record, err := store.GetLiveCall(ctx, hashLiveCallID(callID))
	if err != nil {
		return nil, err
	}
	if record == nil || record.CallID != callID || record.APIKeyID != key.ID || record.UserID != key.User.ID || record.GroupID <= 0 {
		return nil, ErrLiveIdentityMismatch
	}
	if record.Controller == LiveControllerClosed {
		return nil, ErrLiveCallNotFound
	}
	return record, nil
}
