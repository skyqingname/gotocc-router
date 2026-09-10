//go:build unit

package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/usagestats"
	"github.com/LuckyKuang/sub2api-plus/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type marketplaceStatsUsageRepo struct {
	service.UsageLogRepository
}

func (marketplaceStatsUsageRepo) GetDashboardPublicStats(context.Context, time.Time, time.Time, bool) (*service.DashboardPublicStats, error) {
	return &service.DashboardPublicStats{TodayTokens: 11, TotalTokens: 22, TotalUsers: 33}, nil
}

func (marketplaceStatsUsageRepo) GetDashboardStats(context.Context) (*usagestats.DashboardStats, error) {
	panic("full dashboard query must not be used for public homepage stats")
}

func TestMarketplaceStatsHandlerReturnsOnlyPublicAggregates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewMarketplaceStatsHandler(service.NewDashboardService(marketplaceStatsUsageRepo{}, nil, nil, nil))
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/marketplace/stats", nil)

	h.GetPublicStats(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, map[string]any{
		"today_tokens": float64(11),
		"total_tokens": float64(22),
		"total_users":  float64(33),
	}, payload.Data)
}
