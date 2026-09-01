package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	dbent "github.com/LuckyKuang/sub2api-plus/ent"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/logger"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/timezone"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
)

type usageBillingRepository struct {
	db *sql.DB
}

func NewUsageBillingRepository(_ *dbent.Client, sqlDB *sql.DB) service.UsageBillingRepository {
	return &usageBillingRepository{db: sqlDB}
}

func NewOpenAIVideoBillingRepository(_ *dbent.Client, sqlDB *sql.DB) service.OpenAIVideoBillingRepository {
	return &usageBillingRepository{db: sqlDB}
}

func (r *usageBillingRepository) Apply(ctx context.Context, cmd *service.UsageBillingCommand) (_ *service.UsageBillingApplyResult, err error) {
	if cmd == nil {
		return &service.UsageBillingApplyResult{}, nil
	}
	if r == nil || r.db == nil {
		return nil, errors.New("usage billing repository db is nil")
	}

	cmd.Normalize()
	if cmd.RequestID == "" {
		return nil, service.ErrUsageBillingRequestIDRequired
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	applied, err := r.claimUsageBillingKey(ctx, tx, cmd)
	if err != nil {
		return nil, err
	}
	if !applied {
		return &service.UsageBillingApplyResult{Applied: false}, nil
	}

	result := &service.UsageBillingApplyResult{Applied: true}
	if err := r.applyUsageBillingEffects(ctx, tx, cmd, result); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	tx = nil
	return result, nil
}

func (r *usageBillingRepository) claimUsageBillingKey(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand) (bool, error) {
	return r.claimUsageBillingRequest(ctx, tx, cmd.RequestID, cmd.APIKeyID, cmd.RequestFingerprint)
}

func (r *usageBillingRepository) claimUsageBillingRequest(ctx context.Context, tx *sql.Tx, requestID string, apiKeyID int64, requestFingerprint string) (bool, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `
		INSERT INTO usage_billing_dedup (request_id, api_key_id, request_fingerprint)
		VALUES ($1, $2, $3)
		ON CONFLICT (request_id, api_key_id) DO NOTHING
		RETURNING id
	`, requestID, apiKeyID, requestFingerprint).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		var existingFingerprint string
		if err := tx.QueryRowContext(ctx, `
			SELECT request_fingerprint
			FROM usage_billing_dedup
			WHERE request_id = $1 AND api_key_id = $2
		`, requestID, apiKeyID).Scan(&existingFingerprint); err != nil {
			return false, err
		}
		if strings.TrimSpace(existingFingerprint) != strings.TrimSpace(requestFingerprint) {
			return false, service.ErrUsageBillingRequestConflict
		}
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var archivedFingerprint string
	err = tx.QueryRowContext(ctx, `
		SELECT request_fingerprint
		FROM usage_billing_dedup_archive
		WHERE request_id = $1 AND api_key_id = $2
	`, requestID, apiKeyID).Scan(&archivedFingerprint)
	if err == nil {
		if strings.TrimSpace(archivedFingerprint) != strings.TrimSpace(requestFingerprint) {
			return false, service.ErrUsageBillingRequestConflict
		}
		return false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	return true, nil
}

func (r *usageBillingRepository) ReserveBatchImageBalance(ctx context.Context, cmd *service.BatchImageBalanceHoldCommand) (*service.BatchImageBalanceHoldResult, error) {
	return r.applyBatchImageBalanceHold(ctx, cmd, batchImageAllowanceReserve, reserveUsageBillingBatchImageBalance)
}

func (r *usageBillingRepository) CaptureBatchImageBalance(ctx context.Context, cmd *service.BatchImageBalanceHoldCommand) (*service.BatchImageBalanceHoldResult, error) {
	return r.applyBatchImageBalanceHold(ctx, cmd, batchImageAllowanceCapture, captureUsageBillingBatchImageBalance)
}

func (r *usageBillingRepository) ReleaseBatchImageBalance(ctx context.Context, cmd *service.BatchImageBalanceHoldCommand) (*service.BatchImageBalanceHoldResult, error) {
	return r.applyBatchImageBalanceHold(ctx, cmd, batchImageAllowanceRelease, releaseUsageBillingBatchImageBalance)
}

func (r *usageBillingRepository) ReserveOpenAIVideoBalance(ctx context.Context, cmd *service.OpenAIVideoBalanceHoldCommand) error {
	return r.applyOpenAIVideoBalance(ctx, cmd, "hold")
}

func (r *usageBillingRepository) CaptureOpenAIVideoBalance(ctx context.Context, cmd *service.OpenAIVideoBalanceHoldCommand) error {
	return r.applyOpenAIVideoBalance(ctx, cmd, "capture")
}

func (r *usageBillingRepository) ReleaseOpenAIVideoBalance(ctx context.Context, cmd *service.OpenAIVideoBalanceHoldCommand) error {
	return r.applyOpenAIVideoBalance(ctx, cmd, "release")
}

func (r *usageBillingRepository) applyOpenAIVideoBalance(ctx context.Context, cmd *service.OpenAIVideoBalanceHoldCommand, operation string) (err error) {
	if cmd == nil {
		return nil
	}
	if r == nil || r.db == nil {
		return errors.New("openai video billing repository db is nil")
	}
	cmd.LocalRequestID = strings.TrimSpace(cmd.LocalRequestID)
	cmd.HoldAmount = service.QuantizeUsageBillingAmount(cmd.HoldAmount)
	cmd.ActualAmount = service.QuantizeUsageBillingAmount(cmd.ActualAmount)
	requestID := service.OpenAIVideoHoldRequestID(cmd.LocalRequestID)
	switch operation {
	case "capture":
		requestID = service.OpenAIVideoCaptureRequestID(cmd.LocalRequestID)
	case "release":
		requestID = service.OpenAIVideoReleaseRequestID(cmd.LocalRequestID)
	}
	if cmd.LocalRequestID == "" || cmd.TaskID <= 0 || cmd.APIKeyID <= 0 || cmd.UserID <= 0 {
		return service.ErrOpenAIVideoTaskConflict
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()
	fingerprint := openAIVideoBillingFingerprint(cmd, operation)
	applied, err := r.claimUsageBillingRequest(ctx, tx, requestID, cmd.APIKeyID, fingerprint)
	if err != nil {
		return err
	}
	if !applied {
		return nil
	}

	var billingStatus string
	err = tx.QueryRowContext(ctx, `SELECT billing_status FROM openai_video_tasks
		WHERE id=$1 AND local_request_id=$2 AND api_key_id=$3 FOR UPDATE`,
		cmd.TaskID, cmd.LocalRequestID, cmd.APIKeyID).Scan(&billingStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrOpenAIVideoTaskNotFound
	}
	if err != nil {
		return err
	}

	batchCmd := &service.BatchImageBalanceHoldCommand{
		APIKeyID: cmd.APIKeyID, UserID: cmd.UserID, ActorUserID: cmd.ActorUserID,
		TeamID: cmd.TeamID, BatchID: cmd.LocalRequestID, HoldAmount: cmd.HoldAmount,
		ActualAmount: cmd.ActualAmount, AllowanceReserved: cmd.AllowanceReserved,
		ReservedAt: cmd.ReservedAt, RequestPayloadHash: cmd.RequestPayloadHash,
	}
	switch operation {
	case "hold":
		if billingStatus != service.OpenAIVideoBillingStatusNone {
			return service.ErrOpenAIVideoTaskConflict
		}
		if _, err = reserveUsageBillingBatchImageBalance(ctx, tx, batchCmd); err != nil {
			return err
		}
		if cmd.HoldAmount > 0 {
			if err = reserveBatchImageAPIKeyAllowance(ctx, tx, cmd.APIKeyID, cmd.HoldAmount, cmd.ReservedAt); err != nil {
				return err
			}
			if err = reserveBatchImageMemberAllowance(ctx, tx, batchCmd, cmd.HoldAmount); err != nil {
				return err
			}
		}
		_, err = tx.ExecContext(ctx, `UPDATE openai_video_tasks SET
			billing_status='held', allowance_reserved=$2, updated_at=NOW() WHERE id=$1`,
			cmd.TaskID, cmd.HoldAmount > 0)
	case "capture":
		if billingStatus != service.OpenAIVideoBillingStatusHeld {
			return service.ErrOpenAIVideoTaskConflict
		}
		if _, err = captureUsageBillingBatchImageBalance(ctx, tx, batchCmd); err != nil {
			return err
		}
		adjustment := cmd.HoldAmount - cmd.ActualAmount
		if adjustment > 0 && cmd.AllowanceReserved {
			if err = releaseBatchImageAPIKeyAllowance(ctx, tx, cmd.APIKeyID, adjustment, cmd.ReservedAt); err != nil {
				return err
			}
			if err = releaseBatchImageMemberAllowance(ctx, tx, batchCmd, adjustment); err != nil {
				return err
			}
		}
		if cmd.AccountQuotaCost > 0 {
			if _, err = incrementUsageBillingAccountQuota(ctx, tx, cmd.AccountID, cmd.AccountQuotaCost); err != nil {
				return err
			}
		}
		_, err = tx.ExecContext(ctx, `UPDATE openai_video_tasks SET
			billing_status='captured', actual_cost=$2, allowance_reserved=FALSE,
			settled_at=NOW(), next_poll_at=NOW(), lease_until=NULL, lease_token=NULL,
			updated_at=NOW() WHERE id=$1`, cmd.TaskID, cmd.ActualAmount)
	case "release":
		if billingStatus != service.OpenAIVideoBillingStatusHeld {
			return service.ErrOpenAIVideoTaskConflict
		}
		held, heldErr := usageBillingClaimExists(ctx, tx, service.OpenAIVideoHoldRequestID(cmd.LocalRequestID), cmd.APIKeyID)
		if heldErr != nil {
			return heldErr
		}
		if held && cmd.HoldAmount > 0 {
			var balance, frozen float64
			err = tx.QueryRowContext(ctx, `UPDATE users SET
				balance=balance+$1, frozen_balance=COALESCE(frozen_balance,0)-$1,
				updated_at=NOW() WHERE id=$2 AND deleted_at IS NULL
				AND COALESCE(frozen_balance,0)>=$1 RETURNING balance,frozen_balance`,
				cmd.HoldAmount, cmd.UserID).Scan(&balance, &frozen)
			if err != nil {
				return err
			}
			if err = releaseBatchImageAPIKeyAllowance(ctx, tx, cmd.APIKeyID, cmd.HoldAmount, cmd.ReservedAt); err != nil {
				return err
			}
			if err = releaseBatchImageMemberAllowance(ctx, tx, batchCmd, cmd.HoldAmount); err != nil {
				return err
			}
		}
		_, err = tx.ExecContext(ctx, `UPDATE openai_video_tasks SET
			billing_status='released', actual_cost=0, allowance_reserved=FALSE,
			settled_at=NOW(), next_poll_at=NULL, lease_until=NULL, lease_token=NULL,
			updated_at=NOW() WHERE id=$1`, cmd.TaskID)
	default:
		return fmt.Errorf("unsupported openai video billing operation %q", operation)
	}
	if err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	tx = nil
	return nil
}

func openAIVideoBillingFingerprint(cmd *service.OpenAIVideoBalanceHoldCommand, operation string) string {
	raw := fmt.Sprintf("%s|%s|%d|%d|%d|%0.10f|%0.10f|%s",
		operation, strings.TrimSpace(cmd.LocalRequestID), cmd.TaskID, cmd.APIKeyID,
		cmd.UserID, cmd.HoldAmount, cmd.ActualAmount, strings.TrimSpace(cmd.RequestPayloadHash))
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func usageBillingClaimExists(ctx context.Context, tx *sql.Tx, requestID string, apiKeyID int64) (bool, error) {
	var exists int
	for _, table := range []string{"usage_billing_dedup", "usage_billing_dedup_archive"} {
		err := tx.QueryRowContext(ctx, `SELECT 1 FROM `+table+` WHERE request_id=$1 AND api_key_id=$2`, requestID, apiKeyID).Scan(&exists)
		if err == nil {
			return true, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return false, err
		}
	}
	return false, nil
}

type batchImageAllowanceOperation int

const (
	batchImageAllowanceReserve batchImageAllowanceOperation = iota
	batchImageAllowanceCapture
	batchImageAllowanceRelease
)

func (r *usageBillingRepository) applyBatchImageBalanceHold(
	ctx context.Context,
	cmd *service.BatchImageBalanceHoldCommand,
	operation batchImageAllowanceOperation,
	apply func(context.Context, *sql.Tx, *service.BatchImageBalanceHoldCommand) (*service.BatchImageBalanceHoldResult, error),
) (_ *service.BatchImageBalanceHoldResult, err error) {
	if cmd == nil {
		return &service.BatchImageBalanceHoldResult{}, nil
	}
	if r == nil || r.db == nil {
		return nil, errors.New("usage billing repository db is nil")
	}
	cmd.Normalize()
	if cmd.RequestID == "" {
		return nil, service.ErrUsageBillingRequestIDRequired
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	applied, err := r.claimUsageBillingRequest(ctx, tx, cmd.RequestID, cmd.APIKeyID, cmd.RequestFingerprint)
	if err != nil {
		return nil, err
	}
	if !applied {
		return &service.BatchImageBalanceHoldResult{Applied: false}, nil
	}

	result, err := apply(ctx, tx, cmd)
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = &service.BatchImageBalanceHoldResult{}
	}
	result.Applied = true
	if err := applyBatchImageAllowance(ctx, tx, cmd, operation); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	tx = nil
	return result, nil
}

func applyBatchImageAllowance(ctx context.Context, tx *sql.Tx, cmd *service.BatchImageBalanceHoldCommand, operation batchImageAllowanceOperation) error {
	if cmd == nil || cmd.HoldAmount <= 0 {
		return nil
	}
	switch operation {
	case batchImageAllowanceReserve:
		// New jobs persist both values. Legacy jobs intentionally keep the
		// pre-migration fingerprint and do not acquire a new allowance hold.
		if cmd.ReservedAt.IsZero() || cmd.ActorUserID <= 0 {
			return nil
		}
		if err := reserveBatchImageAPIKeyAllowance(ctx, tx, cmd.APIKeyID, cmd.HoldAmount, cmd.ReservedAt); err != nil {
			return err
		}
		if err := reserveBatchImageMemberAllowance(ctx, tx, cmd, cmd.HoldAmount); err != nil {
			return err
		}
		return setBatchImageAllowanceReserved(ctx, tx, cmd.BatchID, true)
	case batchImageAllowanceCapture:
		if cmd.AllowanceReserved {
			adjustment := cmd.HoldAmount - cmd.ActualAmount
			if adjustment > 0 {
				if err := releaseBatchImageAPIKeyAllowance(ctx, tx, cmd.APIKeyID, adjustment, cmd.ReservedAt); err != nil {
					return err
				}
				if err := releaseBatchImageMemberAllowance(ctx, tx, cmd, adjustment); err != nil {
					return err
				}
			}
		} else if cmd.ActualAmount > 0 {
			if err := chargeLegacyBatchImageAPIKey(ctx, tx, cmd.APIKeyID, cmd.ActualAmount); err != nil {
				return err
			}
			if cmd.TeamID != nil && cmd.ActorUserID > 0 && cmd.ActorUserID != cmd.UserID {
				if err := incrementUsageBillingTeamMember(ctx, tx, *cmd.TeamID, cmd.ActorUserID, cmd.ActualAmount, time.Now()); err != nil {
					return err
				}
			}
		}
		return setBatchImageAllowanceReserved(ctx, tx, cmd.BatchID, false)
	case batchImageAllowanceRelease:
		if cmd.AllowanceReserved {
			if cmd.ReservedAt.IsZero() || cmd.ActorUserID <= 0 {
				return errors.New("batch image allowance reservation metadata is missing")
			}
			if err := releaseBatchImageAPIKeyAllowance(ctx, tx, cmd.APIKeyID, cmd.HoldAmount, cmd.ReservedAt); err != nil {
				return err
			}
			if err := releaseBatchImageMemberAllowance(ctx, tx, cmd, cmd.HoldAmount); err != nil {
				return err
			}
		}
		return setBatchImageAllowanceReserved(ctx, tx, cmd.BatchID, false)
	default:
		return nil
	}
}

func setBatchImageAllowanceReserved(ctx context.Context, tx *sql.Tx, batchID string, reserved bool) error {
	result, err := tx.ExecContext(ctx, `
		UPDATE batch_image_jobs
		SET allowance_reserved = $2, updated_at = NOW()
		WHERE batch_id = $1`, batchID, reserved)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrBatchImageJobNotFound
	}
	return nil
}

func reserveBatchImageAPIKeyAllowance(ctx context.Context, tx *sql.Tx, apiKeyID int64, amount float64, reservedAt time.Time) error {
	var id int64
	err := tx.QueryRowContext(ctx, `
		UPDATE api_keys SET
			quota_used = quota_used + $1,
			usage_5h = CASE WHEN window_5h_start IS NULL OR window_5h_start + INTERVAL '5 hours' <= $5 THEN $1 ELSE usage_5h + $1 END,
			usage_1d = CASE WHEN window_1d_start IS NULL OR window_1d_start + INTERVAL '24 hours' <= $5 THEN $1 ELSE usage_1d + $1 END,
			usage_7d = CASE WHEN window_7d_start IS NULL OR window_7d_start + INTERVAL '7 days' <= $5 THEN $1 ELSE usage_7d + $1 END,
			window_5h_start = CASE WHEN window_5h_start IS NULL OR window_5h_start + INTERVAL '5 hours' <= $5 THEN $5 ELSE window_5h_start END,
			window_1d_start = CASE WHEN window_1d_start IS NULL OR window_1d_start + INTERVAL '24 hours' <= $5 THEN date_trunc('day', $5::timestamptz) ELSE window_1d_start END,
			window_7d_start = CASE WHEN window_7d_start IS NULL OR window_7d_start + INTERVAL '7 days' <= $5 THEN date_trunc('day', $5::timestamptz) ELSE window_7d_start END,
			status = CASE WHEN quota > 0 AND quota_used + $1 >= quota THEN $3 ELSE status END,
			updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL AND status = $4 AND team_owner_disabled = FALSE
		  AND (quota <= 0 OR quota_used + $1 <= quota)
		  AND (rate_limit_5h <= 0 OR (CASE WHEN window_5h_start IS NULL OR window_5h_start + INTERVAL '5 hours' <= $5 THEN 0 ELSE usage_5h END) + $1 <= rate_limit_5h)
		  AND (rate_limit_1d <= 0 OR (CASE WHEN window_1d_start IS NULL OR window_1d_start + INTERVAL '24 hours' <= $5 THEN 0 ELSE usage_1d END) + $1 <= rate_limit_1d)
		  AND (rate_limit_7d <= 0 OR (CASE WHEN window_7d_start IS NULL OR window_7d_start + INTERVAL '7 days' <= $5 THEN 0 ELSE usage_7d END) + $1 <= rate_limit_7d)
		RETURNING id`, amount, apiKeyID, service.StatusAPIKeyQuotaExhausted, service.StatusAPIKeyActive, reservedAt).Scan(&id)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	return batchImageAPIKeyAllowanceError(ctx, tx, apiKeyID, amount, reservedAt)
}

func batchImageAPIKeyAllowanceError(ctx context.Context, tx *sql.Tx, apiKeyID int64, amount float64, reservedAt time.Time) error {
	var status string
	var ownerDisabled bool
	var quota, quotaUsed, limit5h, limit1d, limit7d, usage5h, usage1d, usage7d float64
	var start5h, start1d, start7d sql.NullTime
	err := tx.QueryRowContext(ctx, `
		SELECT status, team_owner_disabled, quota, quota_used, rate_limit_5h, rate_limit_1d, rate_limit_7d,
		       usage_5h, usage_1d, usage_7d, window_5h_start, window_1d_start, window_7d_start
		FROM api_keys WHERE id = $1 AND deleted_at IS NULL`, apiKeyID).
		Scan(&status, &ownerDisabled, &quota, &quotaUsed, &limit5h, &limit1d, &limit7d, &usage5h, &usage1d, &usage7d, &start5h, &start1d, &start7d)
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrAPIKeyNotFound
	}
	if err != nil {
		return err
	}
	if ownerDisabled || status != service.StatusAPIKeyActive {
		if status == service.StatusAPIKeyQuotaExhausted {
			return service.ErrAPIKeyQuotaExhausted
		}
		return service.ErrAPIKeyNotFound
	}
	if quota > 0 && quotaUsed+amount > quota {
		return service.ErrAPIKeyQuotaExhausted
	}
	if limit5h > 0 && effectiveSQLWindowUsage(usage5h, start5h, service.RateLimitWindow5h, reservedAt)+amount > limit5h {
		return service.ErrAPIKeyRateLimit5hExceeded
	}
	if limit1d > 0 && effectiveSQLWindowUsage(usage1d, start1d, service.RateLimitWindow1d, reservedAt)+amount > limit1d {
		return service.ErrAPIKeyRateLimit1dExceeded
	}
	if limit7d > 0 && effectiveSQLWindowUsage(usage7d, start7d, service.RateLimitWindow7d, reservedAt)+amount > limit7d {
		return service.ErrAPIKeyRateLimit7dExceeded
	}
	return service.ErrAPIKeyNotFound
}

func effectiveSQLWindowUsage(usage float64, start sql.NullTime, duration time.Duration, at time.Time) float64 {
	if !start.Valid || !start.Time.Add(duration).After(at) {
		return 0
	}
	return usage
}

func reserveBatchImageMemberAllowance(ctx context.Context, tx *sql.Tx, cmd *service.BatchImageBalanceHoldCommand, amount float64) error {
	if cmd.TeamID == nil || cmd.ActorUserID == cmd.UserID {
		return nil
	}
	return incrementUsageBillingTeamMember(ctx, tx, *cmd.TeamID, cmd.ActorUserID, amount, cmd.ReservedAt)
}

func releaseBatchImageAPIKeyAllowance(ctx context.Context, tx *sql.Tx, apiKeyID int64, amount float64, reservedAt time.Time) error {
	result, err := tx.ExecContext(ctx, `
		UPDATE api_keys SET
			quota_used = GREATEST(0, quota_used - $1),
			usage_5h = CASE WHEN window_5h_start <= $3 AND $3 < window_5h_start + INTERVAL '5 hours' THEN GREATEST(0, usage_5h - $1) ELSE usage_5h END,
			usage_1d = CASE WHEN window_1d_start <= $3 AND $3 < window_1d_start + INTERVAL '24 hours' THEN GREATEST(0, usage_1d - $1) ELSE usage_1d END,
			usage_7d = CASE WHEN window_7d_start <= $3 AND $3 < window_7d_start + INTERVAL '7 days' THEN GREATEST(0, usage_7d - $1) ELSE usage_7d END,
			status = CASE WHEN status = $4 AND team_owner_disabled = FALSE AND (quota <= 0 OR GREATEST(0, quota_used - $1) < quota) THEN $5 ELSE status END,
			updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL`, amount, apiKeyID, reservedAt, service.StatusAPIKeyQuotaExhausted, service.StatusAPIKeyActive)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrAPIKeyNotFound
	}
	return nil
}

