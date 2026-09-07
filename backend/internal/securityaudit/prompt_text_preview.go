package securityaudit

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"

	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
)

type TextPreviewRequest struct {
	Text string `json:"text"`
}

type TextPreviewResult struct {
	OK                  bool              `json:"ok"`
	Decision            DecisionKind      `json:"decision"`
	WouldBlock          bool              `json:"would_block"`
	EffectiveMode       Mode              `json:"effective_mode"`
	ConfigVersion       int64             `json:"config_version"`
	ResponseFormat      string            `json:"response_format"`
	ConfidenceThreshold float64           `json:"confidence_threshold"`
	LatencyMS           int               `json:"latency_ms"`
	Result              *NormalizedResult `json:"result,omitempty"`
	ErrorCode           string            `json:"error_code,omitempty"`
	ErrorKind           string            `json:"error_kind,omitempty"`
	GuardEndpointID     string            `json:"guard_endpoint_id,omitempty"`
	HTTPStatus          int               `json:"http_status,omitempty"`
}

// PreviewText runs saved policy through the real extraction, chunking and
// evaluator path. It does not enable blocking or create a user audit event.
func (s *PromptService) PreviewText(ctx context.Context, request TextPreviewRequest, requestID string) (TextPreviewResult, error) {
	if strings.TrimSpace(request.Text) == "" || utf8.RuneCountInString(request.Text) > DefaultTextTestMaxRunes {
		return TextPreviewResult{}, infraerrors.BadRequest("prompt_audit_invalid_preview_text", "请输入有效文本，长度不能超过页面显示的上限")
	}
	cfg, ok := s.config.Active()
	if !ok {
		return TextPreviewResult{}, infraerrors.ServiceUnavailable(ErrorCodeConfigUnavailable, "提示词审计配置暂不可用")
	}
	if len(cfg.EnabledEndpoints()) == 0 {
		return TextPreviewResult{}, infraerrors.BadRequest("prompt_audit_no_enabled_endpoint", "请先保存至少一个启用的审计节点")
	}
	body, err := json.Marshal(map[string]any{
		"input": []map[string]string{{"role": "user", "content": request.Text}},
	})
	if err != nil {
		return TextPreviewResult{}, err
	}
	snapshot, err := ExtractBlockingPromptSnapshot(Request{
		RequestID: requestID, Protocol: "openai_responses", Endpoint: "/admin/prompt-audit/test",
		Provider: "openai", Stage: "admin_text_preview", Body: body,
	}, cfg.BlockingLatestTurnOnly)
	if err != nil {
		return TextPreviewResult{}, err
	}
	started := s.clock.Now()
	decision, scanErr := s.evaluator.Preview(ctx, cfg, snapshot)
	result := TextPreviewResult{
		EffectiveMode: cfg.EffectiveMode(), ConfigVersion: cfg.ConfigVersion,
		ResponseFormat: cfg.ResponseFormat, ConfidenceThreshold: cfg.ConfidenceThreshold,
		LatencyMS: int(s.clock.Now().Sub(started).Milliseconds()),
	}
	if scanErr != nil {
		result.Decision = DecisionUnavailable
		result.ErrorCode, result.ErrorKind = guardErrorCode(scanErr), guardErrorKind(scanErr)
		if result.ErrorCode == ErrorCodeInvalidResponse {
			result.Decision = DecisionInvalid
		}
		var failure *GuardError
		if errors.As(scanErr, &failure) {
			result.HTTPStatus, result.GuardEndpointID = failure.HTTPStatus, failure.EndpointID
		}
		return result, nil
	}
	result.OK, result.Decision, result.Result = true, decision.Kind, decision.Result
	result.WouldBlock = decision.Kind == DecisionBlock
	if decision.Result != nil {
		result.GuardEndpointID = decision.Result.GuardEndpointID
	}
	return result, nil
}

func guardErrorEndpoint(err error) string {
	var failure *GuardError
	if errors.As(err, &failure) {
		return failure.EndpointID
	}
	return ""
}

func guardErrorKind(err error) string {
	var failure *GuardError
	if errors.As(err, &failure) {
		switch {
		case failure.Timeout || errors.Is(err, context.DeadlineExceeded):
			return "timeout"
		case failure.HTTPStatus > 0:
			return "upstream_http"
		case failure.Code == ErrorCodeInvalidResponse:
			return "invalid_response"
		}
	}
	if errors.Is(err, context.Canceled) {
		return "canceled"
	}
	return "unavailable"
}
