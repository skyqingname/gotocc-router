package service

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/config"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/logger"
	"go.uber.org/zap"
)

type OpenAIVideoTaskRuntime struct {
	gateway *OpenAIGatewayService
	repo    OpenAIVideoTaskRepository
	cfg     *config.Config

	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

func ProvideOpenAIVideoTaskRuntime(
	gateway *OpenAIGatewayService,
	repo OpenAIVideoTaskRepository,
	billing OpenAIVideoBillingRepository,
	apiKeys APIKeyRepository,
	authCache APIKeyAuthCacheInvalidator,
	cfg *config.Config,
) *OpenAIVideoTaskRuntime {
	gateway.ConfigureOpenAIVideoTasks(repo, billing, apiKeys, authCache)
	runtime := &OpenAIVideoTaskRuntime{gateway: gateway, repo: repo, cfg: cfg}
	runtime.Start()
	return runtime
}

func (r *OpenAIVideoTaskRuntime) Start() {
	if r == nil || r.gateway == nil || r.repo == nil || r.cfg == nil || !r.cfg.VideoTask.Enabled {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	r.done = make(chan struct{})
	go r.run(ctx, r.done)
}

func (r *OpenAIVideoTaskRuntime) Stop() {
	if r == nil {
		return
	}
	r.mu.Lock()
	cancel, done := r.cancel, r.done
	r.cancel, r.done = nil, nil
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

func (r *OpenAIVideoTaskRuntime) run(ctx context.Context, done chan struct{}) {
	defer close(done)
	interval := time.Duration(r.cfg.VideoTask.ScanIntervalSeconds) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		r.processDue(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (r *OpenAIVideoTaskRuntime) processDue(ctx context.Context) {
	lease := time.Duration(r.cfg.VideoTask.LeaseSeconds) * time.Second
	for processed := 0; processed < r.cfg.VideoTask.ClaimBatchSize; processed++ {
		if ctx.Err() != nil {
			return
		}
		// Claim one row immediately before processing it. Claiming the whole batch
		// up front would let later rows outlive their lease while an earlier
		// upstream poll is still waiting on I/O.
		tasks, err := r.repo.ClaimDue(ctx, time.Now(), lease, 1)
		if err != nil {
			logger.L().Error("openai_video.worker_claim_failed", zap.Error(err))
			return
		}
		if len(tasks) == 0 {
			return
		}
		task := tasks[0]
		if err := r.processTask(ctx, task); err != nil {
			logger.L().Error("openai_video.worker_task_failed",
				zap.Int64("task_row_id", task.ID), zap.String("task_id", derefOpenAIVideoString(task.TaskID)),
				zap.String("status", task.Status), zap.String("billing_status", task.BillingStatus), zap.Error(err))
		}
	}
}

func (r *OpenAIVideoTaskRuntime) processTask(ctx context.Context, task *OpenAIVideoTask) error {
	if task == nil {
		return nil
	}
	if task.Status == OpenAIVideoTaskStatusCompleted {
		if task.BillingStatus != OpenAIVideoBillingStatusCaptured {
			if err := r.gateway.settleOpenAIVideoTask(ctx, task); err != nil {
				return err
			}
			task.BillingStatus = OpenAIVideoBillingStatusCaptured
		}
		if !task.UsageRecorded {
			return r.gateway.recordOpenAIVideoTaskUsage(ctx, task)
		}
		return nil
	}
	if task.Status == OpenAIVideoTaskStatusFailed || task.Status == OpenAIVideoTaskStatusCancelled || task.Status == OpenAIVideoTaskStatusExpired {
		return r.gateway.releaseOpenAIVideoTask(ctx, task)
	}

	now := time.Now()
	timeout := time.Duration(r.cfg.VideoTask.TaskTimeoutSeconds) * time.Second
	if task.Status == OpenAIVideoTaskStatusCreating || now.Sub(task.CreatedAt) >= timeout {
		finishedAt := now
		leaseToken := derefOpenAIVideoString(task.LeaseToken)
		if err := r.repo.RecordPollState(ctx, task.ID, leaseToken, OpenAIVideoTaskStatusExpired, derefOpenAIVideoString(task.UpstreamStatus), "VIDEO_TASK_EXPIRED", "video task exceeded configured lifetime", nil, &finishedAt); err != nil {
			return err
		}
		task.Status = OpenAIVideoTaskStatusExpired
		task.FinishedAt = &finishedAt
		task.LastErrorCode = optionalTrimmedStringPtr("VIDEO_TASK_EXPIRED")
		task.LastErrorMessage = optionalTrimmedStringPtr("video task exceeded configured lifetime")
		return r.gateway.releaseOpenAIVideoTask(ctx, task)
	}

	account, err := r.gateway.accountRepo.GetByID(ctx, task.AccountID)
	if err != nil {
		return r.recordPollError(ctx, task, "VIDEO_ACCOUNT_LOAD_FAILED", err)
	}
	pollCtx, cancel := context.WithTimeout(ctx, time.Duration(r.cfg.VideoTask.RequestTimeoutSeconds)*time.Second)
	poll, err := r.gateway.PollOpenAIVideoTask(pollCtx, task, account)
	cancel()
	if err != nil {
		return r.recordPollError(ctx, task, "VIDEO_STATUS_POLL_FAILED", err)
	}
	normalized := normalizeOpenAIVideoProviderStatus(r.cfg.VideoTask, poll.ProviderStatus)
	var nextPollAt *time.Time
	var finishedAt *time.Time
	if IsOpenAIVideoTerminalStatus(normalized) {
		finished := time.Now()
		finishedAt = &finished
	} else {
		next := time.Now().Add(time.Duration(r.cfg.VideoTask.ScanIntervalSeconds) * time.Second)
		nextPollAt = &next
	}
	errorCode, errorMessage := "", ""
	switch normalized {
	case OpenAIVideoTaskStatusFailed:
		errorCode, errorMessage = poll.ErrorCode, poll.ErrorMessage
		if strings.TrimSpace(errorCode) == "" {
			errorCode = "VIDEO_PROVIDER_FAILED"
		}
	case OpenAIVideoTaskStatusCancelled:
		errorCode, errorMessage = poll.ErrorCode, poll.ErrorMessage
		if strings.TrimSpace(errorCode) == "" {
			errorCode = "VIDEO_PROVIDER_CANCELLED"
		}
	}
	if err := r.repo.RecordPollState(ctx, task.ID, derefOpenAIVideoString(task.LeaseToken), normalized, poll.ProviderStatus, errorCode, errorMessage, nextPollAt, finishedAt); err != nil {
		return err
	}
	task.Status = normalized
	task.UpstreamStatus = optionalTrimmedStringPtr(poll.ProviderStatus)
	task.NextPollAt = nextPollAt
	task.FinishedAt = finishedAt
	task.LastErrorCode = optionalTrimmedStringPtr(errorCode)
	task.LastErrorMessage = optionalTrimmedStringPtr(errorMessage)
	if normalized == OpenAIVideoTaskStatusCompleted {
		if err := r.gateway.settleOpenAIVideoTask(ctx, task); err != nil {
			return err
		}
		task.BillingStatus = OpenAIVideoBillingStatusCaptured
		return r.gateway.recordOpenAIVideoTaskUsage(ctx, task)
	}
	if IsOpenAIVideoTerminalStatus(normalized) {
		return r.gateway.releaseOpenAIVideoTask(ctx, task)
	}
	return nil
}

func (r *OpenAIVideoTaskRuntime) recordPollError(ctx context.Context, task *OpenAIVideoTask, code string, cause error) error {
	next := time.Now().Add(time.Duration(r.cfg.VideoTask.ScanIntervalSeconds) * time.Second)
	safeCause := errors.New(truncateOpenAIVideoError(sanitizeUpstreamErrorMessage(cause.Error())))
	err := r.repo.RecordPollError(ctx, task.ID, derefOpenAIVideoString(task.LeaseToken), code, safeCause.Error(), next)
	if err != nil {
		return errors.Join(safeCause, err)
	}
	return safeCause
}

func (s *OpenAIGatewayService) settleOpenAIVideoTask(ctx context.Context, task *OpenAIVideoTask) error {
	account, err := s.accountRepo.GetByID(ctx, task.AccountID)
	if err != nil {
		return err
	}
	actualCost := QuantizeUsageBillingAmount(task.HoldAmount)
	if task.BillingType == BillingTypeBalance {
		if err := s.openAIVideoBillingRepo.CaptureOpenAIVideoBalance(ctx, s.openAIVideoHoldCommand(task, account, actualCost)); err != nil {
			return err
		}
		task.ActualCost = &actualCost
		s.invalidateOpenAIVideoBillingCaches(ctx, task)
		s.recordOpenAIVideoPlatformQuota(ctx, task, actualCost)
	} else {
		apiKey, err := s.openAIVideoAPIKeyRepo.GetByID(ctx, task.APIKeyID)
		if err != nil {
			return err
		}
		actualCost = QuantizeUsageBillingAmount(task.TotalCost * task.GroupRateMultiplier)
		cmd := &UsageBillingCommand{
			RequestID:          "openai-video:" + derefOpenAIVideoString(task.TaskID),
			RequestPayloadHash: task.RequestPayloadHash,
			APIKeyID:           task.APIKeyID, UserID: task.BillingUserID,
			ActorUserID: task.ActorUserID, TeamID: task.TeamID,
			AccountID: task.AccountID, AccountType: account.Type,
			SubscriptionID: task.SubscriptionID, Model: task.UpstreamModel,
			BillingType: BillingTypeSubscription, SubscriptionCost: actualCost,
		}
		if apiKey.Quota > 0 {
			cmd.APIKeyQuotaCost = actualCost
		}
		if apiKey.HasRateLimits() {
			cmd.APIKeyRateLimitCost = actualCost
		}
		if account.IsAPIKeyOrBedrock() && account.HasAnyQuotaLimit() {
			cmd.AccountQuotaCost = task.TotalCost * task.AccountRateMultiplier
		}
		if _, err := s.usageBillingRepo.Apply(ctx, cmd); err != nil {
			_ = s.openAIVideoTaskRepo.MarkBillingFailed(ctx, task.ID, "VIDEO_SUBSCRIPTION_SETTLEMENT_FAILED", truncateOpenAIVideoError(err.Error()))
			return err
		}
		if err := s.openAIVideoTaskRepo.MarkBillingCaptured(ctx, task.ID, actualCost, time.Now()); err != nil && !errors.Is(err, ErrOpenAIVideoTaskNotFound) {
			return err
		}
		task.ActualCost = &actualCost
		if s.billingCacheService != nil {
			_ = s.billingCacheService.RefreshSubscription(ctx, task.BillingUserID, task.GroupID)
		}
		if s.openAIVideoAuthCache != nil {
			s.openAIVideoAuthCache.InvalidateAuthCacheByUserID(ctx, task.BillingUserID)
		}
	}
	if s.deferredService != nil {
		s.deferredService.ScheduleLastUsedUpdate(task.AccountID)
	}
	logger.L().Info("openai_video.task_settled",
		zap.Int64("task_row_id", task.ID), zap.String("task_id", derefOpenAIVideoString(task.TaskID)),
		zap.Int("request_seconds", task.RequestSeconds), zap.Float64("actual_cost", actualCost),
		zap.Int8("billing_type", task.BillingType))
	return nil
}

func (s *OpenAIGatewayService) releaseOpenAIVideoTask(ctx context.Context, task *OpenAIVideoTask) error {
	if task.BillingStatus == OpenAIVideoBillingStatusReleased {
		return nil
	}
	if task.BillingType == BillingTypeBalance && task.BillingStatus == OpenAIVideoBillingStatusHeld {
		account, err := s.accountRepo.GetByID(ctx, task.AccountID)
		if err != nil {
			return err
		}
		if err := s.openAIVideoBillingRepo.ReleaseOpenAIVideoBalance(ctx, s.openAIVideoHoldCommand(task, account, 0)); err != nil {
			return err
		}
		s.invalidateOpenAIVideoBillingCaches(ctx, task)
	} else if task.BillingType == BillingTypeBalance && task.BillingStatus == OpenAIVideoBillingStatusNone {
		// A failed pre-upstream reservation leaves a durable task row but no
		// frozen funds. Close that billing state explicitly so the worker does not
		// claim the same terminal row forever.
		if err := s.openAIVideoTaskRepo.MarkBillingReleased(ctx, task.ID, time.Now()); err != nil && !errors.Is(err, ErrOpenAIVideoTaskNotFound) {
			return err
		}
	} else if task.BillingType == BillingTypeSubscription && task.BillingStatus != OpenAIVideoBillingStatusCaptured {
		if err := s.openAIVideoTaskRepo.MarkBillingReleased(ctx, task.ID, time.Now()); err != nil && !errors.Is(err, ErrOpenAIVideoTaskNotFound) {
			return err
		}
	}
	task.BillingStatus = OpenAIVideoBillingStatusReleased
	logger.L().Info("openai_video.task_released",
		zap.Int64("task_row_id", task.ID), zap.String("task_id", derefOpenAIVideoString(task.TaskID)),
		zap.String("terminal_status", task.Status), zap.String("error_code", derefOpenAIVideoString(task.LastErrorCode)),
		zap.Float64("released_amount", task.HoldAmount))
	return nil
}

func (s *OpenAIGatewayService) recordOpenAIVideoTaskUsage(ctx context.Context, task *OpenAIVideoTask) error {
	if task == nil || task.TaskID == nil || task.UsageRecorded {
		return nil
	}
	actualCost := task.TotalCost * task.GroupRateMultiplier
	if task.ActualCost != nil {
		actualCost = *task.ActualCost
	}
	billingMode := string(BillingModeVideo)
	durationMs := int(time.Since(task.CreatedAt).Milliseconds())
	complete := true
	usage := &UsageLog{
		UserID: task.ActorUserID, BillingUserID: task.BillingUserID, TeamID: task.TeamID,
		APIKeyID: task.APIKeyID, AccountID: task.AccountID,
		RequestID: *task.TaskID, Model: task.RequestedModel, RequestedModel: task.RequestedModel,
		UpstreamModel: optionalTrimmedStringPtr(task.UpstreamModel), ChannelID: task.ChannelID,
		ModelMappingChain: task.ModelMappingChain, BillingMode: &billingMode,
		InboundEndpoint:  optionalTrimmedStringPtr(task.InboundEndpoint),
		UpstreamEndpoint: optionalTrimmedStringPtr(task.UpstreamEndpoint),
		GroupID:          &task.GroupID, SubscriptionID: task.SubscriptionID,
		TotalCost: task.TotalCost, ActualCost: actualCost,
		RateMultiplier:        task.GroupRateMultiplier,
		AccountRateMultiplier: &task.AccountRateMultiplier,
		BillingType:           task.BillingType, RequestType: RequestTypeSync,
		DurationMs: &durationMs, IsComplete: &complete,
		UserAgent: task.UserAgent, IPAddress: task.IPAddress,
		VideoCount: 1, VideoResolution: &task.Resolution,
		VideoDurationSeconds: &task.RequestSeconds, CreatedAt: time.Now(),
	}
	accountStatsCost := task.TotalCost * task.AccountRateMultiplier
	usage.AccountStatsCost = &accountStatsCost
	if _, err := s.usageLogRepo.Create(ctx, usage); err != nil {
		return err
	}
	if err := s.openAIVideoTaskRepo.MarkUsageRecorded(ctx, task.ID, time.Now()); err != nil {
		return err
	}
	task.UsageRecorded = true
	return nil
}

func (s *OpenAIGatewayService) recordOpenAIVideoPlatformQuota(ctx context.Context, task *OpenAIVideoTask, actualCost float64) {
	if s == nil || task == nil || actualCost <= 0 || s.billingCacheService == nil || s.userPlatformQuotaRepo == nil {
		return
	}
	platform := PlatformOpenAI
	if !s.billingCacheService.HasUserPlatformQuotaLimit(ctx, task.BillingUserID, platform) {
		return
	}
	s.billingCacheService.IncrementUserPlatformQuotaUsage(task.BillingUserID, platform, actualCost)
	if s.cfg == nil || !s.cfg.Database.UserPlatformQuotaFlusherEnabled {
		if err := s.userPlatformQuotaRepo.IncrementUsageWithReset(ctx, task.BillingUserID, platform, actualCost, time.Now().UTC()); err != nil {
			logger.L().Error("openai_video.platform_quota_update_failed", zap.Int64("user_id", task.BillingUserID), zap.Error(err))
		}
	}
}

func derefOpenAIVideoString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