func releaseBatchImageMemberAllowance(ctx context.Context, tx *sql.Tx, cmd *service.BatchImageBalanceHoldCommand, amount float64) error {
	if cmd.TeamID == nil || cmd.ActorUserID == cmd.UserID {
		return nil
	}
	result, err := tx.ExecContext(ctx, `
		UPDATE team_memberships SET
			daily_usage_usd = CASE WHEN daily_window_start <= $4 AND $4 < daily_window_start + INTERVAL '1 day' THEN GREATEST(0, daily_usage_usd - $3) ELSE daily_usage_usd END,
			weekly_usage_usd = CASE WHEN weekly_window_start <= $4 AND $4 < weekly_window_start + INTERVAL '7 days' THEN GREATEST(0, weekly_usage_usd - $3) ELSE weekly_usage_usd END,
			monthly_usage_usd = CASE WHEN monthly_window_start <= $4 AND $4 < monthly_window_start + INTERVAL '1 month' THEN GREATEST(0, monthly_usage_usd - $3) ELSE monthly_usage_usd END,
			updated_at = NOW()
		WHERE team_id = $1 AND user_id = $2
		  AND joined_at <= $4 AND (left_at IS NULL OR left_at > $4)`, *cmd.TeamID, cmd.ActorUserID, amount, cmd.ReservedAt)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrTeamMembershipRequired
	}
	return nil
}

