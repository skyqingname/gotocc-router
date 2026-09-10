//go:build unit

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestTeamAPIKeyErrorsHaveStableGatewayStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "feature_disabled", err: service.ErrTeamFeatureDisabled, wantStatus: http.StatusForbidden, wantCode: "TEAM_FEATURE_DISABLED"},
		{name: "team_suspended", err: service.ErrTeamSuspended, wantStatus: http.StatusForbidden, wantCode: "TEAM_SUSPENDED"},
		{name: "membership_missing", err: service.ErrTeamMembershipRequired, wantStatus: http.StatusForbidden, wantCode: "TEAM_MEMBERSHIP_REQUIRED"},
		{name: "actor_inactive", err: service.ErrTeamActorInactive, wantStatus: http.StatusForbidden, wantCode: "TEAM_ACTOR_INACTIVE"},
		{name: "owner_inactive", err: service.ErrTeamBillingOwnerInactive, wantStatus: http.StatusForbidden, wantCode: "TEAM_BILLING_OWNER_INACTIVE"},
		{name: "daily_limit", err: service.ErrTeamMemberDailyExceeded, wantStatus: http.StatusTooManyRequests, wantCode: "TEAM_MEMBER_DAILY_LIMIT_EXCEEDED"},
		{name: "weekly_limit", err: service.ErrTeamMemberWeeklyExceeded, wantStatus: http.StatusTooManyRequests, wantCode: "TEAM_MEMBER_WEEKLY_LIMIT_EXCEEDED"},
		{name: "monthly_limit", err: service.ErrTeamMemberMonthlyExceeded, wantStatus: http.StatusTooManyRequests, wantCode: "TEAM_MEMBER_MONTHLY_LIMIT_EXCEEDED"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			require.True(t, abortTeamAPIKeyError(ctx, test.err))
			require.Equal(t, test.wantStatus, recorder.Code)
			require.Contains(t, recorder.Body.String(), test.wantCode)
		})
	}
}

func TestTeamMemberLimitsSkipExactAsyncImageReads(t *testing.T) {
	for _, path := range []string{
		"/v1/images/tasks/imgtask_123",
		"/images/tasks/imgtask_123",
		"/v1/images/objects/imgobj_123/url",
		"/images/objects/imgobj_123/url",
	} {
		require.True(t, isAPIKeyNonConsumingRequest(http.MethodGet, path), path)
	}
	require.False(t, isAPIKeyNonConsumingRequest(http.MethodPost, "/v1/images/tasks/imgtask_123"))
	require.False(t, isAPIKeyNonConsumingRequest(http.MethodPost, "/v1/images/objects/imgobj_123/url"))
	require.False(t, isAPIKeyNonConsumingRequest(http.MethodGet, "/v1/images/objects/imgobj_123"))
	require.False(t, isAPIKeyNonConsumingRequest(http.MethodPost, "/v1/images/generations"))
}
