//go:build unit

package handler

import (
	"testing"

	middleware "github.com/LuckyKuang/sub2api-plus/internal/server/middleware"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestBatchImageOwnerFromContext_PreservesActorBillingOwnerAndTeam(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(nil)
	teamID := int64(77)
	groupID := int64(88)
	ctx.Set(string(middleware.ContextKeyAPIKey), &service.APIKey{
		ID:      303,
		UserID:  202,
		TeamID:  &teamID,
		GroupID: &groupID,
		User:    &service.User{ID: 101},
	})

	owner, ok := batchImageOwnerFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, int64(202), owner.UserID)
	require.Equal(t, int64(101), owner.BillingUserID)
	require.Equal(t, &teamID, owner.TeamID)
	require.Equal(t, int64(303), owner.APIKeyID)
	require.Equal(t, &groupID, owner.GroupID)
}

func TestBatchImageOwnerFromContext_RejectsMissingBillingOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(nil)
	ctx.Set(string(middleware.ContextKeyAPIKey), &service.APIKey{ID: 303, UserID: 202})

	_, ok := batchImageOwnerFromContext(ctx)
	require.False(t, ok)
}
