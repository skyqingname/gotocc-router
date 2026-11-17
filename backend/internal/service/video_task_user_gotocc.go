package service

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Task center (GoToCC): the signed-in user's own video tasks. PostgreSQL is the
// task source of truth and content is read with the creation-time account
// (LC-013). Reads never bill or record usage.

// The three states shown to users. The repository maps local statuses onto
// them: failed|cancelled|expired are failed, other non-terminal ones processing.
const (
	VideoTaskUserStatusProcessing = "processing"
	VideoTaskUserStatusCompleted  = "completed"
	VideoTaskUserStatusFailed     = "failed"
)

// Paging follows the console image task list contract.
const (
	VideoTaskUserDefaultLimit = imageTaskDefaultListLimit
	VideoTaskUserMaxLimit     = imageTaskMaxListLimit
)

var (
	ErrVideoTaskUserNotFound           = infraerrors.New(http.StatusNotFound, "VIDEO_TASK_NOT_FOUND", "video task not found")
	ErrVideoTaskUserNotCompleted       = infraerrors.New(http.StatusConflict, "VIDEO_TASK_NOT_COMPLETED", "video task is not completed")
	ErrVideoTaskUserUnavailable        = infraerrors.New(http.StatusServiceUnavailable, "VIDEO_TASK_UNAVAILABLE", "video tasks are unavailable")
	ErrVideoTaskUserContentUnavailable = infraerrors.New(http.StatusBadGateway, "VIDEO_CONTENT_UNAVAILABLE", "video content could not be fetched from the upstream")
)

// VideoTaskUserFilter limits a task center query. Status is one of the three
// user states; Since/Until are unix seconds with an exclusive upper bound.
type VideoTaskUserFilter struct {
	Status string
	Model  string
	Since  int64
	Until  int64
	Limit  int
	Offset int
}

type OpenAIVideoTaskUserRepository interface {
	ListForActor(ctx context.Context, actorUserID int64, filter VideoTaskUserFilter) ([]*OpenAIVideoTask, bool, error)
	ModelsForActor(ctx context.Context, actorUserID int64) ([]string, error)
	SummaryForActor(ctx context.Context, actorUserID int64, filter VideoTaskUserFilter) (*VideoTaskUserSummary, error)
	GetByTaskIDForActor(ctx context.Context, taskID string, actorUserID int64) (*OpenAIVideoTask, error)
}

type VideoTaskUserError struct {
	Code    *string `json:"code"`
	Message *string `json:"message"`
}

// VideoTaskUserView is the user-facing task. Accounts, upstream models,
// local request IDs, IPs and other internal fields stay private.
type VideoTaskUserView struct {
	// ID is the upstream task ID; it is null until the upstream accepts the task.
	ID          *string `json:"id"`
	Model       string  `json:"model"`
	Status      string  `json:"status"`
	Seconds     int     `json:"seconds"`
	Resolution  string  `json:"resolution"`
	BillingMode string  `json:"billing_mode"`
	// Cost is the captured actual cost of a completed task, null while its
	// settlement is still pending, and 0 for tasks that are not completed.
	Cost             *float64            `json:"cost"`
	CreatedAt        int64               `json:"created_at"`
	FinishedAt       *int64              `json:"finished_at"`
	Error            *VideoTaskUserError `json:"error"`
	ContentAvailable bool                `json:"content_available"`
}

type VideoTaskUserListResponse struct {
	Object  string              `json:"object"`
	Data    []VideoTaskUserView `json:"data"`
	HasMore bool                `json:"has_more"`
}

type VideoTaskUserSummary struct {
	Total      int64   `json:"total"`
	Completed  int64   `json:"completed"`
	Processing int64   `json:"processing"`
	Failed     int64   `json:"failed"`
	Cost       float64 `json:"cost"`
}

type VideoTaskUserFilters struct {
	Models []string `json:"models"`
}

type VideoTaskUserService struct {
	tasks    OpenAIVideoTaskUserRepository
	accounts AccountRepository
	gateway  *OpenAIGatewayService
}

func NewVideoTaskUserService(tasks OpenAIVideoTaskUserRepository, accounts AccountRepository, gateway *OpenAIGatewayService) *VideoTaskUserService {
	return &VideoTaskUserService{tasks: tasks, accounts: accounts, gateway: gateway}
}

