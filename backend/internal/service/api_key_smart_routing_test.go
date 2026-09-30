//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/config"
	"github.com/stretchr/testify/require"
)

type autoRouteUserRepo struct {
	UserRepository
	user *User
}

func (r *autoRouteUserRepo) GetByID(context.Context, int64) (*User, error) {
	user := *r.user
	return &user, nil
}

type autoRouteGroupRepo struct {
	GroupRepository
	groups []Group
	loads  []int64
}

func (r *autoRouteGroupRepo) ListActive(context.Context) ([]Group, error) {
	return append([]Group(nil), r.groups...), nil
}

func (r *autoRouteGroupRepo) GetByID(_ context.Context, id int64) (*Group, error) {
	r.loads = append(r.loads, id)
	for _, group := range r.groups {
		if group.ID == id {
			copy := group
			return &copy, nil
		}
	}
	return nil, ErrGroupNotFound
}

func (r *autoRouteGroupRepo) GetByIDLite(ctx context.Context, id int64) (*Group, error) {
	return r.GetByID(ctx, id)
}

type autoRouteKeyRepo struct {
	APIKeyRepository
	key            *APIKey
	lastUsedWrites int
}

func (r *autoRouteKeyRepo) GetByID(context.Context, int64) (*APIKey, error) {
	key := *r.key
	return &key, nil
}

func (r *autoRouteKeyRepo) UpdateLastUsed(context.Context, int64, time.Time) error {
	r.lastUsedWrites++
	return nil
}

type autoRouteSubscriptionRepo struct {
	UserSubscriptionRepository
	subscriptions []UserSubscription
}

func (r *autoRouteSubscriptionRepo) ListActiveByUserID(context.Context, int64) ([]UserSubscription, error) {
	return r.subscriptions, nil
}

func (r *autoRouteSubscriptionRepo) GetActiveByUserIDAndGroupID(_ context.Context, userID, groupID int64) (*UserSubscription, error) {
	for _, sub := range r.subscriptions {
		if sub.UserID == userID && sub.GroupID == groupID {
			copy := sub
			return &copy, nil
		}
	}
	return nil, ErrSubscriptionNotFound
}

type autoRouteCatalogRepo struct {
	catalog *AutoRouteCatalog
	err     error
	calls   int
}

func (r *autoRouteCatalogRepo) LoadAutoRouteCatalog(_ context.Context, _ []int64) (*AutoRouteCatalog, error) {
	r.calls++
	return r.catalog, r.err
}

func newAutoRouteFixture() (*AutoGroupResolver, *APIKey, *autoRouteUserRepo, *autoRouteGroupRepo, *autoRouteCatalogRepo) {
	users := &autoRouteUserRepo{user: &User{ID: 7, Status: StatusActive}}
	groups := &autoRouteGroupRepo{groups: []Group{
		{ID: 10, Platform: PlatformOpenAI, Status: StatusActive, SortOrder: 1},
		{ID: 20, Platform: PlatformAnthropic, Status: StatusActive, SortOrder: 2},
	}}
	catalog := &autoRouteCatalogRepo{catalog: &AutoRouteCatalog{
		Accounts: map[int64][]Account{
			10: {{ID: 100, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true,
				Credentials: map[string]any{"model_mapping": map[string]any{"gpt-test": "gpt-test"}}}},
			20: {{ID: 200, Platform: PlatformAnthropic, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true,
				Credentials: map[string]any{"model_mapping": map[string]any{"claude-test": "claude-test"}}}},
		},
	}}
	keys := &APIKeyService{userRepo: users, groupRepo: groups, userSubRepo: &autoRouteSubscriptionRepo{}}
	channels := &ChannelService{}
	channels.cache.Store(&channelCache{loadedAt: time.Now()})
	resolver := NewAutoGroupResolver(keys, catalog, channels, nil, nil, &config.Config{})
	cachedUser := *users.user
	key := &APIKey{ID: 1, UserID: 7, RoutingMode: APIKeyRoutingAuto, Status: StatusActive, User: &cachedUser}
	key.Key = "sk-auto-route-test"
	databaseKey := *key
	keys.apiKeyRepo = &autoRouteKeyRepo{key: &databaseKey}
	return resolver, key, users, groups, catalog
}

func TestAutoRouteSelectsAuthorizedGroupWithoutMutatingKey(t *testing.T) {
	resolver, key, _, _, catalog := newAutoRouteFixture()
	for model, wantGroup := range map[string]int64{"gpt-test": 10, "claude-test": 20} {
		decision, err := resolver.Resolve(context.Background(), key, AutoRouteRequest{Model: model, Endpoint: CompositeRouteEndpointResponses})
		require.NoError(t, err)
		require.Equal(t, wantGroup, *decision.Key.GroupID)
		require.NotSame(t, key, decision.Key)
		require.NotSame(t, key.User, decision.Key.User)
		require.Nil(t, key.GroupID)
	}
	require.Equal(t, 2, catalog.calls, "one bulk catalog load per request")
}

