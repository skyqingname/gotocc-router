package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/config"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/logger"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

type OpenAIVideoTaskCreateInput struct {
	APIKey           *APIKey
	Subscription     *UserSubscription
	Account          *Account
	RequestedModel   string
	UpstreamModel    string
	Body             []byte
	ChannelFields    ChannelUsageFields
	InboundEndpoint  string
	UpstreamEndpoint string
	UserAgent        string
	IPAddress        string
}

func (s *OpenAIGatewayService) OpenAIVideoTaskLifecycleEnabled() bool {
	return s != nil && s.cfg != nil && s.cfg.VideoTask.Enabled
}

func (s *OpenAIGatewayService) PrepareOpenAIVideoTask(ctx context.Context, input OpenAIVideoTaskCreateInput) (*OpenAIVideoTask, error) {
	if !s.OpenAIVideoTaskLifecycleEnabled() {
		return nil, errors.New("openai video task lifecycle is disabled")
	}
	if s.openAIVideoTaskRepo == nil || s.openAIVideoBillingRepo == nil {
		return nil, errors.New("openai video task lifecycle dependencies are not configured")
	}
	if input.APIKey == nil || input.APIKey.User == nil || input.APIKey.Group == nil || input.APIKey.GroupID == nil || input.Account == nil {
		return nil, errors.New("openai video task identity is incomplete")
	}
	seconds, resolution, err := s.parseOpenAIVideoBillingRequest(ctx, input.APIKey, input.UpstreamModel, input.Body)
	if err != nil {
		return nil, err
	}

	baseMultiplier := s.cfg.Default.RateMultiplier
	if input.APIKey.GroupID != nil {
		baseMultiplier = s.ResolveUserGroupRateMultiplier(ctx, input.APIKey.User.ID, *input.APIKey.GroupID, input.APIKey.Group.RateMultiplier)
	}
	videoMultiplier := resolveVideoRateMultiplier(input.APIKey, baseMultiplier)
	quoteResult := &OpenAIForwardResult{
		Model: input.RequestedModel, UpstreamModel: input.UpstreamModel,
		VideoCount: 1, VideoResolution: resolution, VideoDurationSeconds: seconds,
	}
	cost := s.calculateOpenAIVideoCost(ctx, input.UpstreamModel, input.APIKey, quoteResult, videoMultiplier)
	if cost == nil {
		return nil, errors.New("openai video pricing is unavailable")
	}
	cost.TotalCost = QuantizeUsageBillingAmount(cost.TotalCost)
	cost.ActualCost = QuantizeUsageBillingAmount(cost.ActualCost)
	billingMode, err := normalizeOpenAIVideoTaskBillingMode(cost.BillingMode)
	if err != nil {
		return nil, err
	}

	billingType := int8(BillingTypeBalance)
	var subscriptionID *int64
	if input.Subscription != nil && input.APIKey.Group.IsSubscriptionType() {
		billingType = BillingTypeSubscription
		id := input.Subscription.ID
		subscriptionID = &id
	}
	holdAmount := float64(0)
	if billingType == BillingTypeBalance {
		holdAmount = cost.ActualCost
	}
	channelID := optionalInt64Ptr(input.ChannelFields.ChannelID)
	modelMappingChain := optionalTrimmedStringPtr(input.ChannelFields.ModelMappingChain)
	userAgent := optionalTrimmedStringPtr(input.UserAgent)
	ipAddress := optionalTrimmedStringPtr(input.IPAddress)
	localRequestID := "video-local:" + generateRequestID()
	now := time.Now()
	task, err := s.openAIVideoTaskRepo.Create(ctx, CreateOpenAIVideoTaskParams{
		LocalRequestID:        localRequestID,
		ActorUserID:           usageActorUserID(input.APIKey, input.APIKey.User),
		BillingUserID:         input.APIKey.User.ID,
		TeamID:                input.APIKey.TeamID,
		APIKeyID:              input.APIKey.ID,
		GroupID:               *input.APIKey.GroupID,
		ChannelID:             channelID,
		AccountID:             input.Account.ID,
		SubscriptionID:        subscriptionID,
		RequestedModel:        strings.TrimSpace(input.RequestedModel),
		UpstreamModel:         strings.TrimSpace(input.UpstreamModel),
		RequestSeconds:        seconds,
		Resolution:            resolution,
		BillingMode:           billingMode,
		BillingType:           billingType,
		TotalCost:             cost.TotalCost,
		HoldAmount:            holdAmount,
		GroupRateMultiplier:   videoMultiplier,
		AccountRateMultiplier: input.Account.BillingRateMultiplier(),
		RequestPayloadHash:    HashUsageRequestPayload(input.Body),
		InboundEndpoint:       input.InboundEndpoint,
		UpstreamEndpoint:      input.UpstreamEndpoint,
		ModelMappingChain:     modelMappingChain,
		UserAgent:             userAgent,
		IPAddress:             ipAddress,
		NextPollAt:            now.Add(time.Duration(s.cfg.VideoTask.TaskTimeoutSeconds) * time.Second),
	})
	if err != nil {
		return nil, err
	}
	if billingType == BillingTypeBalance {
		if err := s.openAIVideoBillingRepo.ReserveOpenAIVideoBalance(ctx, s.openAIVideoHoldCommand(task, input.Account, 0)); err != nil {
			_ = s.openAIVideoTaskRepo.MarkCreateFailure(ctx, task.ID, "VIDEO_BALANCE_HOLD_FAILED", err.Error(), time.Now())
			return nil, err
		}
		task.BillingStatus = OpenAIVideoBillingStatusHeld
		task.AllowanceReserved = holdAmount > 0
		s.invalidateOpenAIVideoBillingCaches(ctx, task)
	}
	logger.L().Info("openai_video.task_prepared",
		zap.Int64("task_row_id", task.ID), zap.String("local_request_id", task.LocalRequestID),
		zap.Int64("api_key_id", task.APIKeyID), zap.Int64("account_id", task.AccountID),
		zap.Int("request_seconds", task.RequestSeconds), zap.String("resolution", task.Resolution),
		zap.String("billing_mode", task.BillingMode),
		zap.Float64("hold_amount", task.HoldAmount), zap.Int8("billing_type", task.BillingType))
	return task, nil
}