func (s *VideoTaskUserService) List(ctx context.Context, actorUserID int64, filter VideoTaskUserFilter) (*VideoTaskUserListResponse, error) {
	tasks, hasMore, err := s.tasks.ListForActor(ctx, actorUserID, filter)
	if err != nil {
		return nil, ErrVideoTaskUserUnavailable.WithCause(err)
	}
	data := make([]VideoTaskUserView, 0, len(tasks))
	for _, task := range tasks {
		data = append(data, videoTaskUserView(task))
	}
	return &VideoTaskUserListResponse{Object: "list", Data: data, HasMore: hasMore}, nil
}

func (s *VideoTaskUserService) Filters(ctx context.Context, actorUserID int64) (*VideoTaskUserFilters, error) {
	models, err := s.tasks.ModelsForActor(ctx, actorUserID)
	if err != nil {
		return nil, ErrVideoTaskUserUnavailable.WithCause(err)
	}
	return &VideoTaskUserFilters{Models: models}, nil
}

func (s *VideoTaskUserService) Summary(ctx context.Context, actorUserID int64, filter VideoTaskUserFilter) (*VideoTaskUserSummary, error) {
	summary, err := s.tasks.SummaryForActor(ctx, actorUserID, filter)
	if err != nil {
		return nil, ErrVideoTaskUserUnavailable.WithCause(err)
	}
	return summary, nil
}

// ServeContent streams a completed task's video through the same service path
// as GET /v1/videos/:id/content: ForwardVideo with the task's creation-time
// account, provider snapshot and upstream task ID. The upstream status and
// content type pass through. Nothing is billed.
func (s *VideoTaskUserService) ServeContent(ctx context.Context, c *gin.Context, actorUserID int64, taskID string) error {
	task, err := s.tasks.GetByTaskIDForActor(ctx, taskID, actorUserID)
	if errors.Is(err, ErrOpenAIVideoTaskNotFound) {
		return ErrVideoTaskUserNotFound
	}
	if err != nil {
		return ErrVideoTaskUserUnavailable.WithCause(err)
	}
	if task.Status != OpenAIVideoTaskStatusCompleted {
		return ErrVideoTaskUserNotCompleted
	}
	account, err := s.accounts.GetByID(ctx, task.AccountID)
	if err != nil {
		logger.L().Warn("video_task_user.content_account_failed", zap.Int64("task_row_id", task.ID), zap.Int64("account_id", task.AccountID), zap.Error(err))
		return ErrVideoTaskUserContentUnavailable.WithCause(err)
	}
	_, err = s.gateway.ForwardVideo(ctx, c, account, OpenAIVideoForwardInput{
		ProviderConfig: task.ProviderConfig,
		TaskID:         *task.TaskID,
		Method:         http.MethodGet,
		Path:           "/v1/videos/" + url.PathEscape(*task.TaskID) + "/content",
		Model:          task.RequestedModel,
		UpstreamModel:  task.UpstreamModel,
	})
	if err != nil {
		logger.L().Warn("video_task_user.content_failed", zap.Int64("task_row_id", task.ID), zap.Int64("account_id", task.AccountID), zap.Error(err))
		return ErrVideoTaskUserContentUnavailable.WithCause(err)
	}
	return nil
}

func videoTaskUserView(task *OpenAIVideoTask) VideoTaskUserView {
	view := VideoTaskUserView{
		ID:               task.TaskID,
		Model:            task.RequestedModel,
		Seconds:          task.RequestSeconds,
		Resolution:       task.Resolution,
		BillingMode:      task.BillingMode,
		CreatedAt:        task.CreatedAt.Unix(),
		ContentAvailable: task.Status == OpenAIVideoTaskStatusCompleted,
	}
	if task.FinishedAt != nil {
		finishedAt := task.FinishedAt.Unix()
		view.FinishedAt = &finishedAt
	}
	switch task.Status {
	case OpenAIVideoTaskStatusCompleted:
		view.Status = VideoTaskUserStatusCompleted
		view.Cost = task.ActualCost
	case OpenAIVideoTaskStatusFailed, OpenAIVideoTaskStatusCancelled, OpenAIVideoTaskStatusExpired:
		view.Status = VideoTaskUserStatusFailed
		view.Cost = new(float64)
		view.Error = &VideoTaskUserError{Code: task.LastErrorCode, Message: task.LastErrorMessage}
	default:
		view.Status = VideoTaskUserStatusProcessing
		view.Cost = new(float64)
	}
	return view
}
