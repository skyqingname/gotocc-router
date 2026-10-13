//go:build unit

package service

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/stretchr/testify/require"
	"testing"
)

type userPrioritySettings struct {
	SettingRepository
	values  map[string]string
	writes  int
	failure error
}

func (s *userPrioritySettings) GetValue(_ context.Context, key string) (string, error) {
	if s.failure != nil {
		return "", s.failure
	}
	value, ok := s.values[key]
	if !ok {
		return "", ErrSettingNotFound
	}
	return value, nil
}
func (s *userPrioritySettings) Set(_ context.Context, key, value string) error {
	if s.failure != nil {
		return s.failure
	}
	s.values[key] = value
	s.writes++
	return nil
}
func userPriorityFixture() (*AutoGroupResolver, *APIKey, *userPrioritySettings) {
	s, key, _, groups, catalog := newAutoRouteFixture()
	groups.groups[1].Platform = PlatformOpenAI
	catalog.catalog.Accounts[20] = catalog.catalog.Accounts[10]
	settings := &userPrioritySettings{values: map[string]string{
		autoGroupRoutingPolicySetting: `{"allow_user_override":true,"default_group_order":[10,20],"model_rules":[]}`,
	}}
	s.policy = NewAutoGroupRoutingPolicyService(settings, groups)
	return s, key, settings
}

// Requirement: each key may override order only while the administrator allows
// it. Turning permission off must take effect on new routing and model catalog
// resolutions without erasing preferences. Locked resources keep their group.
func TestUserRoutingPreferenceControlsResolutionAndPermissionRevocation(t *testing.T) {
	ctx := context.Background()
	s, key, settings := userPriorityFixture()
	preference := &AutoGroupRoutingPreference{DefaultGroupOrder: []int64{20, 10}}
	require.NoError(t, s.UpdateKeyRoutingPreference(ctx, 7, key.ID, preference))
	saved := settings.values[keyRoutingPreferenceSetting(key.ID)]
	check := func(want int64) {
		t.Helper()
		decision, err := s.Resolve(ctx, key, AutoRouteRequest{Model: "gpt-test", Endpoint: CompositeRouteEndpointResponses})
		require.NoError(t, err)
		require.Equal(t, want, *decision.Key.GroupID)
		models, err := s.ListModels(ctx, key, AutoRouteRequest{Endpoint: CompositeRouteEndpointResponses})
		require.NoError(t, err)
		require.Len(t, models, 1)
		require.Equal(t, want, models[0].selectedGroupID)
	}
	check(20)
	locked := int64(10)
	decision, err := s.Resolve(ctx, key, AutoRouteRequest{Model: "gpt-test", Endpoint: CompositeRouteEndpointResponses, RequiredGroupID: &locked})
	require.NoError(t, err)
	require.Equal(t, int64(10), *decision.Key.GroupID)
	settings.values[autoGroupRoutingPolicySetting] = `{"allow_user_override":false,"default_group_order":[10,20],"model_rules":[]}`
	check(10)
	writes := settings.writes
	require.ErrorIs(t, s.UpdateKeyRoutingPreference(ctx, 7, key.ID, nil), ErrUserRoutingOverrideDisabled)
	require.Equal(t, writes, settings.writes)
	require.Equal(t, saved, settings.values[keyRoutingPreferenceSetting(key.ID)])
	settings.values[autoGroupRoutingPolicySetting] = `{"allow_user_override":true,"default_group_order":[10,20],"model_rules":[]}`
	check(20)
	require.NoError(t, s.UpdateKeyRoutingPreference(ctx, 7, key.ID, nil))
	check(10)
}

