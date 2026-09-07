package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	pkghttputil "github.com/LuckyKuang/sub2api-plus/internal/pkg/httputil"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/ip"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/logger"
	middleware2 "github.com/LuckyKuang/sub2api-plus/internal/server/middleware"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

const defaultOpenAIVideoModel = "video-ds-2.0-fast"

func valueOrZeroInt64(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

// Videos handles OpenAI-compatible video task creation.
func (h *OpenAIGatewayHandler) Videos(c *gin.Context) {
	h.handleOpenAIVideoProxy(c, true)
}

// VideoTask handles OpenAI-compatible video task status polling.
func (h *OpenAIGatewayHandler) VideoTask(c *gin.Context) {
	h.handleOpenAIVideoProxy(c, false)
}

// VideoContent handles OpenAI-compatible video content download.
func (h *OpenAIGatewayHandler) VideoContent(c *gin.Context) {
	h.handleOpenAIVideoProxy(c, false)
}

func (h *OpenAIGatewayHandler) handleOpenAIVideoProxy(c *gin.Context, chargeRequest bool) {
	requestStart := time.Now()
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}
	reqLog := requestLogger(
		c,
		"handler.openai_gateway.videos",
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
	)
	if !h.ensureResponsesDependencies(c, reqLog) {
		return
	}
	c.Request = c.Request.WithContext(service.WithOpenAIProfitControlSuppressed(c.Request.Context()))

	var body []byte
	var err error
	var persistedTask *service.OpenAIVideoTask
	var persistedAccount *service.Account
	requestModel := strings.TrimSpace(c.Query("model"))
	if chargeRequest {
		body, err = pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
		if err != nil {
			if maxErr, ok := extractMaxBytesError(err); ok {
				h.errorResponse(c, http.StatusRequestEntityTooLarge, "invalid_request_error", buildBodyTooLargeMessage(maxErr.Limit))
				return
			}
			h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
			return
		}
		if len(body) == 0 {
			h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Request body is empty")
			return
		}
		if !gjson.ValidBytes(body) {
			h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body")
			return
		}
		modelResult := gjson.GetBytes(body, "model")
		if !modelResult.Exists() || modelResult.Type != gjson.String || strings.TrimSpace(modelResult.String()) == "" {
			h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "model is required")
			return
		}
		requestModel = strings.TrimSpace(modelResult.String())
	}
	if !chargeRequest && h.gatewayService.OpenAIVideoTaskLifecycleEnabled() {
		taskID := strings.TrimSpace(c.Param("request_id"))
		if taskID == "" {
			h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "video task id is required")
			return
		}
		persistedTask, persistedAccount, err = h.gatewayService.GetOpenAIVideoTaskForAPIKey(c.Request.Context(), taskID, apiKey.ID)
		if err != nil {
			status := http.StatusBadGateway
			message := "Failed to load video task"
			if errors.Is(err, service.ErrOpenAIVideoTaskNotFound) {
				status = http.StatusNotFound
				message = "Video task not found"
			}
			h.errorResponse(c, status, "invalid_request_error", message)
			return
		}
		requestModel = persistedTask.RequestedModel
	}
	if requestModel == "" {
		requestModel = defaultOpenAIVideoModel
	}
	if chargeRequest {
		if decision := h.checkSecurityAudit(c, reqLog, apiKey, subject, service.ContentModerationProtocolOpenAIImages, requestModel, body); decision != nil && !decision.AllowNextStage {
			h.openAISecurityAuditError(c, decision)
			return
		}
	}

	setOpsRequestContext(c, requestModel, false)
	reqLog = reqLog.With(zap.String("model", requestModel), zap.Bool("charge_request", chargeRequest))
	channelMapping, _ := h.gatewayService.ResolveChannelMappingAndRestrict(c.Request.Context(), apiKey.GroupID, requestModel)
	forwardBody := body
	forwardModel := requestModel
	if persistedTask != nil {
		forwardModel = persistedTask.UpstreamModel
		channelMapping = service.ChannelMappingResult{
			MappedModel: forwardModel,
			ChannelID:   valueOrZeroInt64(persistedTask.ChannelID),
			Mapped:      forwardModel != requestModel,
		}
	} else if channelMapping.Mapped {
		forwardModel = channelMapping.MappedModel
		if len(body) > 0 {
			forwardBody = h.gatewayService.ReplaceModelInBody(body, forwardModel)
		}
	}
	setOpsEndpointContext(c, forwardModel, int16(service.RequestTypeSync))

	subscription, _ := middleware2.GetSubscriptionFromContext(c)
	quotaPlatform := service.QuotaPlatform(c.Request.Context(), apiKey)
	service.SetOpsLatencyMs(c, service.OpsAuthLatencyMsKey, time.Since(requestStart).Milliseconds())

	streamStarted := false
	userRelease, acquired := h.acquireResponsesUserSlot(c, subject.UserID, subject.Concurrency, false, &streamStarted, reqLog)
	if !acquired {
		return
	}
	if userRelease != nil {
		defer userRelease()
	}

	if chargeRequest {
		if err := h.billingCacheService.CheckBillingEligibility(c.Request.Context(), apiKey.User, apiKey, apiKey.Group, subscription, quotaPlatform); err != nil {
			reqLog.Info("openai_videos.billing_eligibility_check_failed", zap.Error(err))
			status, code, message, retryAfter := billingErrorDetails(err)
			applyBillingQuotaHeaders(c, err, retryAfter)
			h.errorResponse(c, status, code, message)
			return
		}
	}

	routingStart := time.Now()
	sessionHash := "openai-video-" + requestModel
	var selection *service.AccountSelectionResult
	account := persistedAccount
	if persistedTask == nil {
		selection, _, err = h.gatewayService.SelectAccountWithScheduler(
			c.Request.Context(), apiKey.GroupID, "", sessionHash, forwardModel, nil,
			service.OpenAIUpstreamTransportHTTPSSE, false,
		)
		if err != nil || selection == nil || selection.Account == nil {
			reqLog.Warn("openai_videos.account_select_failed", zap.Error(err))
			classification := classifyNoAccountErrorFromGin(c, h.gatewayService, apiKey, requestModel, forwardModel, service.PlatformOpenAI)
			if !classification.ModelNotFound {
				markOpsRoutingCapacityLimitedIfNoAvailable(c, err)
			}
			h.errorResponse(c, classification.Status, classification.ErrType, classification.Message)
			return
		}
		account = selection.Account
	}
	setOpsSelectedAccount(c, account.ID, account.Platform)
	var accountRelease func()
	if persistedTask == nil {
		var slotResult openAISlotAcquireResult
		accountRelease, slotResult = h.acquireResponsesAccountSlot(c, apiKey.GroupID, sessionHash, selection, false, &streamStarted, reqLog)
		if slotResult != openAISlotAcquireOK {
			return
		}
	}

	service.SetOpsLatencyMs(c, service.OpsRoutingLatencyMsKey, time.Since(routingStart).Milliseconds())
	userAgent := c.GetHeader("User-Agent")
	clientIP := ip.GetClientIP(c)
	var preparedTask *service.OpenAIVideoTask
	if chargeRequest && h.gatewayService.OpenAIVideoTaskLifecycleEnabled() {
		preparedTask, err = h.gatewayService.PrepareOpenAIVideoTask(c.Request.Context(), service.OpenAIVideoTaskCreateInput{
			APIKey: apiKey, Subscription: subscription, Account: account,
			RequestedModel: requestModel, UpstreamModel: forwardModel, Body: body,
			ChannelFields:   channelMapping.ToUsageFields(requestModel, forwardModel),
			InboundEndpoint: GetInboundEndpoint(c), UpstreamEndpoint: "/v1/videos",
			UserAgent: userAgent, IPAddress: clientIP,
		})
		if err != nil {
			if accountRelease != nil {
				accountRelease()
			}
			if errors.Is(err, service.ErrOpenAIVideoSecondsInvalid) || errors.Is(err, service.ErrOpenAIVideoResolutionInvalid) {
				h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", err.Error())
				return
			}
			status, code, message, retryAfter := billingErrorDetails(err)
			applyBillingQuotaHeaders(c, err, retryAfter)
			if status == http.StatusInternalServerError {
				status, code, message = http.StatusBadGateway, "billing_error", "Failed to reserve video billing"
			}
			h.errorResponse(c, status, code, message)
			return
		}
	}
	forwardStart := time.Now()
	result, forwardErr := h.gatewayService.ForwardVideo(c.Request.Context(), c, account, service.OpenAIVideoForwardInput{
		Method:        c.Request.Method,
		Path:          c.Request.URL.Path,
		Body:          forwardBody,
		Model:         requestModel,
		UpstreamModel: forwardModel,
		LocalRequestID: func() string {
			if preparedTask != nil {
				return preparedTask.LocalRequestID
			}
			return ""
		}(),
		DeferResponseWrite: preparedTask != nil,
	})
	if accountRelease != nil {
		accountRelease()
	}
	service.SetOpsLatencyMs(c, service.OpsResponseLatencyMsKey, time.Since(forwardStart).Milliseconds())
	if persistedTask == nil {
		h.gatewayService.ReportOpenAIAccountScheduleResult(
			account, openAIAccountScheduleModel(c, account, requestModel, false, result),
			openAIForwardSucceededForScheduling(forwardErr, result), nil, forwardErr,
		)
	}
	if forwardErr != nil {
		if preparedTask != nil {
			if releaseErr := h.gatewayService.FailOpenAIVideoCreate(context.Background(), preparedTask, "VIDEO_CREATE_FAILED", forwardErr); releaseErr != nil {
				reqLog.Error("openai_videos.create_failure_release_failed", zap.Error(releaseErr))
			}
		}
		h.ensureForwardErrorResponse(c, false)
		reqLog.Warn("openai_videos.forward_failed", zap.Int64("account_id", account.ID), zap.Error(forwardErr))
		return
	}
	if preparedTask != nil {
		if _, err := h.gatewayService.BindOpenAIVideoTaskResponse(c.Request.Context(), preparedTask, result.ResponseBody); err != nil {
			_ = h.gatewayService.FailOpenAIVideoCreate(context.Background(), preparedTask, "VIDEO_CREATE_RESPONSE_INVALID", err)
			h.errorResponse(c, http.StatusBadGateway, "upstream_error", "Upstream video response did not contain a trackable task id")
			return
		}
		if err := h.gatewayService.WriteOpenAIVideoForwardResponse(c, result); err != nil {
			reqLog.Warn("openai_videos.response_write_failed", zap.Error(err))
		}
		return
	}
	if !chargeRequest || result == nil {
		return
	}

	requestPayloadHash := service.HashUsageRequestPayload(body)
	h.submitOpenAIUsageRecordTask(c.Request.Context(), result, func(ctx context.Context) {
		if err := h.gatewayService.RecordUsage(ctx, &service.OpenAIRecordUsageInput{
			Result:             result,
			APIKey:             apiKey,
			User:               apiKey.User,
			Account:            account,
			Subscription:       subscription,
			InboundEndpoint:    GetInboundEndpoint(c),
			UpstreamEndpoint:   GetUpstreamEndpoint(c, account.Platform),
			UserAgent:          userAgent,
			IPAddress:          clientIP,
			RequestPayloadHash: requestPayloadHash,
			APIKeyService:      h.apiKeyService,
			QuotaPlatform:      quotaPlatform,
			ChannelUsageFields: channelMapping.ToUsageFields(requestModel, result.UpstreamModel),
		}); err != nil {
			logger.L().With(
				zap.String("component", "handler.openai_gateway.videos"),
				zap.Int64("user_id", subject.UserID),
				zap.Int64("api_key_id", apiKey.ID),
				zap.Any("group_id", apiKey.GroupID),
				zap.String("model", requestModel),
				zap.Int64("account_id", account.ID),
			).Error("openai_videos.record_usage_failed", zap.Error(err))
		}
	})
}