func chargeLegacyBatchImageAPIKey(ctx context.Context, tx *sql.Tx, apiKeyID int64, amount float64) error {
	result, err := tx.ExecContext(ctx, `
		UPDATE api_keys SET
			quota_used = quota_used + $1,
			usage_5h = CASE WHEN window_5h_start IS NULL OR window_5h_start + INTERVAL '5 hours' <= NOW() THEN $1 ELSE usage_5h + $1 END,
			usage_1d = CASE WHEN window_1d_start IS NULL OR window_1d_start + INTERVAL '24 hours' <= NOW() THEN $1 ELSE usage_1d + $1 END,
			usage_7d = CASE WHEN window_7d_start IS NULL OR window_7d_start + INTERVAL '7 days' <= NOW() THEN $1 ELSE usage_7d + $1 END,
			window_5h_start = CASE WHEN window_5h_start IS NULL OR window_5h_start + INTERVAL '5 hours' <= NOW() THEN NOW() ELSE window_5h_start END,
			window_1d_start = CASE WHEN window_1d_start IS NULL OR window_1d_start + INTERVAL '24 hours' <= NOW() THEN date_trunc('day', NOW()) ELSE window_1d_start END,
			window_7d_start = CASE WHEN window_7d_start IS NULL OR window_7d_start + INTERVAL '7 days' <= NOW() THEN date_trunc('day', NOW()) ELSE window_7d_start END,
			status = CASE WHEN quota > 0 AND quota_used + $1 >= quota AND team_owner_disabled = FALSE THEN $3 ELSE status END,
			updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL`, amount, apiKeyID, service.StatusAPIKeyQuotaExhausted)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrAPIKeyNotFound
	}
	return nil
}

