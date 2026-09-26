package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImageObjectMigrationIsAdditiveAndContainsOwnershipIndexes(t *testing.T) {
	content, err := FS.ReadFile("224_add_image_objects.sql")
	require.NoError(t, err)
	sql := strings.ToLower(string(content))
	for _, marker := range []string{
		"create table if not exists image_objects",
		"object_id varchar(64) not null",
		"user_id bigint not null",
		"api_key_id bigint not null",
		"storage_key varchar(1024) not null",
		"unique index if not exists image_objects_object_id_key",
		"index if not exists image_objects_user_id_created_at_idx",
	} {
		require.Contains(t, sql, marker)
	}
	for _, forbidden := range []string{"drop table", "drop column", "delete from", "update "} {
		require.NotContains(t, sql, forbidden)
	}
}
