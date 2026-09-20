//go:build unit

package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/LuckyKuang/sub2api-plus/internal/config"
	"github.com/stretchr/testify/require"
)

func routingPriorityFixture() (*AutoGroupResolver, *autoRouteGroupRepo, *autoRouteCatalogRepo) {
	s, _, _, groups, catalog := newAutoRouteFixture()
	groups.groups = []Group{
		{ID: 20, Name: "Second", Platform: PlatformOpenAI, Status: StatusActive, SortOrder: 0},
		{ID: 10, Name: "First", Platform: PlatformOpenAI, Status: StatusActive, SortOrder: 0},
		{ID: 30, Name: "Unique", Platform: PlatformAnthropic, Status: StatusActive, SortOrder: -1},
		{ID: 40, Name: "Private", Platform: PlatformOpenAI, Status: StatusActive, IsExclusive: true},
	}
	shared := catalog.catalog.Accounts[10]
	catalog.catalog.Accounts[30] = catalog.catalog.Accounts[20]
	catalog.catalog.Accounts[20] = shared
	catalog.catalog.Accounts[40] = shared
	return s, groups, catalog
}

func TestRoutingPrioritiesDefaultOnlyIncludesCompetingAuthorizedGroups(t *testing.T) {
	s, groups, catalog := routingPriorityFixture()
	got, err := s.GetRoutingPriorities(context.Background(), 7, "personal")
	require.NoError(t, err)
	require.Equal(t, "group_sort", got.DefaultSource)
	require.Equal(t, []AutoRoutePriorityGroup{{ID: 10, Name: "First", Platform: PlatformOpenAI}, {ID: 20, Name: "Second", Platform: PlatformOpenAI}}, got.Groups)
	require.Empty(t, got.ModelRules)
	require.Equal(t, 1, catalog.calls)
	require.Equal(t, int64(20), groups.groups[0].ID, "does not mutate shared group ordering")
	groups.groups[0].SortOrder = -2
	got, err = s.GetRoutingPriorities(context.Background(), 7, "personal")
	require.NoError(t, err)
	require.Equal(t, int64(20), got.Groups[0].ID)
}

func TestRoutingPrioritiesIndependentRuleIncludesDefaultFallback(t *testing.T) {
	s, groups, catalog := routingPriorityFixture()
	groups.groups = append(groups.groups, Group{ID: 50, Name: "Third", Platform: PlatformOpenAI, Status: StatusActive})
	catalog.catalog.Accounts[50] = catalog.catalog.Accounts[10]
	s.policy = NewAutoGroupRoutingPolicyService(&autoPolicySettings{value: `{"default_group_order":[20,10],"model_rules":[{"model":"gpt-*","group_ids":[50]},{"model":"gpt-test","group_ids":[10]}]}`}, groups)
	got, err := s.GetRoutingPriorities(context.Background(), 7, "personal")
	require.NoError(t, err)
	require.Equal(t, "administrator", got.DefaultSource)
	require.Equal(t, []int64{20, 10, 50}, priorityGroupIDs(got.Groups))
	require.Len(t, got.ModelRules, 1)
	require.Equal(t, "gpt-test", got.ModelRules[0].Model)
	require.Equal(t, "gpt-test", got.ModelRules[0].MatchedRule)
	require.Equal(t, []int64{10, 20, 50}, priorityGroupIDs(got.ModelRules[0].Groups))
	data, err := json.Marshal(got)
	require.NoError(t, err)
	require.NotContains(t, string(data), "Private")
	require.NotContains(t, string(data), "credentials")
}

func TestRoutingPrioritiesLongestPrefixAndUniqueModels(t *testing.T) {
	s, groups, _ := routingPriorityFixture()
	s.policy = NewAutoGroupRoutingPolicyService(&autoPolicySettings{value: `{"default_group_order":[],"model_rules":[{"model":"gpt-*","group_ids":[10]},{"model":"gpt-t*","group_ids":[20]},{"model":"claude-test","group_ids":[30]}]}`}, groups)
	got, err := s.GetRoutingPriorities(context.Background(), 7, "personal")
	require.NoError(t, err)
	require.Len(t, got.ModelRules, 1)
	require.Equal(t, "gpt-t*", got.ModelRules[0].MatchedRule)
	require.Equal(t, []int64{20, 10}, priorityGroupIDs(got.ModelRules[0].Groups))
}

