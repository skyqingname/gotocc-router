package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAgentProfilesMigrationIsAdditiveAndConstrainsStatus(t *testing.T) {
	content, err := FS.ReadFile("279_agent_profiles.sql")
	require.NoError(t, err)
	sql := strings.ToLower(strings.Join(strings.Fields(string(content)), " "))

	for _, marker := range []string{
		"create table if not exists agent_profiles",
		"user_id bigint primary key references users(id) on delete cascade",
		"status varchar(16) not null default 'pending'",
		"source varchar(24) not null default 'applied'",
		"constraint agent_profiles_status_check check (status in ('pending', 'approved', 'rejected'))",
		"constraint agent_profiles_source_check check (source in ('applied', 'grandfathered'))",
		"reviewed_by bigint references users(id) on delete set null",
	} {
		require.Contains(t, sql, marker)
	}

	// The enrollment cutoff must not be baked into SQL: build and migration time
	// are not the moment the new build serves traffic.
	require.NotContains(t, sql, "agent_enrollment_cutoff")

	for _, forbidden := range []string{"drop table", "drop column", "delete from", "update "} {
		require.NotContains(t, sql, forbidden)
	}
}
