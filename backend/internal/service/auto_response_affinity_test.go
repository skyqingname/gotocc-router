//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type autoResponseCacheStub struct {
	GatewayCache
	values map[string][]byte
	err    error
}

func (s *autoResponseCacheStub) StoreAutoResponseAffinity(_ context.Context, key string, value []byte, _ time.Duration) error {
	if s.err != nil {
		return s.err
	}
	s.values[key] = append([]byte(nil), value...)
	return nil
}
func (s *autoResponseCacheStub) LoadAutoResponseAffinity(_ context.Context, key string) ([]byte, error) {
	return s.values[key], s.err
}

func TestAutoResponseAffinitySharedKeyScopedAndStrict(t *testing.T) {
	cache := &autoResponseCacheStub{values: make(map[string][]byte)}
	writer, reader := &OpenAIGatewayService{cache: cache}, &OpenAIGatewayService{cache: cache}
	groupID, teamID := int64(12), int64(5)
	key := &APIKey{ID: 2, UserID: 3, TeamID: &teamID, User: &User{ID: 7}, RoutingMode: APIKeyRoutingAuto, GroupID: &groupID}
	require.NoError(t, writer.BindAutoResponseAffinity(context.Background(), key, "resp_shared", 100))
	binding, err := reader.LookupAutoResponseAffinity(context.Background(), key, "resp_shared")
	require.NoError(t, err)
	require.Equal(t, int64(12), binding.GroupID)
	require.Equal(t, int64(100), binding.AccountID)
	other := *key
	other.ID = 9
	_, err = reader.LookupAutoResponseAffinity(context.Background(), &other, "resp_shared")
	require.ErrorIs(t, err, ErrAutoRouteContext)
	other = *key
	other.User = &User{ID: 8}
	_, err = reader.LookupAutoResponseAffinity(context.Background(), &other, "resp_shared")
	require.ErrorIs(t, err, ErrAutoRouteNoAccess)
	cache.err = errors.New("redis unavailable")
	_, err = reader.LookupAutoResponseAffinity(context.Background(), key, "resp_shared")
	require.ErrorIs(t, err, ErrAutoRouteUnavailable)
}

func TestAutoResponseAffinityRequiresSharedStore(t *testing.T) {
	s := &OpenAIGatewayService{}
	_, err := s.LookupAutoResponseAffinity(context.Background(), &APIKey{ID: 1, UserID: 2, User: &User{ID: 2}, RoutingMode: APIKeyRoutingAuto}, "resp_a")
	require.ErrorIs(t, err, ErrAutoRouteUnavailable)
}

func TestAutoVideoAffinityRetainsGroupAndOriginalSubscription(t *testing.T) {
	cache := &autoResponseCacheStub{GatewayCache: &stubGatewayCache{}, values: make(map[string][]byte)}
	s := &OpenAIGatewayService{cache: cache}
	group := int64(12)
	key := &APIKey{ID: 2, UserID: 3, User: &User{ID: 3}, GroupID: &group, Group: &Group{ID: group, Platform: PlatformGrok}, RoutingMode: APIKeyRoutingAuto}
	d := &AutoRouteDecision{Key: key, Platform: PlatformGrok}
	ctx := WithAutoRouteSubscription(d.RequestContext(context.Background()), 42)
	require.NoError(t, s.BindGrokMediaVideoRequestAccount(ctx, &group, "task-1", 3, 2, 100))
	binding, err := s.LookupAutoResponseAffinity(ctx, key, "video:task-1")
	require.NoError(t, err)
	require.Equal(t, PlatformGrok, binding.Platform)
	require.Equal(t, int64(42), binding.SubscriptionID)
	require.Equal(t, group, binding.GroupID)
}
