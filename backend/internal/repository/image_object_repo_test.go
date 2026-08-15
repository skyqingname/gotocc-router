package repository

import (
	"context"
	"database/sql"
	"testing"

	dbent "github.com/LuckyKuang/sub2api-plus/ent"
	"github.com/LuckyKuang/sub2api-plus/ent/enttest"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func newImageObjectRepositoryForTest(t *testing.T) service.ImageObjectRepository {
	t.Helper()
	db, err := sql.Open("sqlite", "file:image_objects?mode=memory&cache=shared&_fk=1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)
	driver := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(driver)))
	t.Cleanup(func() { _ = client.Close() })
	return NewImageObjectRepository(client)
}

func TestImageObjectRepositoryPersistsAndScopesOwnership(t *testing.T) {
	repo := newImageObjectRepositoryForTest(t)
	require.NoError(t, repo.CreateMany(context.Background(), []service.ImageObjectRecord{{
		ObjectID: "imgobj_123", UserID: 7, APIKeyID: 9, TaskID: "imgtask_123",
		StorageKey: "images/imgtask_123-0.png", ContentType: "image/png", Bytes: 42,
	}}))

	object, err := repo.GetOwned(context.Background(), "imgobj_123", 7)
	require.NoError(t, err)
	require.Equal(t, int64(9), object.APIKeyID)
	require.Equal(t, "images/imgtask_123-0.png", object.StorageKey)

	_, err = repo.GetOwned(context.Background(), "imgobj_123", 8)
	require.ErrorIs(t, err, service.ErrImageObjectNotFound)
	_, err = repo.GetOwned(context.Background(), "imgobj_missing", 7)
	require.ErrorIs(t, err, service.ErrImageObjectNotFound)
}