func TestAutoRouteDefaultOpenAIGroupDoesNotClaimClaudeModels(t *testing.T) {
	resolver, key, _, _, catalog := newAutoRouteFixture()
	account := catalog.catalog.Accounts[10][0]
	account.Credentials = nil
	catalog.catalog.Accounts[10][0] = account
	decision, err := resolver.Resolve(context.Background(), key, AutoRouteRequest{Model: "claude-test", Endpoint: CompositeRouteEndpointResponses})
	require.NoError(t, err)
	require.Equal(t, int64(20), *decision.Key.GroupID)
}

func TestAutoRouteRechecksAuthorization(t *testing.T) {
	resolver, key, users, groups, _ := newAutoRouteFixture()
	groups.groups[1].IsExclusive = true
	users.user.AllowedGroups = []int64{20}
	_, err := resolver.Resolve(context.Background(), key, AutoRouteRequest{Model: "claude-test", Endpoint: CompositeRouteEndpointResponses})
	require.NoError(t, err)
	users.user.AllowedGroups = nil
	_, err = resolver.Resolve(context.Background(), key, AutoRouteRequest{Model: "claude-test", Endpoint: CompositeRouteEndpointResponses})
	require.ErrorIs(t, err, ErrAutoRouteModelNotFound)
}

func TestAutoRouteDoesNotSwitchGroupsForTransientAccountLimits(t *testing.T) {
	resolver, key, _, groups, catalog := newAutoRouteFixture()
	groups.groups[1].Platform = PlatformOpenAI
	catalog.catalog.Accounts[20] = []Account{catalog.catalog.Accounts[10][0]}
	limitedUntil := time.Now().Add(time.Hour)
	first := catalog.catalog.Accounts[10][0]
	first.RateLimitResetAt = &limitedUntil
	catalog.catalog.Accounts[10][0] = first
	decision, err := resolver.Resolve(context.Background(), key, AutoRouteRequest{Model: "gpt-test", Endpoint: CompositeRouteEndpointResponses})
	require.NoError(t, err)
	require.Equal(t, int64(10), *decision.Key.GroupID)
}

func TestAutoRouteCatalogFailureDoesNotFallBack(t *testing.T) {
	resolver, key, _, _, catalog := newAutoRouteFixture()
	catalog.err = errors.New("catalog unavailable")
	_, err := resolver.Resolve(context.Background(), key, AutoRouteRequest{Model: "gpt-test", Endpoint: CompositeRouteEndpointResponses})
	require.ErrorIs(t, err, ErrAutoRouteUnavailable)
	require.Nil(t, key.GroupID)
}

func TestAutoRouteRejectsSimpleModeBeforeLoadingCatalog(t *testing.T) {
	resolver, key, _, _, catalog := newAutoRouteFixture()
	resolver.cfg.RunMode = config.RunModeSimple
	_, err := resolver.Resolve(context.Background(), key, AutoRouteRequest{Model: "gpt-test", Endpoint: CompositeRouteEndpointResponses})
	require.Error(t, err)
	require.Zero(t, catalog.calls)
}

func TestAutoRouteRequiresDirectImageModelSupport(t *testing.T) {
	resolver, key, _, groups, catalog := newAutoRouteFixture()
	groups.groups[0].AllowImageGeneration = true
	groups.groups[1].AllowImageGeneration = true
	groups.groups[1].Platform = PlatformOpenAI
	second := catalog.catalog.Accounts[10][0]
	catalog.catalog.Accounts[20] = []Account{second}
	first := catalog.catalog.Accounts[10][0]
	first.Credentials = map[string]any{"model_mapping": map[string]any{"gpt-test": "another-image-model"}}
	catalog.catalog.Accounts[10][0] = first
	decision, err := resolver.Resolve(context.Background(), key, AutoRouteRequest{Model: "gpt-test", Endpoint: CompositeRouteEndpointImages})
	require.NoError(t, err)
	require.Equal(t, int64(20), *decision.Key.GroupID)
}

func TestAutoRouteRequiresWebSocketTransport(t *testing.T) {
	resolver, key, _, groups, catalog := newAutoRouteFixture()
	resolver.cfg.Gateway.OpenAIWS.ModeRouterV2Enabled = true
	groups.groups[1].Platform = PlatformOpenAI
	second := catalog.catalog.Accounts[10][0]
	second.Extra = map[string]any{"openai_apikey_responses_websockets_v2_mode": OpenAIWSIngressModeHTTPBridge}
	catalog.catalog.Accounts[20] = []Account{second}
	first := catalog.catalog.Accounts[10][0]
	first.Extra = map[string]any{"openai_apikey_responses_websockets_v2_mode": OpenAIWSIngressModeOff}
	catalog.catalog.Accounts[10][0] = first
	decision, err := resolver.Resolve(context.Background(), key, AutoRouteRequest{Model: "gpt-test", Endpoint: AutoRouteEndpointResponsesWS})
	require.NoError(t, err)
	require.Equal(t, int64(20), *decision.Key.GroupID)
}

