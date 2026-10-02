//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/service"
	dbmigrations "github.com/LuckyKuang/sub2api-plus/migrations"
	"github.com/stretchr/testify/require"
)

func TestSmartRoutingMigrationUpgradesLegacyKeysAndInvalidatesAuth(t *testing.T) {
	tx, ctx := testTx(t), context.Background()
	userID := batchImageTestUserID(t, ctx, tx)
	_, err := tx.ExecContext(ctx, `ALTER TABLE api_keys DROP COLUMN routing_mode`)
	require.NoError(t, err)
	var id int64
	key := "sk-smart-migration-test"
	require.NoError(t, tx.QueryRowContext(ctx, `INSERT INTO api_keys(user_id,key,name) VALUES($1,$2,'legacy') RETURNING id`, userID, key).Scan(&id))
	data, err := dbmigrations.FS.ReadFile("253_api_key_smart_routing.sql")
	require.NoError(t, err)
	for i := 0; i < 2; i++ {
		_, err = tx.ExecContext(ctx, string(data))
		require.NoError(t, err)
	}
	var mode string
	require.NoError(t, tx.QueryRowContext(ctx, `SELECT routing_mode FROM api_keys WHERE id=$1`, id).Scan(&mode))
	require.Equal(t, "fixed", mode)
	var before, after int
	countSQL := `SELECT count(*) FROM auth_cache_invalidation_outbox WHERE cache_key=encode(sha256(convert_to($1,'UTF8')),'hex')`
	require.NoError(t, tx.QueryRowContext(ctx, countSQL, key).Scan(&before))
	_, err = tx.ExecContext(ctx, `UPDATE api_keys SET routing_mode='auto' WHERE id=$1`, id)
	require.NoError(t, err)
	require.NoError(t, tx.QueryRowContext(ctx, countSQL, key).Scan(&after))
	require.Equal(t, before+1, after)
	_, err = tx.ExecContext(ctx, `SAVEPOINT invalid_mode`)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, `UPDATE api_keys SET routing_mode='unknown' WHERE id=$1`, id)
	require.Error(t, err)
	_, err = tx.ExecContext(ctx, `ROLLBACK TO SAVEPOINT invalid_mode`)
	require.NoError(t, err)
}

func TestSmartRoutingBatchGroupPersistsAndMigrationIsIdempotent(t *testing.T) {
	tx, ctx := testTx(t), context.Background()
	userID := batchImageTestUserID(t, ctx, tx)
	data, err := dbmigrations.FS.ReadFile("254_batch_image_routing_group.sql")
	require.NoError(t, err)
	for i := 0; i < 2; i++ {
		_, err = tx.ExecContext(ctx, string(data))
		require.NoError(t, err)
	}
	repo := newBatchImageRepositoryWithSQL(tx)
	groupID := int64(4321)
	job, err := repo.CreateBatchImageJob(ctx, service.CreateBatchImageJobParams{
		BatchID: batchImageTestID(t, "group"), UserID: userID, Provider: service.BatchImageProviderGeminiAPI, Model: "gemini-2.5-flash-image", GroupID: &groupID,
	})
	require.NoError(t, err)
	require.Equal(t, groupID, *job.GroupID)
	loaded, err := repo.GetBatchImageJobByBatchID(ctx, job.BatchID)
	require.NoError(t, err)
	require.Equal(t, groupID, *loaded.GroupID)
}

func TestSmartRoutingResponseAffinityIsSharedAtomicAndKeyScoped(t *testing.T) {
	rdb := testRedis(t)
	first, second := &gatewayCache{rdb: rdb}, &gatewayCache{rdb: rdb}
	ctx := context.Background()
	require.NoError(t, first.StoreAutoResponseAffinity(ctx, "1:hash", []byte("original"), time.Minute))
	require.Error(t, second.StoreAutoResponseAffinity(ctx, "1:hash", []byte("replacement"), time.Minute))
	data, err := second.LoadAutoResponseAffinity(ctx, "1:hash")
	require.NoError(t, err)
	require.Equal(t, "original", string(data))
	data, err = second.LoadAutoResponseAffinity(ctx, "2:hash")
	require.NoError(t, err)
	require.Nil(t, data)
}
