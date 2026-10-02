package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTeamMigrationAvoidsHistoricalFullTableBackfill(t *testing.T) {
	content, err := FS.ReadFile("221_add_teams.sql")
	require.NoError(t, err)
	sql := strings.ToLower(string(content))
	for _, forbidden := range []string{
		"update usage_logs set billing_user_id",
		"update batch_image_jobs set billing_user_id",
		"alter column billing_user_id set not null",
	} {
		require.NotContains(t, sql, forbidden)
	}
}

func TestTeamAttributionIndexesUseNonTransactionalMigration(t *testing.T) {
	content, err := FS.ReadFile("223_add_team_attribution_indexes_notx.sql")
	require.NoError(t, err)
	sql := strings.ToUpper(string(content))
	require.Contains(t, sql, "CREATE INDEX CONCURRENTLY IF NOT EXISTS")
	require.NotContains(t, sql, "ALTER TABLE")
	require.NotContains(t, sql, "UPDATE ")
}
