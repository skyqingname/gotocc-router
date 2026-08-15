package service

import (
	"context"
	"errors"
	"strings"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/logger"
	"go.uber.org/zap"
)

const (
	batchImageHoldRequestPrefix    = "batch_image_hold:"
	batchImageCaptureRequestPrefix = "batch_image_capture:"
	batchImageReleaseRequestPrefix = "batch_image_release:"
)

func BatchImageHoldRequestID(batchID string) string {
	return batchImageHoldRequestPrefix + strings.TrimSpace(batchID)
}

func BatchImageCaptureRequestID(batchID string) string {
	return batchImageCaptureRequestPrefix + strings.TrimSpace(batchID)
}

func BatchImageReleaseRequestID(batchID string) string {
	return batchImageReleaseRequestPrefix + strings.TrimSpace(batchID)
}

func batchImageBillingUserID(job *BatchImageJob) int64 {
	if job == nil {
		return 0
	}
	if job.BillingUserID > 0 {
		return job.BillingUserID
	}
	return job.UserID
}

func buildBatchImageHoldCommand(job *BatchImageJob, requestID string, actualAmount float64, payloadHash string) (*BatchImageBalanceHoldCommand, error) {
	if job == nil {
		return nil, ErrBatchImageBillingHoldFailed
	}
	if job.APIKeyID == nil || *job.APIKeyID <= 0 {
		return nil, ErrBatchImageSettlementMissingAPIKeyID
	}
	holdAmount := job.EstimatedCost
	if job.HoldAmount != nil {
		holdAmount = *job.HoldAmount
	}
	if holdAmount < 0 {
		holdAmount = 0
	}
	if actualAmount < 0 {
		actualAmount = 0
	}
	cmd := &BatchImageBalanceHoldCommand{
		RequestID:          requestID,
		APIKeyID:           *job.APIKeyID,
		UserID:             batchImageBillingUserID(job),
		BatchID:            job.BatchID,
		HoldAmount:         holdAmount,
		ActualAmount:       actualAmount,
		RequestPayloadHash: strings.TrimSpace(payloadHash),
	}
	// Historical in-flight jobs keep their original fingerprint and legacy
	// allowance behavior. New jobs always persist billing_user_id.
	if job.BillingUserID > 0 || job.TeamID != nil {
		cmd.ActorUserID = job.UserID
		cmd.TeamID = job.TeamID
		cmd.AllowanceReserved = job.AllowanceReserved
		cmd.ReservedAt = job.CreatedAt
	}
	return cmd, nil
}

func reserveBatchImageBalanceHold(ctx context.Context, repo UsageBillingRepository, job *BatchImageJob, payloadHash string) error {
	if repo == nil {
		return ErrBatchImageBillingHoldFailed.WithCause(errors.New("batch image billing repository is not configured"))
	}
	cmd, err := buildBatchImageHoldCommand(job, BatchImageHoldRequestID(job.BatchID), 0, payloadHash)
	if err != nil {
		return err
	}
	if cmd.HoldAmount <= 0 {
		return nil
	}
	if _, err := repo.ReserveBatchImageBalance(ctx, cmd); err != nil {
		if errors.Is(err, ErrBatchImageInsufficientBalance) {
			return ErrBatchImageInsufficientBalance
		}
		if errors.Is(err, ErrAPIKeyQuotaExhausted) || errors.Is(err, ErrAPIKeyRateLimit5hExceeded) || errors.Is(err, ErrAPIKeyRateLimit1dExceeded) || errors.Is(err, ErrAPIKeyRateLimit7dExceeded) ||
			errors.Is(err, ErrTeamMemberDailyExceeded) || errors.Is(err, ErrTeamMemberWeeklyExceeded) || errors.Is(err, ErrTeamMemberMonthlyExceeded) {
			return err
		}
		return ErrBatchImageBillingHoldFailed.WithCause(err)
	}
	job.AllowanceReserved = true
	return nil
}

func captureBatchImageBalanceHold(ctx context.Context, repo UsageBillingRepository, job *BatchImageJob, actualAmount float64, payloadHash string) error {
	if repo == nil {
		return ErrBatchImageSettlementBillingFailed.WithCause(errors.New("batch image billing repository is not configured"))
	}
	cmd, err := buildBatchImageHoldCommand(job, BatchImageCaptureRequestID(job.BatchID), actualAmount, payloadHash)
	if err != nil {
		return err
	}
	if _, err := repo.CaptureBatchImageBalance(ctx, cmd); err != nil {
		return ErrBatchImageSettlementBillingFailed.WithCause(err)
	}
	job.AllowanceReserved = false
	return nil
}

func releaseBatchImageBalanceHold(ctx context.Context, repo UsageBillingRepository, job *BatchImageJob, payloadHash string) error {
	if repo == nil || job == nil {
		return nil
	}
	cmd, err := buildBatchImageHoldCommand(job, BatchImageReleaseRequestID(job.BatchID), 0, payloadHash)
	if err != nil {
		return err
	}
	if cmd.HoldAmount <= 0 {
		return nil
	}
	if _, err := repo.ReleaseBatchImageBalance(ctx, cmd); err != nil {
		// 同一 release request id 出现指纹冲突，说明此前已有一次携带不同
		// payloadHash 的释放成功提交（资金已归还）。视为幂等成功，
		// 避免历史指纹不一致的 job 永远卡在释放失败的毒消息循环里。
		if errors.Is(err, ErrUsageBillingRequestConflict) {
			logger.L().Warn("batch_image.release_fingerprint_conflict_treated_as_released",
				zap.String("batch_id", job.BatchID),
			)
			job.AllowanceReserved = false
			return nil
		}
		return ErrBatchImageBillingHoldFailed.WithCause(err)
	}
	job.AllowanceReserved = false
	return nil
}