func (r *usageBillingRepository) applyUsageBillingEffects(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand, result *service.UsageBillingApplyResult) error {
	if cmd.SubscriptionCost > 0 && cmd.SubscriptionID != nil {
		if err := incrementUsageBillingSubscription(ctx, tx, *cmd.SubscriptionID, cmd.SubscriptionCost); err != nil {
			return err
		}
	}

	if cmd.BalanceCost > 0 {
		newBalance, sufficient, err := deductUsageBillingBalance(ctx, tx, cmd.UserID, cmd.BalanceCost)
		if err != nil {
			return err
		}
		result.NewBalance = &newBalance
		result.BalanceOverdrafted = !sufficient
	}

	if cmd.TeamID != nil && cmd.ActorUserID > 0 && cmd.ActorUserID != cmd.UserID {
		if err := incrementUsageBillingTeamMember(ctx, tx, *cmd.TeamID, cmd.ActorUserID, cmd.SubscriptionCost+cmd.BalanceCost, time.Now()); err != nil {
			return err
		}
	}

	if cmd.APIKeyQuotaCost > 0 {
		exhausted, err := incrementUsageBillingAPIKeyQuota(ctx, tx, cmd.APIKeyID, cmd.APIKeyQuotaCost)
		if err != nil {
			return err
		}
		result.APIKeyQuotaExhausted = exhausted
	}

	if cmd.APIKeyRateLimitCost > 0 {
		if err := incrementUsageBillingAPIKeyRateLimit(ctx, tx, cmd.APIKeyID, cmd.APIKeyRateLimitCost); err != nil {
			return err
		}
	}

	if cmd.AccountQuotaCost > 0 && (strings.EqualFold(cmd.AccountType, service.AccountTypeAPIKey) || strings.EqualFold(cmd.AccountType, service.AccountTypeBedrock)) {
		quotaState, err := incrementUsageBillingAccountQuota(ctx, tx, cmd.AccountID, cmd.AccountQuotaCost)
		if err != nil {
			return err
		}
		result.QuotaState = quotaState
	}

	return nil
}

