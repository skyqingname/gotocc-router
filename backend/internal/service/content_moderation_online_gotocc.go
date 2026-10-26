package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/ctxkey"
	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
)

type ContentModerationOnlineTestInput struct {
	EndpointID string `json:"endpoint_id" binding:"required"`
	Prompt     string `json:"prompt" binding:"required"`
}

type ContentModerationOnlineTestResult struct {
	EndpointID      string                            `json:"endpoint_id"`
	EndpointName    string                            `json:"endpoint_name"`
	Model           string                            `json:"model"`
	APIKeyMasked    string                            `json:"api_key_masked"`
	HTTPStatus      int                               `json:"http_status"`
	LatencyMS       int64                             `json:"latency_ms"`
	BlockingEnabled bool                              `json:"blocking_enabled"`
	AuditResult     *ContentModerationTestAuditResult `json:"audit_result"`
	Error           string                            `json:"error"`
}

// TestText calls one saved endpoint directly, including paused/cooling endpoints
// and frozen keys, so administrators can diagnose its configuration. It does not
// enqueue audit records, update endpoint/key health, or apply user penalties.
func (s *ContentModerationService) TestText(ctx context.Context, input ContentModerationOnlineTestInput) (*ContentModerationOnlineTestResult, error) {
	prompt := normalizeContentModerationText(input.Prompt)
	if strings.TrimSpace(prompt) == "" {
		return nil, infraerrors.BadRequest("EMPTY_MODERATION_TEST_INPUT", "请输入待审核内容")
	}
	if utf8.RuneCountInString(prompt) > maxModerationInputRunes {
		return nil, infraerrors.BadRequest("MODERATION_TEST_INPUT_TOO_LONG", fmt.Sprintf("待审核内容最多 %d 字", maxModerationInputRunes))
	}
	cfg, err := s.loadConfig(ctx)
	if err != nil {
		return nil, err
	}
	var selected *ContentModerationEndpoint
	for i := range cfg.Endpoints {
		if cfg.Endpoints[i].ID == input.EndpointID {
			selected = &cfg.Endpoints[i]
			break
		}
	}
	if selected == nil {
		return nil, infraerrors.NotFound("CONTENT_MODERATION_ENDPOINT_NOT_FOUND", "内容审计平台不存在，请保存配置后重试")
	}
	keys := selected.APIKeys
	if len(keys) == 0 {
		return nil, infraerrors.BadRequest("CONTENT_MODERATION_API_KEY_MISSING", "所选审核节点未配置 API Key")
	}
	key := keys[int(s.apiKeyCursor.Add(1)-1)%len(keys)]
	cfg.Protocol, cfg.BaseURL, cfg.Model = selected.Protocol, selected.BaseURL, selected.Model
	cfg.ProxyID, cfg.TimeoutMS = selected.ProxyID, selected.TimeoutMS
	result := &ContentModerationOnlineTestResult{
		EndpointID: selected.ID, EndpointName: selected.Name, Model: selected.Model,
		APIKeyMasked: maskSecretTail(key),
		BlockingEnabled: s.isRiskControlEnabled(ctx) && cfg.Enabled && selected.Enabled && !selected.ManualPaused &&
			cfg.Mode == ContentModerationModePreBlock && cfg.effectiveTextAPIMode(false) == ContentModerationTextAPIModeBlocking &&
			cfg.KeywordBlockingMode != ContentModerationKeywordModeKeywordOnly,
	}
	started := time.Now()
	audit, callErr := s.callModerationOnceWithInput(ctx, cfg, key, prompt, &result.HTTPStatus)
	result.LatencyMS = time.Since(started).Milliseconds()
	if callErr != nil {
		requestID, _ := ctx.Value(ctxkey.RequestID).(string)
		slog.WarnContext(ctx, "content_moderation.online_test_failed",
			"request_id", requestID,
			"endpoint", "/api/v1/admin/risk-control/test-text",
			"protocol", cfg.Protocol,
			"stage", "online_test",
			"error_code", contentModerationNotReviewedReason(callErr),
			"error_kind", contentModerationErrorKind(callErr),
			"input_bytes", len(input.Prompt),
			"audit_input_bytes", len(prompt),
			"upstream_http_status", result.HTTPStatus,
			"latency_ms", result.LatencyMS,
		)
		result.Error = callErr.Error()
		return result, nil
	}
	result.AuditResult = buildContentModerationTestAuditResult(audit, cfg.Thresholds)
	return result, nil
}
