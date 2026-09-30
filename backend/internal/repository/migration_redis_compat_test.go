package repository

import (
	"context"
	"testing"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func newMigrationFixtureRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return server, client
}

func TestPlusCandidateReadsLegacyRefreshTokenKeysAndJSON(t *testing.T) {
	server, client := newMigrationFixtureRedis(t)
	const tokenHash = "legacy-token-hash"
	const familyID = "legacy-family"
	const userID = int64(77)
	legacyJSON := `{"user_id":77,"token_version":12,"family_id":"legacy-family","created_at":"2025-01-02T03:04:05Z","expires_at":"2099-01-02T03:04:05Z"}`
	require.NoError(t, server.Set("refresh_token:"+tokenHash, legacyJSON))
	_, err := server.SAdd("user_refresh_tokens:77", tokenHash)
	require.NoError(t, err)
	_, err = server.SAdd("token_family:"+familyID, tokenHash)
	require.NoError(t, err)

	cache := NewRefreshTokenCache(client)
	data, err := cache.GetRefreshToken(context.Background(), tokenHash)
	require.NoError(t, err)
	require.Equal(t, userID, data.UserID)
	require.Equal(t, int64(12), data.TokenVersion)
	require.Equal(t, familyID, data.FamilyID)
	require.Empty(t, data.BindingHash, "pre-binding refresh tokens must remain readable")
	hashes, err := cache.GetUserTokenHashes(context.Background(), userID)
	require.NoError(t, err)
	require.Equal(t, []string{tokenHash}, hashes)
	inFamily, err := cache.IsTokenInFamily(context.Background(), familyID, tokenHash)
	require.NoError(t, err)
	require.True(t, inFamily)
}

func TestPlusCandidateReadsLegacyExecutionCacheFormats(t *testing.T) {
	server, client := newMigrationFixtureRedis(t)
	ctx := context.Background()

	require.NoError(t, server.Set("sticky_session:8:legacy-session", "901"))
	gateway := NewGatewayCache(client)
	accountID, err := gateway.GetSessionAccountID(ctx, 8, "legacy-session")
	require.NoError(t, err)
	require.Equal(t, int64(901), accountID)

	require.NoError(t, server.Set("image_task:imgtask_legacy", `{"id":"imgtask_legacy","user_id":77,"api_key_id":88,"status":"completed","http_status":200,"result":{"data":[{"url":"https://cdn.example.test/legacy.png"}]},"created_at":1700000000,"completed_at":1700000010,"expires_at":1700086400}`))
	imageStore := NewImageTaskStore(client)
	imageTask, err := imageStore.Get(ctx, "imgtask_legacy")
	require.NoError(t, err)
	require.Equal(t, service.ImageTaskStatusCompleted, imageTask.Status)
	require.Empty(t, imageTask.RequestType, "new optional history metadata must not break old task JSON")

	server.HSet("billing:sub:77:8", "status", "active")
	server.HSet("billing:sub:77:8", "expires_at", "4102444800")
	server.HSet("billing:sub:77:8", "daily_usage", "1.25")
	server.HSet("billing:sub:77:8", "weekly_usage", "2.5")
	server.HSet("billing:sub:77:8", "monthly_usage", "3.75")
	server.HSet("billing:sub:77:8", "version", "1700000000")
	billing := NewBillingCache(client)
	subscription, err := billing.GetSubscriptionCache(ctx, 77, 8)
	require.NoError(t, err)
	require.Equal(t, 1.25, subscription.DailyUsage)
	require.False(t, subscription.FiveHourStatePresent, "old billing hashes must trigger database rehydration for new quota state")

	now := time.Now().Unix()
	_, err = server.ZAdd("concurrency:account:901", float64(now), "legacy-request")
	require.NoError(t, err)
	concurrency := NewConcurrencyCache(client, 15, 900)
	count, err := concurrency.GetAccountConcurrency(ctx, 901)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestMigrationCriticalRedisPrefixesRemainStable(t *testing.T) {
	require.Equal(t, "refresh_token:hash", refreshTokenKey("hash"))
	require.Equal(t, "user_refresh_tokens:77", userRefreshTokensKey(77))
	require.Equal(t, "token_family:family", tokenFamilyKey("family"))
	require.Equal(t, "sticky_session:8:session", buildSessionKey(8, "session"))
	require.Equal(t, "image_task:imgtask_1", imageTaskKey("imgtask_1"))
	require.Equal(t, "billing:balance:77", billingBalanceKey(77))
	require.Equal(t, "billing:sub:77:8", billingSubKey(77, 8))
	require.Equal(t, "concurrency:account:901", accountSlotKeyPrefix+"901")
	require.Equal(t, "sched:acc:901", schedulerAccountPrefix+"901")
}
