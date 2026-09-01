package service

import (
	"context"
	"errors"
	"strings"
	"time"
)

const (
	OpenAIVideoTaskStatusCreating   = "creating"
	OpenAIVideoTaskStatusPending    = "pending"
	OpenAIVideoTaskStatusProcessing = "processing"
	OpenAIVideoTaskStatusCompleted  = "completed"
	OpenAIVideoTaskStatusFailed     = "failed"
	OpenAIVideoTaskStatusCancelled  = "cancelled"
	OpenAIVideoTaskStatusExpired    = "expired"

	OpenAIVideoBillingStatusNone     = "none"
	OpenAIVideoBillingStatusHeld     = "held"
	OpenAIVideoBillingStatusCaptured = "captured"
	OpenAIVideoBillingStatusReleased = "released"
	OpenAIVideoBillingStatusFailed   = "failed"
)

var (
	ErrOpenAIVideoTaskNotFound      = errors.New("openai video task not found")
	ErrOpenAIVideoTaskConflict      = errors.New("openai video task identity conflict")
	ErrOpenAIVideoTaskLeaseLost     = errors.New("openai video task lease lost")
	ErrOpenAIVideoTaskIDMissing     = errors.New("upstream video task id is missing")
	ErrOpenAIVideoSecondsInvalid    = errors.New("video seconds must be a positive integer")
	ErrOpenAIVideoResolutionInvalid = errors.New("video size does not map to a configured billing resolution")
)

type OpenAIVideoTask struct {
	ID                    int64
	LocalRequestID        string
	TaskID                *string
	ActorUserID           int64
	BillingUserID         int64
	TeamID                *int64
	APIKeyID              int64
	GroupID               int64
	ChannelID             *int64
	AccountID             int64
	SubscriptionID        *int64
	RequestedModel        string
	UpstreamModel         string
	RequestSeconds        int
	Resolution            string
	Status                string
	UpstreamStatus        *string
	BillingType           int8
	BillingStatus         string
	TotalCost             float64
	ActualCost            *float64
	HoldAmount            float64
	GroupRateMultiplier   float64
	AccountRateMultiplier float64
	AllowanceReserved     bool
	RequestPayloadHash    string
	InboundEndpoint       string
	UpstreamEndpoint      string
	ModelMappingChain     *string
	UserAgent             *string
	IPAddress             *string
	RetryCount            int
	NextPollAt            *time.Time
	LeaseUntil            *time.Time
	LeaseToken            *string
	LastErrorCode         *string
	LastErrorMessage      *string
	UsageRecorded         bool
	CreatedAt             time.Time
	UpdatedAt             time.Time
	SubmittedAt           *time.Time
	FinishedAt            *time.Time
	SettledAt             *time.Time
	UsageRecordedAt       *time.Time
}

type CreateOpenAIVideoTaskParams struct {
	LocalRequestID        string
	ActorUserID           int64
	BillingUserID         int64
	TeamID                *int64
	APIKeyID              int64
	GroupID               int64
	ChannelID             *int64
	AccountID             int64
	SubscriptionID        *int64
	RequestedModel        string
	UpstreamModel         string
	RequestSeconds        int
	Resolution            string
	BillingType           int8
	TotalCost             float64
	HoldAmount            float64
	GroupRateMultiplier   float64
	AccountRateMultiplier float64
	RequestPayloadHash    string
	InboundEndpoint       string
	UpstreamEndpoint      string
	ModelMappingChain     *string
	UserAgent             *string
	IPAddress             *string
	NextPollAt            time.Time
}

type OpenAIVideoTaskRepository interface {
	Create(ctx context.Context, params CreateOpenAIVideoTaskParams) (*OpenAIVideoTask, error)
	BindUpstreamTask(ctx context.Context, localRequestID, taskID, upstreamStatus string, nextPollAt time.Time) (*OpenAIVideoTask, error)
	GetByTaskIDForAPIKey(ctx context.Context, taskID string, apiKeyID int64) (*OpenAIVideoTask, error)
	ClaimDue(ctx context.Context, now time.Time, leaseDuration time.Duration, limit int) ([]*OpenAIVideoTask, error)
	RecordPollState(ctx context.Context, id int64, leaseToken, status, upstreamStatus, errorCode, errorMessage string, nextPollAt *time.Time, finishedAt *time.Time) error
	RecordPollError(ctx context.Context, id int64, leaseToken, code, message string, nextPollAt time.Time) error
	MarkCreateFailure(ctx context.Context, id int64, code, message string, at time.Time) error
	MarkBillingCaptured(ctx context.Context, id int64, actualCost float64, at time.Time) error
	MarkBillingReleased(ctx context.Context, id int64, at time.Time) error
	MarkBillingFailed(ctx context.Context, id int64, code, message string) error
	MarkUsageRecorded(ctx context.Context, id int64, at time.Time) error
}

type OpenAIVideoBalanceHoldCommand struct {
	TaskID             int64
	LocalRequestID     string
	APIKeyID           int64
	UserID             int64
	ActorUserID        int64
	TeamID             *int64
	HoldAmount         float64
	ActualAmount       float64
	AllowanceReserved  bool
	ReservedAt         time.Time
	RequestPayloadHash string
	AccountID          int64
	AccountType        string
	AccountQuotaCost   float64
}

type OpenAIVideoBillingRepository interface {
	ReserveOpenAIVideoBalance(ctx context.Context, cmd *OpenAIVideoBalanceHoldCommand) error
	CaptureOpenAIVideoBalance(ctx context.Context, cmd *OpenAIVideoBalanceHoldCommand) error
	ReleaseOpenAIVideoBalance(ctx context.Context, cmd *OpenAIVideoBalanceHoldCommand) error
}

func OpenAIVideoHoldRequestID(localRequestID string) string {
	return "openai_video_hold:" + strings.TrimSpace(localRequestID)
}

func OpenAIVideoCaptureRequestID(taskID string) string {
	return "openai_video_capture:" + strings.TrimSpace(taskID)
}

func OpenAIVideoReleaseRequestID(localRequestID string) string {
	return "openai_video_release:" + strings.TrimSpace(localRequestID)
}

func IsOpenAIVideoTerminalStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case OpenAIVideoTaskStatusCompleted, OpenAIVideoTaskStatusFailed, OpenAIVideoTaskStatusCancelled, OpenAIVideoTaskStatusExpired:
		return true
	default:
		return false
	}
}