func TestRoutingPrioritiesEmptyAndReadFailure(t *testing.T) {
	s, _, catalog := routingPriorityFixture()
	delete(catalog.catalog.Accounts, 20)
	got, err := s.GetRoutingPriorities(context.Background(), 7, "personal")
	require.NoError(t, err)
	data, err := json.Marshal(got)
	require.NoError(t, err)
	require.JSONEq(t, `{"default_source":"group_sort","groups":[],"model_rules":[]}`, string(data))
	catalog.err = errors.New("catalog unavailable")
	_, err = s.GetRoutingPriorities(context.Background(), 7, "personal")
	require.ErrorIs(t, err, ErrAutoRouteUnavailable)
	_, err = s.GetRoutingPriorities(context.Background(), 7, "invalid")
	require.Error(t, err)
}

func priorityGroupIDs(groups []AutoRoutePriorityGroup) []int64 {
	ids := make([]int64, 0, len(groups))
	for _, group := range groups {
		ids = append(ids, group.ID)
	}
	return ids
}

func TestRoutingPrioritiesIgnoresIncompatibleGroupsAndPolicyFailures(t *testing.T) {
	s, groups, _ := routingPriorityFixture()
	groups.groups[0].RequireOAuthOnly = true
	got, err := s.GetRoutingPriorities(context.Background(), 7, "personal")
	require.NoError(t, err)
	require.Empty(t, got.Groups, "API key accounts cannot compete in OAuth-only groups")
	s.policy = NewAutoGroupRoutingPolicyService(&autoPolicySettings{err: errors.New("settings offline")}, groups)
	_, err = s.GetRoutingPriorities(context.Background(), 7, "personal")
	require.ErrorIs(t, err, ErrAutoRouteUnavailable, "never invent a fallback order when policy lookup fails")
}

type priorityScopedUsers struct {
	UserRepository
	users map[int64]*User
}

func (r *priorityScopedUsers) GetByID(_ context.Context, id int64) (*User, error) {
	return r.users[id], nil
}

func TestRoutingPrioritiesTeamUsesPayerPermissions(t *testing.T) {
	s, groups, _ := routingPriorityFixture()
	s.keys.userRepo = &priorityScopedUsers{users: map[int64]*User{
		7: {ID: 7, Status: StatusActive},
		8: {ID: 8, Status: StatusActive, AllowedGroups: []int64{10, 20}},
	}}
	groups.groups[0].IsExclusive = true
	groups.groups[1].IsExclusive = true
	s.keys.cfg = &config.Config{Team: config.TeamConfig{Enabled: true}}
	s.keys.teamRepo = &fakeTeamRepository{teamContext: &TeamContext{Owner: &TeamMembership{UserID: 8}}}
	personal, err := s.GetRoutingPriorities(context.Background(), 7, "personal")
	require.NoError(t, err)
	require.Empty(t, personal.Groups)
	team, err := s.GetRoutingPriorities(context.Background(), 7, "team")
	require.NoError(t, err)
	require.Equal(t, []int64{10, 20}, priorityGroupIDs(team.Groups))
	s.keys.teamRepo = &fakeTeamRepository{}
	_, err = s.GetRoutingPriorities(context.Background(), 7, "team")
	require.ErrorIs(t, err, ErrTeamMembershipRequired)
}

func TestRoutingPrioritiesKnownExactRuleCanExplainWildcardModels(t *testing.T) {
	s, groups, catalog := routingPriorityFixture()
	for _, id := range []int64{10, 20} {
		account := catalog.catalog.Accounts[id][0]
		account.Credentials = map[string]any{"model_mapping": map[string]any{"gpt-*": "gpt-*"}}
		catalog.catalog.Accounts[id] = []Account{account}
	}
	s.policy = NewAutoGroupRoutingPolicyService(&autoPolicySettings{value: `{"default_group_order":[],"model_rules":[{"model":"gpt-private-deploy","group_ids":[20,10]}]}`}, groups)
	got, err := s.GetRoutingPriorities(context.Background(), 7, "personal")
	require.NoError(t, err)
	require.Len(t, got.ModelRules, 1)
	require.Equal(t, "gpt-private-deploy", got.ModelRules[0].Model)
	require.Equal(t, []int64{20, 10}, priorityGroupIDs(got.ModelRules[0].Groups))
}