func TestAutoRouteRejectsChangedModeDespiteCachedAuthentication(t *testing.T) {
	resolver, key, _, _, catalog := newAutoRouteFixture()
	resolver.keys.apiKeyRepo.(*autoRouteKeyRepo).key.RoutingMode = APIKeyRoutingFixed
	_, err := resolver.Resolve(context.Background(), key, AutoRouteRequest{Model: "gpt-test", Endpoint: CompositeRouteEndpointResponses})
	require.ErrorIs(t, err, ErrAutoRouteContext)
	require.Zero(t, catalog.calls)
}

func TestAutoRouteLockPreventsClaudeCodeGroupFallback(t *testing.T) {
	_, _, _, groups, _ := newAutoRouteFixture()
	groupID, fallbackID := int64(10), int64(20)
	groups.groups[0].ClaudeCodeOnly = true
	groups.groups[0].FallbackGroupID = &fallbackID
	gateway := &GatewayService{groupRepo: groups}
	ctx := WithAutoRouteLock(context.Background(), 1, groupID)
	_, _, err := gateway.resolveGatewayGroup(ctx, &groupID)
	require.ErrorIs(t, err, ErrClaudeCodeOnly)
	require.Equal(t, []int64{10}, groups.loads)
}

func TestAutoRouteLockRejectsUnscopedAccountSelection(t *testing.T) {
	_, _, _, groups, _ := newAutoRouteFixture()
	gateway := &GatewayService{groupRepo: groups}
	ctx := WithAutoRouteLock(context.Background(), 1, 10)
	_, _, err := gateway.resolveGatewayGroup(ctx, nil)
	require.ErrorIs(t, err, ErrAutoRouteContext)
	require.Empty(t, groups.loads)
}

func TestAutoRouteAdmissionPreservesSubscriptionAtZeroBalance(t *testing.T) {
	resolver, key, _, groups, _ := newAutoRouteFixture()
	groups.groups[1].SubscriptionType = SubscriptionTypeSubscription
	now := time.Now()
	subs := &autoRouteSubscriptionRepo{subscriptions: []UserSubscription{{
		ID: 30, UserID: 7, GroupID: 20, Status: SubscriptionStatusActive,
		StartsAt: now.Add(-time.Hour), ExpiresAt: now.Add(7 * 24 * time.Hour),
		DailyWindowStart: &now, WeeklyWindowStart: &now, MonthlyWindowStart: &now,
	}}}
	resolver.keys.userSubRepo = subs
	resolver.subscriptions = &SubscriptionService{userSubRepo: subs, now: time.Now}
	decision, err := resolver.Resolve(context.Background(), key, AutoRouteRequest{Model: "claude-test", Endpoint: CompositeRouteEndpointResponses})
	require.NoError(t, err)
	require.Zero(t, resolver.keys.apiKeyRepo.(*autoRouteKeyRepo).lastUsedWrites)
	admission, err := resolver.Admit(decision.RequestContext(context.Background()), decision.Key)
	require.NoError(t, err)
	require.NotNil(t, admission.Subscription)
	require.Equal(t, int64(20), admission.Subscription.GroupID)
	require.Zero(t, admission.Key.User.Balance)
}

func TestAutoRouteAdmissionDoesNotChangeGroupAfterLimit(t *testing.T) {
	resolver, key, users, groups, catalog := newAutoRouteFixture()
	users.user.Balance = 100
	groups.groups[0].Platform = PlatformAnthropic
	groups.groups[1].SortOrder = 0
	groups.groups[1].SubscriptionType = SubscriptionTypeSubscription
	limit := 1.0
	groups.groups[1].DailyLimitUSD = &limit
	catalog.catalog.Accounts[10] = catalog.catalog.Accounts[20]
	now := time.Now()
	subs := &autoRouteSubscriptionRepo{subscriptions: []UserSubscription{{
		ID: 30, UserID: 7, GroupID: 20, Status: SubscriptionStatusActive,
		StartsAt: now.Add(-time.Hour), ExpiresAt: now.Add(7 * 24 * time.Hour),
		DailyWindowStart: &now, WeeklyWindowStart: &now, MonthlyWindowStart: &now,
		DailyUsageUSD: limit,
	}}}
	resolver.keys.userSubRepo = subs
	resolver.subscriptions = &SubscriptionService{userSubRepo: subs, now: time.Now}
	decision, err := resolver.Resolve(context.Background(), key, AutoRouteRequest{Model: "claude-test", Endpoint: CompositeRouteEndpointResponses})
	require.NoError(t, err)
	require.Equal(t, int64(20), *decision.Key.GroupID)
	_, err = resolver.Admit(decision.RequestContext(context.Background()), decision.Key)
	require.ErrorIs(t, err, ErrDailyLimitExceeded)
	require.Equal(t, 1, catalog.calls)
	require.Equal(t, int64(20), *decision.Key.GroupID)
}
