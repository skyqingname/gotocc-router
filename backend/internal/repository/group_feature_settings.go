package repository

import (
	"context"
	"strconv"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/LuckyKuang/sub2api-plus/ent"
	"github.com/LuckyKuang/sub2api-plus/ent/setting"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
)

// CompareAndSwapGroupFeature serializes feature edits with group deletion and
// other feature writers. It cannot modify arbitrary settings or audit secrets.
// An empty expected value means that no previous setting exists.
func (r *settingRepository) CompareAndSwapGroupFeature(ctx context.Context, groupID int64, key, expected, replacement string) error {
	id := strconv.FormatInt(groupID, 10)
	if groupID <= 0 || replacement == "" || len(replacement) > 128*1024 ||
		(key != service.GroupRateScheduleKeyPrefix+id && key != "video_group_config:"+id) {
		return service.ErrGroupFeatureUnavailable
	}
	tx, err := r.client.Tx(ctx)
	if err != nil { return err }
	defer func() { _ = tx.Rollback() }()
	client := tx.Client()
	if client.Driver().Dialect() != dialect.Postgres {
		return service.ErrGroupFeatureUnavailable
	}
	var rows entsql.Rows
	if err := client.Driver().Query(ctx,
		"SELECT id FROM groups WHERE id = $1 AND deleted_at IS NULL FOR UPDATE",
		[]any{groupID}, &rows); err != nil { return err }
	if !rows.Next() {
		readErr := rows.Err()
		_ = rows.Close()
		if readErr != nil { return readErr }
		return service.ErrGroupNotFound
	}
	var lockedID int64
	readErr := rows.Scan(&lockedID)
	closeErr := rows.Close()
	if readErr != nil { return readErr }
	if closeErr != nil { return closeErr }
	if lockedID != groupID { return service.ErrGroupNotFound }

	current, err := tx.Setting.Query().Where(setting.KeyEQ(key)).Only(ctx)
	actual := ""
	if err != nil && !ent.IsNotFound(err) { return err }
	if current != nil { actual = current.Value }
	if actual != expected { return service.ErrGroupFeatureConflict }
	if err := tx.Setting.Create().SetKey(key).SetValue(replacement).SetUpdatedAt(time.Now()).
		OnConflictColumns(setting.FieldKey).UpdateNewValues().Exec(ctx); err != nil { return err }
	return tx.Commit()
}

var _ service.GroupFeatureSettingsStore = (*settingRepository)(nil)
