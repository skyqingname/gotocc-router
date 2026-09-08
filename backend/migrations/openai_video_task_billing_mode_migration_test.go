package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAIVideoTaskBillingModeMigrationFreezesSupportedUnits(t *testing.T) {
	content, err := FS.ReadFile("239_openai_video_task_billing_mode.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "ADD COLUMN billing_mode VARCHAR(32) NOT NULL DEFAULT 'video'")
	require.Contains(t, sql, "ALTER COLUMN billing_mode DROP DEFAULT")
	require.Contains(t, sql, "CHECK (billing_mode IN ('per_request', 'video'))")
}