// Requirement: customization must not grant group access or edit another key.
func TestUserRoutingPreferenceRejectsUnauthorizedWritesWithoutSideEffects(t *testing.T) {
	ctx := context.Background()
	s, key, settings := userPriorityFixture()
	_, err := s.GetKeyRoutingPreference(ctx, 8, key.ID)
	require.ErrorIs(t, err, ErrAPIKeyNotFound)
	require.ErrorIs(t, s.UpdateKeyRoutingPreference(ctx, 8, key.ID, nil), ErrAPIKeyNotFound)
	for _, preference := range []*AutoGroupRoutingPreference{
		{DefaultGroupOrder: []int64{999}},
		{DefaultGroupOrder: []int64{10, 10}},
		{ModelRules: []AutoGroupRoutingRule{{Model: "g*t", GroupIDs: []int64{10}}}},
		{ModelRules: []AutoGroupRoutingRule{{Model: "gpt-test", GroupIDs: []int64{999}}}},
	} {
		require.Error(t, s.UpdateKeyRoutingPreference(ctx, 7, key.ID, preference))
	}
	require.Zero(t, settings.writes)
	repo := s.keys.apiKeyRepo.(*autoRouteKeyRepo)
	repo.key.RoutingMode = APIKeyRoutingFixed
	require.Error(t, s.UpdateKeyRoutingPreference(ctx, 7, key.ID, nil))
	require.Zero(t, settings.writes)
}

func TestUserRoutingPreferenceIsPerKeyAndFailsClosed(t *testing.T) {
	ctx := context.Background()
	s, key, settings := userPriorityFixture()
	require.NoError(t, s.UpdateKeyRoutingPreference(ctx, 7, key.ID, &AutoGroupRoutingPreference{DefaultGroupOrder: []int64{20}}))
	state, err := s.GetKeyRoutingPreference(ctx, 7, key.ID)
	require.NoError(t, err)
	require.True(t, state.AllowUserOverride)
	require.Equal(t, []int64{20}, state.Preference.DefaultGroupOrder)
	other, err := s.policy.getKeyPreference(ctx, 2)
	require.NoError(t, err)
	require.Nil(t, other)
	settings.values[keyRoutingPreferenceSetting(key.ID)] = `{"default_group_order":[999,20],"model_rules":[]}`
	state, err = s.GetKeyRoutingPreference(ctx, 7, key.ID)
	require.NoError(t, err)
	require.Equal(t, []int64{20}, state.Preference.DefaultGroupOrder)
	settings.failure = errors.New("storage offline")
	_, err = s.Resolve(ctx, key, AutoRouteRequest{Model: "gpt-test", Endpoint: CompositeRouteEndpointResponses})
	require.ErrorIs(t, err, ErrAutoRouteUnavailable)
}

func TestUserRoutingOrderPrecedenceAndDefaultOff(t *testing.T) {
	// Independent expected order from the approved contract: user exact/prefix,
	// user defaults, then administrator model/default, without new candidates.
	admin := &AutoGroupRoutingPolicy{DefaultGroupOrder: []int64{1, 2, 3}, ModelRules: []AutoGroupRoutingRule{{Model: "gpt-*", GroupIDs: []int64{3}}}}
	admin.userPreference = &AutoGroupRoutingPolicy{DefaultGroupOrder: []int64{2}, ModelRules: []AutoGroupRoutingRule{{Model: "gpt-*", GroupIDs: []int64{1}}, {Model: "gpt-5", GroupIDs: []int64{3}}}}
	groups := []Group{{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}}
	for _, tt := range []struct {
		model string
		want  []int64
	}{{"other", []int64{2, 1, 3, 4}}, {"gpt-4", []int64{1, 2, 3, 4}}, {"gpt-5", []int64{3, 2, 1, 4}}} {
		got := admin.OrderGroups(groups, tt.model)
		ids := []int64{}
		for _, g := range got {
			ids = append(ids, g.ID)
		}
		require.Equal(t, tt.want, ids)
	}
	var legacy AutoGroupRoutingPolicy
	require.NoError(t, json.Unmarshal([]byte(`{"default_group_order":[1],"model_rules":[]}`), &legacy))
	require.False(t, legacy.AllowUserOverride)
}

// Creation supports rules for all currently authorized groups, including those
// which do not yet compete for a listed model. The read-only ranking stays narrow.
func TestUserRoutingCreationIncludesAllAuthorizedCandidates(t *testing.T) {
	s, groups, _ := routingPriorityFixture()
	s.policy = NewAutoGroupRoutingPolicyService(&autoPolicySettings{value: `{"allow_user_override":true,"default_group_order":[],"model_rules":[]}`}, groups)
	result, err := s.GetRoutingPriorities(context.Background(), 7, "personal")
	require.NoError(t, err)
	require.Equal(t, []int64{10, 20}, priorityGroupIDs(result.Groups))
	require.ElementsMatch(t, []int64{10, 20, 30}, priorityGroupIDs(result.AvailableGroups))
}