func incrementUsageBillingTeamMember(ctx context.Context, tx *sql.Tx, teamID, actorUserID int64, amount float64, now time.Time) error {
	if amount <= 0 {
		return nil
	}
	dailyStart := timezone.StartOfDay(now)
	weeklyStart := timezone.StartOfWeek(now)
	monthlyStart := timezone.StartOfMonth(now)
	var id int64
	err := tx.QueryRowContext(ctx, `
		UPDATE team_memberships SET
			daily_usage_usd = CASE WHEN daily_window_start IS NULL OR daily_window_start < $4 THEN $3 ELSE daily_usage_usd + $3 END,
			weekly_usage_usd = CASE WHEN weekly_window_start IS NULL OR weekly_window_start < $5 THEN $3 ELSE weekly_usage_usd + $3 END,
			monthly_usage_usd = CASE WHEN monthly_window_start IS NULL OR monthly_window_start < $6 THEN $3 ELSE monthly_usage_usd + $3 END,
			daily_window_start = CASE WHEN daily_window_start IS NULL OR daily_window_start < $4 THEN $4 ELSE daily_window_start END,
			weekly_window_start = CASE WHEN weekly_window_start IS NULL OR weekly_window_start < $5 THEN $5 ELSE weekly_window_start END,
			monthly_window_start = CASE WHEN monthly_window_start IS NULL OR monthly_window_start < $6 THEN $6 ELSE monthly_window_start END,
			updated_at = $7
		WHERE team_id = $1 AND user_id = $2 AND left_at IS NULL AND role = 'member'
		  AND (daily_limit_usd <= 0 OR (CASE WHEN daily_window_start IS NULL OR daily_window_start < $4 THEN 0 ELSE daily_usage_usd END) + $3 <= daily_limit_usd)
		  AND (weekly_limit_usd <= 0 OR (CASE WHEN weekly_window_start IS NULL OR weekly_window_start < $5 THEN 0 ELSE weekly_usage_usd END) + $3 <= weekly_limit_usd)
		  AND (monthly_limit_usd <= 0 OR (CASE WHEN monthly_window_start IS NULL OR monthly_window_start < $6 THEN 0 ELSE monthly_usage_usd END) + $3 <= monthly_limit_usd)
		RETURNING id`,
		teamID, actorUserID, amount, dailyStart, weeklyStart, monthlyStart, now).Scan(&id)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	return usageBillingTeamMemberAllowanceError(ctx, tx, teamID, actorUserID, amount, dailyStart, weeklyStart, monthlyStart)
}