func normalizeOpenAIVideoTaskBillingMode(mode string) (string, error) {
	switch BillingMode(strings.TrimSpace(mode)) {
	case BillingModePerRequest:
		return string(BillingModePerRequest), nil
	case BillingModeVideo:
		return string(BillingModeVideo), nil
	default:
		return "", fmt.Errorf("openai video billing mode is invalid: %q", mode)
	}
}

func (s *OpenAIGatewayService) BindOpenAIVideoTaskResponse(_ context.Context, task *OpenAIVideoTask, body []byte) (*OpenAIVideoTask, error) {
	if task == nil {
		return nil, ErrOpenAIVideoTaskNotFound
	}
	taskID, upstreamStatus := parseOpenAIVideoTaskIdentity(body)
	if taskID == "" {
		return nil, ErrOpenAIVideoTaskIDMissing
	}
	// The upstream may already have accepted and started the task when the
	// client disconnects. Persist that identity with a bounded detached context
	// so client cancellation cannot strand an accepted task in `creating`.
	bindCtx, cancel := context.WithTimeout(context.Background(), time.Duration(s.cfg.VideoTask.RequestTimeoutSeconds)*time.Second)
	defer cancel()
	bound, err := s.openAIVideoTaskRepo.BindUpstreamTask(bindCtx, task.LocalRequestID, taskID, upstreamStatus, time.Now())
	if err == nil {
		logger.L().Info("openai_video.task_bound",
			zap.Int64("task_row_id", task.ID), zap.String("task_id", taskID),
			zap.Int64("account_id", task.AccountID), zap.String("upstream_status", upstreamStatus))
	}
	return bound, err
}

func (s *OpenAIGatewayService) FailOpenAIVideoCreate(ctx context.Context, task *OpenAIVideoTask, code string, cause error) error {
	if task == nil || s.openAIVideoTaskRepo == nil {
		return nil
	}
	message := "upstream video create failed"
	if cause != nil {
		message = cause.Error()
	}
	now := time.Now()
	markErr := s.openAIVideoTaskRepo.MarkCreateFailure(ctx, task.ID, code, truncateOpenAIVideoError(message), now)
	if task.BillingType == BillingTypeBalance && task.BillingStatus == OpenAIVideoBillingStatusHeld {
		account, accountErr := s.accountRepo.GetByID(ctx, task.AccountID)
		if accountErr != nil {
			return errors.Join(markErr, accountErr)
		}
		releaseErr := s.openAIVideoBillingRepo.ReleaseOpenAIVideoBalance(ctx, s.openAIVideoHoldCommand(task, account, 0))
		if releaseErr == nil {
			s.invalidateOpenAIVideoBillingCaches(ctx, task)
		}
		return errors.Join(markErr, releaseErr)
	}
	return markErr
}

func (s *OpenAIGatewayService) GetOpenAIVideoTaskForAPIKey(ctx context.Context, taskID string, apiKeyID int64) (*OpenAIVideoTask, *Account, error) {
	if s == nil || s.openAIVideoTaskRepo == nil {
		return nil, nil, errors.New("openai video task repository is not configured")
	}
	task, err := s.openAIVideoTaskRepo.GetByTaskIDForAPIKey(ctx, taskID, apiKeyID)
	if err != nil {
		return nil, nil, err
	}
	account, err := s.accountRepo.GetByID(ctx, task.AccountID)
	if err != nil {
		return nil, nil, err
	}
	return task, account, nil
}

