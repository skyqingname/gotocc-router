package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"

	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
)

// Preferences contain ordering only, never permissions or additional candidates.
type AutoGroupRoutingPreference struct {
	DefaultGroupOrder []int64                `json:"default_group_order"`
	ModelRules        []AutoGroupRoutingRule `json:"model_rules"`
}

func (p *AutoGroupRoutingPreference) policy() *AutoGroupRoutingPolicy {
	return &AutoGroupRoutingPolicy{DefaultGroupOrder: p.DefaultGroupOrder, ModelRules: p.ModelRules}
}

type KeyRoutingPreference struct {
	AllowUserOverride bool                        `json:"allow_user_override"`
	Preference        *AutoGroupRoutingPreference `json:"preference"`
	AvailableGroups   []AutoRoutePriorityGroup    `json:"available_groups"`
}

var ErrUserRoutingOverrideDisabled = infraerrors.Forbidden("USER_ROUTING_OVERRIDE_DISABLED", "The administrator has disabled custom routing priorities")

func keyRoutingPreferenceSetting(id int64) string {
	return "api_key_routing_preference:" + strconv.FormatInt(id, 10)
}

func (s *AutoGroupRoutingPolicyService) getKeyPreference(ctx context.Context, id int64) (*AutoGroupRoutingPreference, error) {
	raw, err := s.settings.GetValue(ctx, keyRoutingPreferenceSetting(id))
	if errors.Is(err, ErrSettingNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, ErrAutoRouteUnavailable.WithCause(err)
	}
	if len(raw) > 1<<20 {
		return nil, ErrAutoRouteUnavailable
	}
	var preference *AutoGroupRoutingPreference
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&preference) != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return nil, ErrAutoRouteUnavailable
	}
	if preference != nil {
		policy := preference.policy()
		if err := policy.Validate(); err != nil {
			return nil, ErrAutoRouteUnavailable.WithCause(err)
		}
		preference.DefaultGroupOrder, preference.ModelRules = policy.DefaultGroupOrder, policy.ModelRules
	}
	return preference, nil
}

func (s *AutoGroupResolver) ownedRoutingKey(ctx context.Context, userID, keyID int64) (*APIKey, error) {
	if s == nil || s.keys == nil || s.keys.apiKeyRepo == nil || s.policy == nil || s.policy.settings == nil {
		return nil, ErrAutoRouteUnavailable
	}
	key, err := s.keys.apiKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return nil, err
	}
	if key == nil || key.ID != keyID || key.UserID != userID {
		return nil, ErrAPIKeyNotFound
	}
	if key.TeamID != nil {
		return s.keys.hydrateTeamAPIKey(ctx, key, nil)
	}
	return key, nil
}

func (s *AutoGroupResolver) routingPreferenceGroups(ctx context.Context, key *APIKey) ([]Group, error) {
	scope := "personal"
	if key.TeamID != nil {
		scope = "team"
	}
	return s.keys.GetAvailableGroupsForScope(ctx, key.UserID, scope)
}

func (s *AutoGroupResolver) GetKeyRoutingPreference(ctx context.Context, userID, keyID int64) (*KeyRoutingPreference, error) {
	key, err := s.ownedRoutingKey(ctx, userID, keyID)
	if err != nil {
		return nil, err
	}
	policy, err := s.policy.Get(ctx)
	if err != nil {
		return nil, err
	}
	groups, err := s.routingPreferenceGroups(ctx, key)
	if err != nil {
		return nil, err
	}
	preference, err := s.policy.getKeyPreference(ctx, keyID)
	if err != nil {
		return nil, err
	}
	active := make(map[int64]bool)
	for _, group := range groups {
		if group.IsActive() {
			active[group.ID] = true
		}
	}
	// Do not disclose IDs of groups whose access has since been revoked. Keep
	// the stored preference intact until the owner explicitly saves a draft.
	if preference != nil {
		filter := func(ids []int64) []int64 {
			result := []int64{}
			for _, id := range ids {
				if active[id] {
					result = append(result, id)
				}
			}
			return result
		}
		preference.DefaultGroupOrder = filter(preference.DefaultGroupOrder)
		for i := range preference.ModelRules {
			preference.ModelRules[i].GroupIDs = filter(preference.ModelRules[i].GroupIDs)
		}
	}
	return &KeyRoutingPreference{AllowUserOverride: policy.AllowUserOverride, Preference: preference, AvailableGroups: priorityGroups(groups, active)}, nil
}

func (s *AutoGroupResolver) UpdateKeyRoutingPreference(ctx context.Context, userID, keyID int64, preference *AutoGroupRoutingPreference) error {
	key, err := s.ownedRoutingKey(ctx, userID, keyID)
	if err != nil {
		return err
	}
	policy, err := s.policy.Get(ctx)
	if err != nil {
		return err
	}
	if !policy.AllowUserOverride {
		return ErrUserRoutingOverrideDisabled
	}
	if !key.IsAutoRouting() {
		return infraerrors.BadRequest("AUTO_ROUTING_REQUIRED", "Custom priorities require an automatic routing key")
	}
	groups, err := s.routingPreferenceGroups(ctx, key)
	if err != nil {
		return err
	}
	if preference != nil {
		userPolicy := preference.policy()
		if err := userPolicy.Validate(); err != nil {
			return err
		}
		allowed := make(map[int64]bool, len(groups))
		for _, group := range groups {
			if group.IsActive() {
				allowed[group.ID] = true
			}
		}
		lists := [][]int64{userPolicy.DefaultGroupOrder}
		for _, rule := range userPolicy.ModelRules {
			lists = append(lists, rule.GroupIDs)
		}
		for _, list := range lists {
			for _, id := range list {
				if !allowed[id] {
					return ErrGroupNotAllowed
				}
			}
		}
		preference = &AutoGroupRoutingPreference{DefaultGroupOrder: userPolicy.DefaultGroupOrder, ModelRules: userPolicy.ModelRules}
	}
	data, err := json.Marshal(preference)
	if err != nil {
		return ErrAutoRouteUnavailable.WithCause(err)
	}
	if err := s.policy.settings.Set(ctx, keyRoutingPreferenceSetting(keyID), string(data)); err != nil {
		return ErrAutoRouteUnavailable.WithCause(err)
	}
	return nil
}