func usageBillingTeamMemberAllowanceError(ctx context.Context, tx *sql.Tx, teamID, userID int64, amount float64, dailyStart, weeklyStart, monthlyStart time.Time) error {
	var role string
	var dailyLimit, weeklyLimit, monthlyLimit, dailyUsage, weeklyUsage, monthlyUsage float64
	var dailyWindow, weeklyWindow, monthlyWindow sql.NullTime
	err := tx.QueryRowContext(ctx, `
		SELECT role, daily_limit_usd, weekly_limit_usd, monthly_limit_usd,
		       daily_usage_usd, weekly_usage_usd, monthly_usage_usd,
		       daily_window_start, weekly_window_start, monthly_window_start
		FROM team_memberships WHERE team_id = $1 AND user_id = $2 AND left_at IS NULL`, teamID, userID).
		Scan(&role, &dailyLimit, &weeklyLimit, &monthlyLimit, &dailyUsage, &weeklyUsage, &monthlyUsage, &dailyWindow, &weeklyWindow, &monthlyWindow)
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrTeamMembershipRequired
	}
	if err != nil {
		return err
	}
	if role != service.TeamRoleMember {
		return service.ErrTeamMembershipRequired
	}
	if dailyLimit > 0 && effectiveNaturalWindowUsage(dailyUsage, dailyWindow, dailyStart)+amount > dailyLimit {
		return service.ErrTeamMemberDailyExceeded
	}
	if weeklyLimit > 0 && effectiveNaturalWindowUsage(weeklyUsage, weeklyWindow, weeklyStart)+amount > weeklyLimit {
		return service.ErrTeamMemberWeeklyExceeded
	}
	if monthlyLimit > 0 && effectiveNaturalWindowUsage(monthlyUsage, monthlyWindow, monthlyStart)+amount > monthlyLimit {
		return service.ErrTeamMemberMonthlyExceeded
	}
	return service.ErrTeamMembershipRequired
}

func effectiveNaturalWindowUsage(usage float64, window sql.NullTime, expectedStart time.Time) float64 {
	if !window.Valid || window.Time.Before(expectedStart) {
		return 0
	}
	return usage
}

func incrementUsageBillingSubscription(ctx context.Context, tx *sql.Tx, subscriptionID int64, costUSD float64) error {
	const updateSQL = `
		UPDATE user_subscriptions us
		SET
			daily_usage_usd = us.daily_usage_usd + $1,
			weekly_usage_usd = us.weekly_usage_usd + $1,
			monthly_usage_usd = us.monthly_usage_usd + $1,
			five_hour_usage_usd = CASE
				WHEN $1 <= 0 THEN us.five_hour_usage_usd
				WHEN us.five_hour_window_start IS NULL
					OR us.five_hour_window_start + INTERVAL '5 hours' <= NOW() THEN $1
				ELSE us.five_hour_usage_usd + $1
			END,
			five_hour_window_start = CASE
				WHEN $1 <= 0 THEN us.five_hour_window_start
				WHEN us.five_hour_window_start IS NULL
					OR us.five_hour_window_start + INTERVAL '5 hours' <= NOW() THEN NOW()
				ELSE us.five_hour_window_start
			END,
			-- The row lock serializes concurrent charges. Advancing by at least one
			-- microsecond gives cache snapshots a strict monotonic version even when
			-- several transactions land in the same wall-clock tick.
			updated_at = GREATEST(clock_timestamp(), us.updated_at + INTERVAL '1 microsecond')
		FROM groups g
		WHERE us.id = $2
			AND us.deleted_at IS NULL
			AND us.group_id = g.id
			AND g.deleted_at IS NULL
	`
	res, err := tx.ExecContext(ctx, updateSQL, costUSD, subscriptionID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected > 0 {
		return nil
	}
	return service.ErrSubscriptionNotFound
}

func deductUsageBillingBalance(ctx context.Context, tx *sql.Tx, userID int64, amount float64) (float64, bool, error) {
	var newBalance float64
	err := tx.QueryRowContext(ctx, `
		UPDATE users
		SET balance = balance - $1,
			updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL AND balance >= $1
		RETURNING balance
	`, amount, userID).Scan(&newBalance)
	if err == nil {
		return newBalance, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, false, err
	}

	err = tx.QueryRowContext(ctx, `
		UPDATE users
		SET balance = balance - $1,
			updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
		RETURNING balance
	`, amount, userID).Scan(&newBalance)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, service.ErrUserNotFound
	}
	if err != nil {
		return 0, false, err
	}
	return newBalance, false, nil
}

