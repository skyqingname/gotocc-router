package handler

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestParseTeamUsageQueryDateEndUsesExclusiveNextDay(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/api/v1/team/usage?from=2026-08-01&to=2026-08-13&member_id=7&api_key_id=9&limit=25&offset=5", nil)

	query, err := parseTeamUsageQuery(c)
	require.NoError(t, err)
	require.Equal(t, time.Date(2026, 8, 1, 0, 0, 0, 0, query.From.Location()), query.From)
	require.Equal(t, time.Date(2026, 8, 14, 0, 0, 0, 0, query.To.Location()), query.To)
	require.Equal(t, int64(7), *query.ActorUserID)
	require.Equal(t, int64(9), *query.APIKeyID)
	require.Equal(t, 25, query.Limit)
	require.Equal(t, 5, query.Offset)
}

func TestParseTeamUsageQueryPreservesRFC3339Instants(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/api/v1/team/usage?from=2026-08-01T02%3A03%3A04%2B08%3A00&to=2026-08-13T05%3A06%3A07.123Z", nil)

	query, err := parseTeamUsageQuery(c)
	require.NoError(t, err)
	require.Equal(t, "2026-08-01T02:03:04+08:00", query.From.Format(time.RFC3339Nano))
	require.Equal(t, "2026-08-13T05:06:07.123Z", query.To.Format(time.RFC3339Nano))
}

func TestParseTeamUsageQueryRejectsInvalidMemberID(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/api/v1/team/usage?member_id=not-a-number", nil)

	_, err := parseTeamUsageQuery(c)
	require.Error(t, err)
}
