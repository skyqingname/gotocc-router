//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAutoBatchSettlementKeepsStoredGroup(t *testing.T) {
	group, key, account := int64(7), int64(8), int64(9)
	job := &BatchImageJob{GroupID: &group, APIKeyID: &key, AccountID: &account, UserID: 4, BillingUserID: 4}
	usage := buildBatchImageSettlementUsageLog(job, 1, "request", time.Now())
	require.Equal(t, group, *usage.GroupID)
}

func TestAutoBatchIdempotentLookupDoesNotRequireCurrentGroup(t *testing.T) {
	svc, _, _, _, _ := newTestBatchImagePublicService(true)
	req := validBatchImageSubmitRequest()
	created, err := svc.Submit(context.Background(), testBatchImageOwner(), req, "stable")
	require.NoError(t, err)
	owner := testBatchImageOwner()
	id := int64(999)
	owner.GroupID = &id
	got, err := svc.FindIdempotentSubmission(context.Background(), owner, req, "stable")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, created.ID, got.ID)
}
