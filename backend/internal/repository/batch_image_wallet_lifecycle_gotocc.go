package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/LuckyKuang/sub2api-plus/internal/service"
)

// The caller holds the job row, so reserve, capture and release cannot commit
// conflicting operations against the same task's share of a pooled wallet.
func checkBatchImageWalletOperation(ctx context.Context, tx *sql.Tx, cmd *service.BatchImageBalanceHoldCommand, operation batchImageAllowanceOperation, status string) error {
	switch operation {
	case batchImageAllowanceReserve:
		if status != service.BatchImageJobStatusCreated && status != service.BatchImageJobStatusUploading {
			return service.ErrBatchImageInvalidTransition
		}
	case batchImageAllowanceCapture:
		released, err := batchImageHoldClaimExists(ctx, tx, service.BatchImageReleaseRequestID(cmd.BatchID), cmd.APIKeyID)
		if err != nil {
			return err
		}
		if released || status != service.BatchImageJobStatusSettling {
			return service.ErrBatchImageSettlementInvalidStatus
		}
	case batchImageAllowanceRelease:
		captured, err := batchImageHoldClaimExists(ctx, tx, service.BatchImageCaptureRequestID(cmd.BatchID), cmd.APIKeyID)
		if err != nil {
			return err
		}
		if captured || status == service.BatchImageJobStatusCompleted {
			return service.ErrBatchImageAlreadySettled
		}
	}
	return nil
}

func completeBatchImageWalletSettlement(ctx context.Context, tx *sql.Tx, cmd *service.BatchImageBalanceHoldCommand) error {
	if cmd.Settlement == nil {
		return nil
	}
	err := (&batchImageRepository{}).markBatchImageJobSettledWithSQL(ctx, tx, *cmd.Settlement)
	if errors.Is(err, service.ErrBatchImageAlreadySettled) {
		return nil
	}
	return err
}
