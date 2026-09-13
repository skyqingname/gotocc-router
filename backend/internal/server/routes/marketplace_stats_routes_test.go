package routes

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPublicHomepageStatsRouteRemainsAnonymousAndNarrow(t *testing.T) {
	source, err := os.ReadFile("auth.go")
	require.NoError(t, err)
	text := string(source)

	marketplace := strings.Index(text, `marketplace := v1.Group("/marketplace")`)
	authenticated := strings.Index(text, `authenticated := v1.Group("")`)
	require.GreaterOrEqual(t, marketplace, 0)
	require.Greater(t, authenticated, marketplace, "homepage stats must be registered before authenticated routes")
	require.Contains(t, text[marketplace:authenticated], `marketplace.GET("/stats", h.MarketplaceStats.GetPublicStats)`)
	require.NotContains(t, text[marketplace:authenticated], `GET("/models"`)
}
