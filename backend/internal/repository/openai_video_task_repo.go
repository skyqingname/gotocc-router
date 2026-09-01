package repository

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/service"
)

type openAIVideoTaskRepository struct{ db *sql.DB }

func NewOpenAIVideoTaskRepository(db *sql.DB) service.OpenAIVideoTaskRepository {
	return &openAIVideoTaskRepository{db: db}
}

const openAIVideoTaskColumns = `
	id, local_request_id, task_id, actor_user_id, billing_user_id, team_id,
	api_key_id, group_id, channel_id, account_id, subscription_id,
	requested_model, upstream_model, request_seconds, resolution, status,
	upstream_status, billing_type, billing_status, total_cost, actual_cost,
	hold_amount, group_rate_multiplier, account_rate_multiplier,
	allowance_reserved, request_payload_hash, inbound_endpoint,
	upstream_endpoint, model_mapping_chain, user_agent, ip_address, retry_count,
	next_poll_at, lease_until, lease_token, last_error_code, last_error_message,
	usage_recorded, created_at, updated_at, submitted_at, finished_at,
	settled_at, usage_recorded_at`

type openAIVideoTaskScanner interface{ Scan(dest ...any) error }

func scanOpenAIVideoTask(row openAIVideoTaskScanner) (*service.OpenAIVideoTask, error) {
	task := &service.OpenAIVideoTask{}
	err := row.Scan(
		&task.ID, &task.LocalRequestID, &task.TaskID, &task.ActorUserID,
		&task.BillingUserID, &task.TeamID, &task.APIKeyID, &task.GroupID,
		&task.ChannelID, &task.AccountID, &task.SubscriptionID,
		&task.RequestedModel, &task.UpstreamModel, &task.RequestSeconds,
		&task.Resolution, &task.Status, &task.UpstreamStatus, &task.BillingType,
		&task.BillingStatus, &task.TotalCost, &task.ActualCost, &task.HoldAmount,
		&task.GroupRateMultiplier, &task.AccountRateMultiplier,
		&task.AllowanceReserved, &task.RequestPayloadHash, &task.InboundEndpoint,
		&task.UpstreamEndpoint, &task.ModelMappingChain, &task.UserAgent,
		&task.IPAddress, &task.RetryCount, &task.NextPollAt, &task.LeaseUntil,
		&task.LeaseToken, &task.LastErrorCode, &task.LastErrorMessage,
		&task.UsageRecorded, &task.CreatedAt, &task.UpdatedAt, &task.SubmittedAt,
		&task.FinishedAt, &task.SettledAt, &task.UsageRecordedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrOpenAIVideoTaskNotFound
	}
	return task, err
}

