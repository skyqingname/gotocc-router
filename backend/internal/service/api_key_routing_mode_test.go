//go:build unit

package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAPIKeyUpdate_AutoModeUnbindsWithoutResettingUsage(t *testing.T) {
	groupID := int64(42)
	svc, repo := newUpdateFieldsAPIKeyService(&APIKey{
		ID: 1, UserID: 7, Key: "sk-test", Status: StatusActive,
		GroupID: &groupID, Quota: 100, QuotaUsed: 30, Usage5h: 12,
	})
	var req UpdateAPIKeyRequest
	require.NoError(t, json.Unmarshal([]byte(`{"routing_mode":"auto"}`), &req))

	updated, err := svc.Update(context.Background(), 1, 7, req)
	require.NoError(t, err)
	require.Nil(t, updated.GroupID, "auto mode must clear the persisted fixed group")
	require.Equal(t, 30.0, updated.QuotaUsed)
	require.Equal(t, 12.0, updated.Usage5h)
	require.Len(t, repo.updateFields, 1)
	require.True(t, repo.updateFields[0].GroupID)
	require.False(t, repo.updateFields[0].QuotaUsed)
	require.False(t, repo.updateFields[0].RateLimitUsage)
}

func TestAPIKeyUpdate_UnknownRoutingModeDoesNotWrite(t *testing.T) {
	svc, repo := newUpdateFieldsAPIKeyService(&APIKey{
		ID: 1, UserID: 7, Key: "sk-test", Status: StatusActive,
	})
	var req UpdateAPIKeyRequest
	require.NoError(t, json.Unmarshal([]byte(`{"routing_mode":"unknown"}`), &req))

	_, err := svc.Update(context.Background(), 1, 7, req)
	require.Error(t, err)
	require.Empty(t, repo.updateFields)
}

func TestAPIKeyRoutingModeSurvivesAuthSnapshot(t *testing.T) {
	svc := &APIKeyService{}
	key := &APIKey{ID: 1, UserID: 7, RoutingMode: APIKeyRoutingAuto, User: &User{ID: 7}}
	snapshot := svc.snapshotFromAPIKey(context.Background(), key)
	require.NotNil(t, snapshot)
	restored := svc.snapshotToAPIKey("sk-test", snapshot)
	require.True(t, restored.IsAutoRouting())
	require.Nil(t, restored.GroupID)
	require.Equal(t, APIKeyRoutingFixed, (&APIKey{}).EffectiveRoutingMode())
	snapshot.Version = 23
	_, used, err := svc.applyAuthCacheEntry("sk-test", &APIKeyAuthCacheEntry{Snapshot: snapshot})
	require.NoError(t, err)
	require.False(t, used, "snapshots without routing mode must be reloaded")
}

func TestAPIKeyUpdate_RoutingPresenceSemantics(t *testing.T) {
	fixed := APIKeyRoutingFixed
	groupID := int64(42)
	for _, tc := range []struct {
		name      string
		key       APIKey
		req       UpdateAPIKeyRequest
		wantError bool
		wantMode  string
		wantGroup *int64
	}{
		{"legacy null preserves fixed group", APIKey{GroupID: &groupID}, UpdateAPIKeyRequest{GroupIDSet: true}, false, APIKeyRoutingFixed, &groupID},
		{"explicit fixed null unassigns", APIKey{RoutingMode: APIKeyRoutingAuto}, UpdateAPIKeyRequest{RoutingMode: &fixed, GroupIDSet: true}, false, APIKeyRoutingFixed, nil},
		{"fixed requires group on mode switch", APIKey{RoutingMode: APIKeyRoutingAuto}, UpdateAPIKeyRequest{RoutingMode: &fixed}, true, "", nil},
		{"binding requires explicit fixed mode", APIKey{RoutingMode: APIKeyRoutingAuto}, UpdateAPIKeyRequest{GroupID: &groupID}, true, "", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.key.ID, tc.key.UserID, tc.key.Key, tc.key.Status = 1, 7, "sk-test", StatusActive
			svc, repo := newUpdateFieldsAPIKeyService(&tc.key)
			updated, err := svc.Update(context.Background(), 1, 7, tc.req)
			if tc.wantError {
				require.Error(t, err)
				require.Empty(t, repo.updateFields)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantMode, updated.EffectiveRoutingMode())
			require.Equal(t, tc.wantGroup, updated.GroupID)
		})
	}
}