func (s *OpenAIGatewayService) parseOpenAIVideoBillingRequest(ctx context.Context, apiKey *APIKey, billingModel string, body []byte) (int, string, error) {
	secondsValue := gjson.GetBytes(body, "seconds")
	if !secondsValue.Exists() || secondsValue.Type != gjson.Number || secondsValue.Float() != float64(secondsValue.Int()) {
		return 0, "", ErrOpenAIVideoSecondsInvalid
	}
	seconds := int(secondsValue.Int())
	if seconds < VideoBillingMinDurationSeconds || seconds > VideoBillingMaxDurationSeconds {
		return 0, "", fmt.Errorf("%w: allowed range is %d-%d", ErrOpenAIVideoSecondsInvalid, VideoBillingMinDurationSeconds, VideoBillingMaxDurationSeconds)
	}
	resolution, known := openAIVideoResolutionFromBody(body)
	if !known {
		resolved := s.resolveOpenAIChannelPricing(ctx, billingModel, apiKey)
		if resolved != nil && resolved.Mode == BillingModeVideo && len(resolved.RequestTiers) > 0 {
			return 0, "", ErrOpenAIVideoResolutionInvalid
		}
		resolution = VideoBillingResolution480P
	}
	return seconds, resolution, nil
}

func openAIVideoResolutionFromBody(body []byte) (string, bool) {
	if raw := strings.TrimSpace(gjson.GetBytes(body, "resolution").String()); raw != "" {
		return LookupVideoBillingResolution(raw)
	}
	switch strings.ToLower(strings.TrimSpace(gjson.GetBytes(body, "size").String())) {
	case "854x480", "480x854":
		return VideoBillingResolution480P, true
	case "1280x720", "720x1280":
		return VideoBillingResolution720P, true
	case "1920x1080", "1080x1920":
		return VideoBillingResolution1080P, true
	default:
		return "", false
	}
}

func parseOpenAIVideoTaskIdentity(body []byte) (string, string) {
	var taskID string
	for _, path := range []string{"id", "task_id", "request_id", "data.id", "data.task_id", "data.request_id"} {
		if value := strings.TrimSpace(gjson.GetBytes(body, path).String()); value != "" {
			taskID = value
			break
		}
	}
	status := strings.TrimSpace(gjson.GetBytes(body, "status").String())
	if status == "" {
		status = strings.TrimSpace(gjson.GetBytes(body, "data.status").String())
	}
	return taskID, status
}

func (s *OpenAIGatewayService) openAIVideoHoldCommand(task *OpenAIVideoTask, account *Account, actual float64) *OpenAIVideoBalanceHoldCommand {
	accountType := ""
	if account != nil {
		accountType = account.Type
	}
	return &OpenAIVideoBalanceHoldCommand{
		TaskID: task.ID, LocalRequestID: task.LocalRequestID,
		APIKeyID: task.APIKeyID, UserID: task.BillingUserID,
		ActorUserID: task.ActorUserID, TeamID: task.TeamID,
		HoldAmount: task.HoldAmount, ActualAmount: actual,
		AllowanceReserved: task.AllowanceReserved, ReservedAt: task.CreatedAt,
		RequestPayloadHash: task.RequestPayloadHash, AccountID: task.AccountID,
		AccountType: accountType, AccountQuotaCost: task.TotalCost * task.AccountRateMultiplier,
	}
}

func (s *OpenAIGatewayService) invalidateOpenAIVideoBillingCaches(ctx context.Context, task *OpenAIVideoTask) {
	if s == nil || task == nil {
		return
	}
	if s.billingCacheService != nil {
		_ = s.billingCacheService.InvalidateUserBalance(ctx, task.BillingUserID)
	}
	if s.openAIVideoAuthCache != nil {
		s.openAIVideoAuthCache.InvalidateAuthCacheByUserID(ctx, task.BillingUserID)
	}
}

func truncateOpenAIVideoError(message string) string {
	message = strings.TrimSpace(message)
	if len(message) <= 1024 {
		return message
	}
	return message[:1024]
}

func normalizeOpenAIVideoProviderStatus(cfg config.VideoTaskConfig, providerStatus string) string {
	providerStatus = strings.ToLower(strings.TrimSpace(providerStatus))
	for _, status := range cfg.SuccessStatuses {
		if providerStatus == strings.ToLower(strings.TrimSpace(status)) {
			return OpenAIVideoTaskStatusCompleted
		}
	}
	for _, status := range cfg.FailureStatuses {
		if providerStatus == strings.ToLower(strings.TrimSpace(status)) {
			return OpenAIVideoTaskStatusFailed
		}
	}
	for _, status := range cfg.CancelledStatuses {
		if providerStatus == strings.ToLower(strings.TrimSpace(status)) {
			return OpenAIVideoTaskStatusCancelled
		}
	}
	return OpenAIVideoTaskStatusProcessing
}
