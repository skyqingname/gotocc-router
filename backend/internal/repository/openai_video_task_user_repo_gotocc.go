package repository

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/service"
)

// NewOpenAIVideoTaskUserRepository serves the task center. It reads the same
// openai_video_tasks rows as the worker (LC-013), always scoped to the actor.
func NewOpenAIVideoTaskUserRepository(db *sql.DB) service.OpenAIVideoTaskUserRepository {
	return &openAIVideoTaskRepository{db: db}
}

// Local statuses grouped into the three states the task center shows:
// failed|cancelled|expired are failed, every other non-terminal status is processing.
const (
	openAIVideoTaskUserCompletedSQL  = `status = 'completed'`
	openAIVideoTaskUserFailedSQL     = `status IN ('failed', 'cancelled', 'expired')`
	openAIVideoTaskUserProcessingSQL = `status NOT IN ('completed', 'failed', 'cancelled', 'expired')`
)

// openAIVideoTaskActorWhere builds the shared actor/status/model/time filter.
// The [since, until) range matches the image task history.
func openAIVideoTaskActorWhere(actorUserID int64, filter service.VideoTaskUserFilter) (string, []any) {
	where := "actor_user_id = $1"
	args := []any{actorUserID}
	switch filter.Status {
	case service.VideoTaskUserStatusCompleted:
		where += " AND " + openAIVideoTaskUserCompletedSQL
	case service.VideoTaskUserStatusFailed:
		where += " AND " + openAIVideoTaskUserFailedSQL
	case service.VideoTaskUserStatusProcessing:
		where += " AND " + openAIVideoTaskUserProcessingSQL
	}
	if filter.Model != "" {
		args = append(args, filter.Model)
		where += " AND requested_model = $" + strconv.Itoa(len(args))
	}
	if filter.Since > 0 {
		args = append(args, time.Unix(filter.Since, 0).UTC())
		where += " AND created_at >= $" + strconv.Itoa(len(args))
	}
	if filter.Until > 0 {
		args = append(args, time.Unix(filter.Until, 0).UTC())
		where += " AND created_at < $" + strconv.Itoa(len(args))
	}
	return where, args
}

func (r *openAIVideoTaskRepository) ListForActor(ctx context.Context, actorUserID int64, filter service.VideoTaskUserFilter) ([]*service.OpenAIVideoTask, bool, error) {
	where, args := openAIVideoTaskActorWhere(actorUserID, filter)
	args = append(args, filter.Limit+1, filter.Offset)
	rows, err := r.db.QueryContext(ctx, `SELECT `+openAIVideoTaskColumns+` FROM openai_video_tasks WHERE `+where+
		` ORDER BY created_at DESC, id DESC LIMIT $`+strconv.Itoa(len(args)-1)+` OFFSET $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = rows.Close() }()
	tasks := make([]*service.OpenAIVideoTask, 0, filter.Limit+1)
	for rows.Next() {
		task, scanErr := scanOpenAIVideoTask(rows)
		if scanErr != nil {
			return nil, false, scanErr
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	hasMore := len(tasks) > filter.Limit
	if hasMore {
		tasks = tasks[:filter.Limit]
	}
	return tasks, hasMore, nil
}

func (r *openAIVideoTaskRepository) ModelsForActor(ctx context.Context, actorUserID int64) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT DISTINCT requested_model FROM openai_video_tasks WHERE actor_user_id = $1 ORDER BY requested_model`, actorUserID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	models := []string{}
	for rows.Next() {
		var model string
		if err := rows.Scan(&model); err != nil {
			return nil, err
		}
		models = append(models, model)
	}
	return models, rows.Err()
}

// SummaryForActor counts the filtered tasks; cost is the actual cost captured
// by completed tasks, so unsettled or released tasks add nothing.
func (r *openAIVideoTaskRepository) SummaryForActor(ctx context.Context, actorUserID int64, filter service.VideoTaskUserFilter) (*service.VideoTaskUserSummary, error) {
	where, args := openAIVideoTaskActorWhere(actorUserID, filter)
	summary := &service.VideoTaskUserSummary{}
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*),
		COUNT(*) FILTER (WHERE `+openAIVideoTaskUserCompletedSQL+`),
		COUNT(*) FILTER (WHERE `+openAIVideoTaskUserProcessingSQL+`),
		COUNT(*) FILTER (WHERE `+openAIVideoTaskUserFailedSQL+`),
		COALESCE(SUM(actual_cost) FILTER (WHERE `+openAIVideoTaskUserCompletedSQL+`), 0)
		FROM openai_video_tasks WHERE `+where, args...).
		Scan(&summary.Total, &summary.Completed, &summary.Processing, &summary.Failed, &summary.Cost)
	if err != nil {
		return nil, err
	}
	return summary, nil
}

// GetByTaskIDForActor resolves an upstream task ID only for its actor, so a
// foreign task is indistinguishable from a missing one.
func (r *openAIVideoTaskRepository) GetByTaskIDForActor(ctx context.Context, taskID string, actorUserID int64) (*service.OpenAIVideoTask, error) {
	return scanOpenAIVideoTask(r.db.QueryRowContext(ctx, `SELECT `+openAIVideoTaskColumns+`
 FROM openai_video_tasks WHERE task_id = $1 AND actor_user_id = $2`, taskID, actorUserID))
}
