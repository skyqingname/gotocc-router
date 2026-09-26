//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type autoPolicySettings struct {
	SettingRepository
	value  string
	err    error
	writes int
}

func (s *autoPolicySettings) GetValue(context.Context, string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	if s.value == "" {
		return "", ErrSettingNotFound
	}
	return s.value, nil
}
func (s *autoPolicySettings) Set(_ context.Context, _ string, value string) error {
	s.value = value
	s.writes++
	return s.err
}

func TestAutoRoutingPolicyExactThenLongestPrefixThenDefault(t *testing.T) {
	policy := AutoGroupRoutingPolicy{DefaultGroupOrder: []int64{3, 1}, ModelRules: []AutoGroupRoutingRule{
		{Model: "gpt-*", GroupIDs: []int64{2}}, {Model: "gpt-5*", GroupIDs: []int64{1}}, {Model: "gpt-5", GroupIDs: []int64{3, 2}},
	}}
	groups := []Group{{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}}
	for _, tt := range []struct {
		model string
		want  []int64
	}{{"gpt-4", []int64{2, 3, 1, 4}}, {"gpt-5.1", []int64{1, 3, 2, 4}}, {"gpt-5", []int64{3, 2, 1, 4}}, {"claude", []int64{3, 1, 2, 4}}} {
		ordered := policy.OrderGroups(groups, tt.model)
		ids := make([]int64, 0, len(ordered))
		for _, group := range ordered {
			ids = append(ids, group.ID)
		}
		require.Equal(t, tt.want, ids)
	}
	require.Equal(t, int64(1), groups[0].ID)
}

func TestAutoRoutingPolicyNeverAddsUnauthorizedGroups(t *testing.T) {
	policy := AutoGroupRoutingPolicy{DefaultGroupOrder: []int64{99, 2, 1}}
	got := policy.OrderGroups([]Group{{ID: 1}, {ID: 2}}, "gpt-5")
	require.Len(t, got, 2)
	require.Equal(t, int64(2), got[0].ID)
}

func TestAutoRoutingPolicySaveValidatesBeforeWritingAndReadFailsClosed(t *testing.T) {
	repo := &autoPolicySettings{}
	groups := &autoRouteGroupRepo{groups: []Group{{ID: 1, Status: StatusActive}}}
	s := NewAutoGroupRoutingPolicyService(repo, groups)
	for _, policy := range []AutoGroupRoutingPolicy{
		{DefaultGroupOrder: []int64{1, 1}}, {DefaultGroupOrder: []int64{99}},
		{ModelRules: []AutoGroupRoutingRule{{Model: "g*t", GroupIDs: []int64{1}}}},
	} {
		require.Error(t, s.Update(context.Background(), policy))
		require.Zero(t, repo.writes)
	}
	require.NoError(t, s.Update(context.Background(), AutoGroupRoutingPolicy{DefaultGroupOrder: []int64{1}}))
	got, err := s.Get(context.Background())
	require.NoError(t, err)
	require.Equal(t, []int64{1}, got.DefaultGroupOrder)
	repo.value = "null"
	_, err = s.Get(context.Background())
	require.ErrorIs(t, err, ErrAutoRouteUnavailable)
	repo.value = "broken"
	_, err = s.Get(context.Background())
	require.ErrorIs(t, err, ErrAutoRouteUnavailable)
	repo.err = errors.New("database unavailable")
	_, err = s.Get(context.Background())
	require.ErrorIs(t, err, ErrAutoRouteUnavailable)
}

func TestAutoRoutingPolicyControlsResolverAndCatalogButNotLockedRequests(t *testing.T) {
	resolver, key, _, groups, catalog := newAutoRouteFixture()
	groups.groups[1].Platform = PlatformOpenAI
	account := catalog.catalog.Accounts[10][0]
	account.ID = 200
	catalog.catalog.Accounts[20] = []Account{account}
	settings := &autoPolicySettings{value: `{"default_group_order":[10],"model_rules":[{"model":"gpt-test","group_ids":[20]}]}`}
	resolver.policy = NewAutoGroupRoutingPolicyService(settings, groups)
	route, err := resolver.Resolve(context.Background(), key, AutoRouteRequest{Model: "gpt-test", Endpoint: CompositeRouteEndpointResponses})
	require.NoError(t, err)
	require.Equal(t, int64(20), *route.Key.GroupID)
	models, err := resolver.ListModels(context.Background(), key, AutoRouteRequest{Endpoint: CompositeRouteEndpointResponses})
	require.NoError(t, err)
	require.Len(t, models, 1)
	require.Equal(t, int64(20), models[0].selectedGroupID)
	locked := int64(10)
	settings.err = errors.New("policy storage unavailable")
	route, err = resolver.Resolve(context.Background(), key, AutoRouteRequest{Model: "gpt-test", Endpoint: CompositeRouteEndpointResponses, RequiredGroupID: &locked})
	require.NoError(t, err)
	require.Equal(t, locked, *route.Key.GroupID)
}