func (r *openAIVideoTaskRepository) Create(ctx context.Context, p service.CreateOpenAIVideoTaskParams) (*service.OpenAIVideoTask, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("openai video task repository db is nil")
	}
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO openai_video_tasks (
			local_request_id, actor_user_id, billing_user_id, team_id, api_key_id,
			group_id, channel_id, account_id, subscription_id, requested_model,
			upstream_model, request_seconds, resolution, billing_type, total_cost,
			hold_amount, group_rate_multiplier, account_rate_multiplier,
			request_payload_hash, inbound_endpoint, upstream_endpoint,
			model_mapping_chain, user_agent, ip_address, next_poll_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,
			$19,$20,$21,$22,$23,$24,$25
		)
		RETURNING `+openAIVideoTaskColumns,
		strings.TrimSpace(p.LocalRequestID), p.ActorUserID, p.BillingUserID,
		p.TeamID, p.APIKeyID, p.GroupID, p.ChannelID, p.AccountID,
		p.SubscriptionID, strings.TrimSpace(p.RequestedModel),
		strings.TrimSpace(p.UpstreamModel), p.RequestSeconds,
		strings.TrimSpace(p.Resolution), p.BillingType, p.TotalCost,
		p.HoldAmount, p.GroupRateMultiplier, p.AccountRateMultiplier,
		strings.TrimSpace(p.RequestPayloadHash), strings.TrimSpace(p.InboundEndpoint),
		strings.TrimSpace(p.UpstreamEndpoint), p.ModelMappingChain, p.UserAgent,
		p.IPAddress, p.NextPollAt,
	)
	return scanOpenAIVideoTask(row)
}

func (r *openAIVideoTaskRepository) BindUpstreamTask(ctx context.Context, localRequestID, taskID, upstreamStatus string, nextPollAt time.Time) (*service.OpenAIVideoTask, error) {
	localRequestID = strings.TrimSpace(localRequestID)
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, service.ErrOpenAIVideoTaskIDMissing
	}
	row := r.db.QueryRowContext(ctx, `
		UPDATE openai_video_tasks
		SET task_id=$2, status=$3, upstream_status=NULLIF($4,''),
			next_poll_at=$5, submitted_at=NOW(), updated_at=NOW()
		WHERE local_request_id=$1 AND status='creating'
		RETURNING `+openAIVideoTaskColumns,
		localRequestID, taskID, service.OpenAIVideoTaskStatusPending,
		strings.TrimSpace(upstreamStatus), nextPollAt)
	task, err := scanOpenAIVideoTask(row)
	if err == nil {
		return task, nil
	}
	if !errors.Is(err, service.ErrOpenAIVideoTaskNotFound) {
		return nil, err
	}
	existing := r.db.QueryRowContext(ctx, `SELECT `+openAIVideoTaskColumns+` FROM openai_video_tasks WHERE local_request_id=$1`, localRequestID)
	task, err = scanOpenAIVideoTask(existing)
	if err != nil {
		return nil, err
	}
	if task.TaskID == nil || strings.TrimSpace(*task.TaskID) != taskID {
		return nil, service.ErrOpenAIVideoTaskConflict
	}
	return task, nil
}

func (r *openAIVideoTaskRepository) GetByTaskIDForAPIKey(ctx context.Context, taskID string, apiKeyID int64) (*service.OpenAIVideoTask, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+openAIVideoTaskColumns+`
		FROM openai_video_tasks WHERE task_id=$1 AND api_key_id=$2`, strings.TrimSpace(taskID), apiKeyID)
	return scanOpenAIVideoTask(row)
}

func (r *openAIVideoTaskRepository) ClaimDue(ctx context.Context, now time.Time, leaseDuration time.Duration, limit int) ([]*service.OpenAIVideoTask, error) {
	if limit <= 0 {
		return nil, nil
	}
	leaseToken, err := newOpenAIVideoLeaseToken()
	if err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `
		WITH due AS (
			SELECT id FROM openai_video_tasks
			WHERE (lease_until IS NULL OR lease_until <= $1)
			  AND (
				(status IN ('creating','pending','processing') AND next_poll_at <= $1)
				OR (status='completed' AND (billing_status IN ('none','held','failed') OR usage_recorded=FALSE))
				OR (status IN ('failed','cancelled','expired') AND billing_status IN ('none','held','failed'))
			  )
			ORDER BY COALESCE(next_poll_at, updated_at), id
			FOR UPDATE SKIP LOCKED
			LIMIT $2
		)
		UPDATE openai_video_tasks t
		SET lease_until=$1 + ($3 * INTERVAL '1 second'), lease_token=$4, updated_at=NOW()
		FROM due WHERE t.id=due.id
		RETURNING `+qualifiedOpenAIVideoTaskColumns("t"),
		now, limit, int(leaseDuration.Seconds()), leaseToken)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var tasks []*service.OpenAIVideoTask
	for rows.Next() {
		task, scanErr := scanOpenAIVideoTask(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

func qualifiedOpenAIVideoTaskColumns(alias string) string {
	columns := strings.Fields(openAIVideoTaskColumns)
	for i := range columns {
		columns[i] = alias + "." + columns[i]
	}
	return strings.Join(columns, " ")
}

func (r *openAIVideoTaskRepository) RecordPollState(ctx context.Context, id int64, leaseToken, status, upstreamStatus, errorCode, errorMessage string, nextPollAt *time.Time, finishedAt *time.Time) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE openai_video_tasks SET status=$3, upstream_status=NULLIF($4,''),
			next_poll_at=$5, finished_at=COALESCE($6, finished_at),
			lease_until=NULL, lease_token=NULL, last_error_code=NULLIF($7,''),
			last_error_message=NULLIF($8,''), updated_at=NOW()
		WHERE id=$1 AND lease_token=$2`, id, leaseToken, status,
		strings.TrimSpace(upstreamStatus), nextPollAt, finishedAt,
		strings.TrimSpace(errorCode), strings.TrimSpace(errorMessage))
	return openAIVideoLeaseResult(result, err)
}

