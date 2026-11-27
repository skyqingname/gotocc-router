package service

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/accountbalance"
	"github.com/gin-gonic/gin"
)

const openAIRelayIncompatibleReason GatewayFailureReason = "openai_relay_incompatible_request"

func isOpenAIRelayIncompatible400(account *Account, status int, message string, payload []byte) bool {
	if status != http.StatusBadRequest || account == nil || !account.IsOpenAIApiKey() || openAIInputTerminalCode(status, payload) != "" {
		return false
	}
	if hit, _, _ := detectOpenAICyberPolicy(payload); hit || isOpenAIContextWindowError(message, payload) {
		return false
	}
	body := strings.ToLower(string(payload))
	message = strings.ToLower(message + "\n" + extractUpstreamErrorMessage(payload))
	for _, keyword := range accountbalance.OpenAIRelayIncompatible400Keywords {
		keyword = strings.ToLower(keyword)
		if strings.Contains(body, keyword) || strings.Contains(message, keyword) {
			return true
		}
	}
	return false
}

func newOpenAIRelayIncompatibleFailoverError(account *Account, headers http.Header, payload []byte) *UpstreamFailoverError {
	return &UpstreamFailoverError{
		StatusCode:        http.StatusBadRequest,
		ResponseHeaders:   headers.Clone(),
		ResponseBody:      payload,
		Stage:             GatewayFailureStageInference,
		Scope:             GatewayFailureScopeAccount,
		Reason:            openAIRelayIncompatibleReason,
		NextAccountAction: NextAccountRetry,
		ClientMessage:     sanitizeOpenAIPassthroughClientErrorText(account, codexRequestRejectionMessage(http.StatusBadRequest, payload)),
	}
}

func (s *OpenAIGatewayService) openAIRelayEventFailover(c *gin.Context, account *Account, headers http.Header, payload []byte, passthrough bool) *UpstreamFailoverError {
	message := extractOpenAISSEErrorMessage(payload)
	if !isOpenAIRelayIncompatible400(account, openAIWSRejectionStatus(payload), message, payload) {
		return nil
	}
	s.recordOpenAIStreamUpstreamError(c, account, passthrough, headers.Get("x-request-id"), "failover", payload, message)
	return newOpenAIRelayIncompatibleFailoverError(account, headers, payload)
}

func (s *OpenAIGatewayService) openAIWSDialFailover(ctx context.Context, c *gin.Context, account *Account, err error, passthrough bool) *UpstreamFailoverError {
	var dialErr *openAIWSDialError
	if !errors.As(err, &dialErr) {
		return nil
	}
	body := s.redactAgentIdentitySensitiveBody(ctx, account, dialErr.ResponseBody)
	message := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(body)))
	if !isOpenAIRelayIncompatible400(account, dialErr.StatusCode, message, body) {
		return nil
	}
	setOpsUpstreamError(c, dialErr.StatusCode, message, "")
	appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
		Platform:           account.Platform,
		AccountID:          account.ID,
		AccountName:        account.Name,
		ProxyID:            opsUpstreamProxyID(account),
		ProxyName:          opsUpstreamProxyName(account),
		UpstreamStatusCode: dialErr.StatusCode,
		UpstreamRequestID:  dialErr.ResponseHeaders.Get("x-request-id"),
		Passthrough:        passthrough,
		Kind:               "failover",
		Message:            message,
	})
	return newOpenAIRelayIncompatibleFailoverError(account, dialErr.ResponseHeaders, body)
}
