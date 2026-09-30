//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAutoRouteResourceRestoresOriginalGroupWithoutModelSelection(t *testing.T) {
	resolver, key, _, _, catalog := newAutoRouteFixture()
	route, err := resolver.RestoreGroup(context.Background(), key, 20, PlatformAnthropic)
	require.NoError(t, err)
	require.Equal(t, int64(20), *route.Key.GroupID)
	require.Zero(t, catalog.calls)
	require.Nil(t, key.GroupID)
}

func TestAutoRouteResourceCannotRestoreUnauthorizedOrChangedPlatform(t *testing.T) {
	resolver, key, _, groups, _ := newAutoRouteFixture()
	groups.groups[1].IsExclusive = true
	_, err := resolver.RestoreGroup(context.Background(), key, 20, PlatformAnthropic)
	require.ErrorIs(t, err, ErrAutoRouteNoAccess)
	_, err = resolver.RestoreGroup(context.Background(), key, 10, PlatformGemini)
	require.ErrorIs(t, err, ErrAutoRouteNoAccess)
}

func TestAutoRouteRechecksFreshIPRestrictions(t *testing.T) {
	resolver, key, _, _, _ := newAutoRouteFixture()
	resolver.keys.apiKeyRepo.(*autoRouteKeyRepo).key.IPWhitelist = []string{"10.0.0.0/8"}
	_, err := resolver.FreshKey(WithAutoRouteClientIP(context.Background(), "192.0.2.1"), key)
	require.ErrorIs(t, err, ErrAutoRouteNoAccess)
}
