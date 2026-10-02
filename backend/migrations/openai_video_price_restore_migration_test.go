package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPlusVideoCleanupLeavesOpenAIJimengPricesToCompensatingMigration(t *testing.T) {
	plusCleanup, err := FS.ReadFile("218_clear_non_grok_video_generation_config.sql")
	require.NoError(t, err)
	require.Contains(t, string(plusCleanup), "groups_video_price_backup_218")
	require.NotContains(t, strings.ToLower(string(plusCleanup)), "platform = 'openai'")

	restore, err := FS.ReadFile("225_restore_openai_video_prices.sql")
	require.NoError(t, err)
	sql := strings.ToLower(string(restore))
	require.Contains(t, sql, "groups_video_price_backup_218")
	require.Contains(t, sql, "b.platform = 'openai'")
	require.Contains(t, sql, "g.platform = 'openai'")
	require.NotContains(t, sql, "drop table")
	require.NotContains(t, sql, "delete from")
}