func (r *openAIVideoTaskRepository) RecordPollError(ctx context.Context, id int64, leaseToken, code, message string, nextPollAt time.Time) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE openai_video_tasks SET retry_count=retry_count+1,
			last_error_code=$3, last_error_message=$4, next_poll_at=$5,
			lease_until=NULL, lease_token=NULL, updated_at=NOW()
		WHERE id=$1 AND lease_token=$2`, id, leaseToken,
		strings.TrimSpace(code), strings.TrimSpace(message), nextPollAt)
	return openAIVideoLeaseResult(result, err)
}

func (r *openAIVideoTaskRepository) MarkCreateFailure(ctx context.Context, id int64, code, message string, at time.Time) error {
	result, err := r.db.ExecContext(ctx, `UPDATE openai_video_tasks SET
		status='failed', finished_at=$2, next_poll_at=$2,
		last_error_code=$3, last_error_message=$4, updated_at=NOW()
		WHERE id=$1 AND status='creating'`, id, at, strings.TrimSpace(code), strings.TrimSpace(message))
	return openAIVideoTaskUpdateResult(result, err)
}

func (r *openAIVideoTaskRepository) MarkBillingCaptured(ctx context.Context, id int64, actualCost float64, at time.Time) error {
	result, err := r.db.ExecContext(ctx, `UPDATE openai_video_tasks SET
		billing_status='captured', actual_cost=$2, allowance_reserved=FALSE,
		settled_at=$3, next_poll_at=$3, lease_until=NULL, lease_token=NULL,
		last_error_code=NULL, last_error_message=NULL, updated_at=NOW()
		WHERE id=$1 AND billing_status IN ('none','failed')`, id, actualCost, at)
	return openAIVideoTaskUpdateResult(result, err)
}

func (r *openAIVideoTaskRepository) MarkBillingReleased(ctx context.Context, id int64, at time.Time) error {
	result, err := r.db.ExecContext(ctx, `UPDATE openai_video_tasks SET
		billing_status='released', actual_cost=0, allowance_reserved=FALSE,
		settled_at=$2, next_poll_at=NULL, lease_until=NULL, lease_token=NULL,
		updated_at=NOW() WHERE id=$1 AND billing_status IN ('none','failed')`, id, at)
	return openAIVideoTaskUpdateResult(result, err)
}

func (r *openAIVideoTaskRepository) MarkBillingFailed(ctx context.Context, id int64, code, message string) error {
	result, err := r.db.ExecContext(ctx, `UPDATE openai_video_tasks SET
		billing_status='failed', retry_count=retry_count+1,
		last_error_code=$2, last_error_message=$3, next_poll_at=NOW(),
		lease_until=NULL, lease_token=NULL, updated_at=NOW() WHERE id=$1`,
		id, strings.TrimSpace(code), strings.TrimSpace(message))
	return openAIVideoTaskUpdateResult(result, err)
}

func (r *openAIVideoTaskRepository) MarkUsageRecorded(ctx context.Context, id int64, at time.Time) error {
	result, err := r.db.ExecContext(ctx, `UPDATE openai_video_tasks
		SET usage_recorded=TRUE, usage_recorded_at=$2, lease_until=NULL,
			lease_token=NULL, updated_at=NOW() WHERE id=$1`, id, at)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrOpenAIVideoTaskNotFound
	}
	return nil
}

func openAIVideoLeaseResult(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrOpenAIVideoTaskLeaseLost
	}
	return nil
}

func openAIVideoTaskUpdateResult(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrOpenAIVideoTaskNotFound
	}
	return nil
}

func newOpenAIVideoLeaseToken() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}