func reserveUsageBillingBatchImageBalance(ctx context.Context, tx *sql.Tx, cmd *service.BatchImageBalanceHoldCommand) (*service.BatchImageBalanceHoldResult, error) {
	if cmd.HoldAmount <= 0 {
		return &service.BatchImageBalanceHoldResult{}, nil
	}
	var balance, frozen float64
	err := tx.QueryRowContext(ctx, `
		UPDATE users
		SET balance = balance - $1,
			frozen_balance = COALESCE(frozen_balance, 0) + $1,
			updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL AND balance >= $1
		RETURNING balance, frozen_balance
	`, cmd.HoldAmount, cmd.UserID).Scan(&balance, &frozen)
	if err == nil {
		return &service.BatchImageBalanceHoldResult{NewBalance: &balance, FrozenBalance: &frozen}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if exists, existsErr := userExistsForBilling(ctx, tx, cmd.UserID); existsErr != nil {
		return nil, existsErr
	} else if !exists {
		return nil, service.ErrUserNotFound
	}
	return nil, service.ErrBatchImageInsufficientBalance
}

func captureUsageBillingBatchImageBalance(ctx context.Context, tx *sql.Tx, cmd *service.BatchImageBalanceHoldCommand) (*service.BatchImageBalanceHoldResult, error) {
	if cmd.HoldAmount <= 0 && cmd.ActualAmount <= 0 {
		return &service.BatchImageBalanceHoldResult{}, nil
	}
	if cmd.ActualAmount-cmd.HoldAmount > 0.00000001 {
		return nil, service.ErrBatchImageSettlementCostExceedsHold
	}
	var balance, frozen float64
	err := tx.QueryRowContext(ctx, `
		UPDATE users
		SET balance = balance
				+ CASE WHEN $1 > $2 THEN $1 - $2 ELSE 0 END
				- CASE WHEN $2 > $1 THEN $2 - $1 ELSE 0 END,
			frozen_balance = COALESCE(frozen_balance, 0) - $1,
			updated_at = NOW()
		WHERE id = $3 AND deleted_at IS NULL AND COALESCE(frozen_balance, 0) >= $1
		RETURNING balance, frozen_balance
	`, cmd.HoldAmount, cmd.ActualAmount, cmd.UserID).Scan(&balance, &frozen)
	if err == nil {
		return &service.BatchImageBalanceHoldResult{NewBalance: &balance, FrozenBalance: &frozen}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if exists, existsErr := userExistsForBilling(ctx, tx, cmd.UserID); existsErr != nil {
		return nil, existsErr
	} else if !exists {
		return nil, service.ErrUserNotFound
	}
	return nil, errors.New("batch image frozen balance is insufficient")
}

func releaseUsageBillingBatchImageBalance(ctx context.Context, tx *sql.Tx, cmd *service.BatchImageBalanceHoldCommand) (*service.BatchImageBalanceHoldResult, error) {
	if cmd.HoldAmount <= 0 {
		return &service.BatchImageBalanceHoldResult{}, nil
	}
	// 释放前校验该 job 确实预留过 hold（hold request id 已被 claim），
	// 防止从未成功冻结的 job 触发"幻影释放"，从其他用户的冻结资金池中凭空生成余额。
	held, heldErr := batchImageHoldClaimExists(ctx, tx, service.BatchImageHoldRequestID(cmd.BatchID), cmd.APIKeyID)
	if heldErr != nil {
		return nil, heldErr
	}
	if !held {
		logger.LegacyPrintf("repository.usage_billing", "[BatchImage] release skipped, hold was never reserved: batch=%s", cmd.BatchID)
		return &service.BatchImageBalanceHoldResult{}, nil
	}
	var balance, frozen float64
	err := tx.QueryRowContext(ctx, `
		UPDATE users
		SET balance = balance + $1,
			frozen_balance = COALESCE(frozen_balance, 0) - $1,
			updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL AND COALESCE(frozen_balance, 0) >= $1
		RETURNING balance, frozen_balance
	`, cmd.HoldAmount, cmd.UserID).Scan(&balance, &frozen)
	if err == nil {
		return &service.BatchImageBalanceHoldResult{NewBalance: &balance, FrozenBalance: &frozen}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if exists, existsErr := userExistsForBilling(ctx, tx, cmd.UserID); existsErr != nil {
		return nil, existsErr
	} else if !exists {
		return nil, service.ErrUserNotFound
	}
	return nil, errors.New("batch image frozen balance is insufficient")
}

// batchImageHoldClaimExists 检查 hold request id 是否已在 dedup（或归档）表中被 claim，
// 即该 batch 的冻结操作确实成功提交过。
func batchImageHoldClaimExists(ctx context.Context, tx *sql.Tx, holdRequestID string, apiKeyID int64) (bool, error) {
	var exists int
	err := tx.QueryRowContext(ctx, `
		SELECT 1
		FROM usage_billing_dedup
		WHERE request_id = $1 AND api_key_id = $2
	`, holdRequestID, apiKeyID).Scan(&exists)
	if err == nil {
		return true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	err = tx.QueryRowContext(ctx, `
		SELECT 1
		FROM usage_billing_dedup_archive
		WHERE request_id = $1 AND api_key_id = $2
	`, holdRequestID, apiKeyID).Scan(&exists)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return false, err
}

func userExistsForBilling(ctx context.Context, tx *sql.Tx, userID int64) (bool, error) {
	var exists int
	err := tx.QueryRowContext(ctx, `
		SELECT 1
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`, userID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func incrementUsageBillingAPIKeyQuota(ctx context.Context, tx *sql.Tx, apiKeyID int64, amount float64) (bool, error) {
	var exhausted bool
	err := tx.QueryRowContext(ctx, `
		UPDATE api_keys
		SET quota_used = quota_used + $1,
			status = CASE
				WHEN quota > 0
					AND status = $3
					AND quota_used < quota
					AND quota_used + $1 >= quota
				THEN $4
				ELSE status
			END,
			updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
		RETURNING quota > 0 AND quota_used >= quota AND quota_used - $1 < quota
	`, amount, apiKeyID, service.StatusAPIKeyActive, service.StatusAPIKeyQuotaExhausted).Scan(&exhausted)
	if errors.Is(err, sql.ErrNoRows) {
		return false, service.ErrAPIKeyNotFound
	}
	if err != nil {
		return false, err
	}
	return exhausted, nil
}

func incrementUsageBillingAPIKeyRateLimit(ctx context.Context, tx *sql.Tx, apiKeyID int64, cost float64) error {
	res, err := tx.ExecContext(ctx, `
		UPDATE api_keys SET
			usage_5h = CASE WHEN window_5h_start IS NOT NULL AND window_5h_start + INTERVAL '5 hours' <= NOW() THEN $1 ELSE usage_5h + $1 END,
			usage_1d = CASE WHEN window_1d_start IS NOT NULL AND window_1d_start + INTERVAL '24 hours' <= NOW() THEN $1 ELSE usage_1d + $1 END,
			usage_7d = CASE WHEN window_7d_start IS NOT NULL AND window_7d_start + INTERVAL '7 days' <= NOW() THEN $1 ELSE usage_7d + $1 END,
			window_5h_start = CASE WHEN window_5h_start IS NULL OR window_5h_start + INTERVAL '5 hours' <= NOW() THEN NOW() ELSE window_5h_start END,
			window_1d_start = CASE WHEN window_1d_start IS NULL OR window_1d_start + INTERVAL '24 hours' <= NOW() THEN date_trunc('day', NOW()) ELSE window_1d_start END,
			window_7d_start = CASE WHEN window_7d_start IS NULL OR window_7d_start + INTERVAL '7 days' <= NOW() THEN date_trunc('day', NOW()) ELSE window_7d_start END,
			updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
	`, cost, apiKeyID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrAPIKeyNotFound
	}
	return nil
}

func incrementUsageBillingAccountQuota(ctx context.Context, tx *sql.Tx, accountID int64, amount float64) (*service.AccountQuotaState, error) {
	rows, err := tx.QueryContext(ctx,
		`UPDATE accounts SET extra = (
			COALESCE(extra, '{}'::jsonb)
			|| jsonb_build_object('quota_used', COALESCE((extra->>'quota_used')::numeric, 0) + $1)
			|| CASE WHEN COALESCE((extra->>'quota_daily_limit')::numeric, 0) > 0 THEN
				jsonb_build_object(
					'quota_daily_used',
					CASE WHEN `+dailyExpiredExpr+`
					THEN $1
					ELSE COALESCE((extra->>'quota_daily_used')::numeric, 0) + $1 END,
					'quota_daily_start',
					CASE WHEN `+dailyExpiredExpr+`
					THEN `+nowUTC+`
					ELSE COALESCE(extra->>'quota_daily_start', `+nowUTC+`) END
				)
				|| CASE WHEN `+dailyExpiredExpr+` AND `+nextDailyResetAtExpr+` IS NOT NULL
				   THEN jsonb_build_object('quota_daily_reset_at', `+nextDailyResetAtExpr+`)
				   ELSE '{}'::jsonb END
			ELSE '{}'::jsonb END
			|| CASE WHEN COALESCE((extra->>'quota_weekly_limit')::numeric, 0) > 0 THEN
				jsonb_build_object(
					'quota_weekly_used',
					CASE WHEN `+weeklyExpiredExpr+`
					THEN $1
					ELSE COALESCE((extra->>'quota_weekly_used')::numeric, 0) + $1 END,
					'quota_weekly_start',
					CASE WHEN `+weeklyExpiredExpr+`
					THEN `+nowUTC+`
					ELSE COALESCE(extra->>'quota_weekly_start', `+nowUTC+`) END
				)
				|| CASE WHEN `+weeklyExpiredExpr+` AND `+nextWeeklyResetAtExpr+` IS NOT NULL
				   THEN jsonb_build_object('quota_weekly_reset_at', `+nextWeeklyResetAtExpr+`)
				   ELSE '{}'::jsonb END
			ELSE '{}'::jsonb END
		), updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
		RETURNING
			COALESCE((extra->>'quota_used')::numeric, 0),
			COALESCE((extra->>'quota_limit')::numeric, 0),
			COALESCE((extra->>'quota_daily_used')::numeric, 0),
			COALESCE((extra->>'quota_daily_limit')::numeric, 0),
			COALESCE((extra->>'quota_weekly_used')::numeric, 0),
			COALESCE((extra->>'quota_weekly_limit')::numeric, 0)`,
		amount, accountID)
	if err != nil {
		return nil, err
	}

	var state service.AccountQuotaState
	if rows.Next() {
		if err := rows.Scan(
			&state.TotalUsed, &state.TotalLimit,
			&state.DailyUsed, &state.DailyLimit,
			&state.WeeklyUsed, &state.WeeklyLimit,
		); err != nil {
			_ = rows.Close()
			return nil, err
		}
	} else {
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return nil, err
		}
		_ = rows.Close()
		return nil, service.ErrAccountNotFound
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	// 必须在执行下一条 SQL 前显式关闭 rows：pq 驱动在同一连接上
	// 不允许前一条查询的结果集未耗尽时启动新查询，否则会返回
	// "unexpected Parse response" 错误。
	if err := rows.Close(); err != nil {
		return nil, err
	}
	// 任意维度额度在本次递增中从"未超"跨越到"已超"时，必须刷新调度快照，
	// 否则 Redis 中缓存的 Account 仍显示旧的 used 值，后续请求会继续选中本账号，
	// 最终观察到 daily_used / weekly_used 大幅超过配置的 limit。
	// 对于日/周额度，即使本次触发了周期重置（pre=0、post=amount），
	// 判定式 (post-amount) < limit 同样成立，逻辑与总额度保持一致。
	crossedTotal := state.TotalLimit > 0 && state.TotalUsed >= state.TotalLimit && (state.TotalUsed-amount) < state.TotalLimit
	crossedDaily := state.DailyLimit > 0 && state.DailyUsed >= state.DailyLimit && (state.DailyUsed-amount) < state.DailyLimit
	crossedWeekly := state.WeeklyLimit > 0 && state.WeeklyUsed >= state.WeeklyLimit && (state.WeeklyUsed-amount) < state.WeeklyLimit
	if crossedTotal || crossedDaily || crossedWeekly {
		if err := enqueueSchedulerOutbox(ctx, tx, service.SchedulerOutboxEventAccountChanged, &accountID, nil, nil); err != nil {
			logger.LegacyPrintf("repository.usage_billing", "[SchedulerOutbox] enqueue quota exceeded failed: account=%d err=%v", accountID, err)
			return nil, err
		}
	}
	return &state, nil
}
