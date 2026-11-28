package service

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/accountbalance"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
)

const openAIUpstreamUnavailableReason GatewayFailureReason = "openai_upstream_temporarily_unavailable"
const OpenAIUpstreamUnavailablePublicKey = "upstream_unavailable"

func openAIUpstreamUnavailableHTTPMessage(account *Account, status int, body []byte, afterWSFallback bool) (string, bool) {
	if account == nil || !account.IsOpenAI() {
		return "", false
	}
	if hit, _, _ := detectOpenAICyberPolicy(body); hit ||
		openAIInputTerminalCode(status, body) != "" || isOpenAIContextWindowError("", body) {
		return "", false
	}
	message := strings.TrimSpace(extractUpstreamErrorMessage(body))
	if afterWSFallback && openAIWSFallbackHTTPStatusUnavailable(status) {
		if message == "" {
			message = http.StatusText(status)
		}
		return message, true
	}
	if (status != http.StatusBadGateway && status != http.StatusServiceUnavailable) || isOpenAIHTTPCapacityShed(status, message, body) {
		return "", false
	}
	upstreamText := strings.ToLower(string(body) + "\n" + message)
	for _, keyword := range accountbalance.OpenAIUpstreamUnavailableHTTPKeywords {
		if strings.Contains(upstreamText, strings.ToLower(keyword)) {
			if message == "" {
				message = keyword
			}
			return message, true
		}
	}
	return "", false
}

func (s *OpenAIGatewayService) openAIUpstreamUnavailableHTTPFailover(
	ctx context.Context, c *gin.Context, account *Account, resp *http.Response, body []byte, passthrough bool,
) *UpstreamFailoverError {
	message, unavailable := openAIUpstreamUnavailableHTTPMessage(account, resp.StatusCode, body, isOpenAIWSAvailabilityHTTPFallback(ctx, account))
	if !unavailable {
		return nil
	}
	message = sanitizeUpstreamErrorMessage(message)
	setOpsUpstreamError(c, resp.StatusCode, message, "")
	appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
		Platform: account.Platform, AccountID: account.ID, AccountName: account.Name,
		ProxyID: opsUpstreamProxyID(account), ProxyName: opsUpstreamProxyName(account),
		UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: resp.Header.Get("x-request-id"),
		Passthrough: passthrough, Kind: "failover", Reason: string(openAIUpstreamUnavailableReason), Message: message,
	})
	return s.cooldownOpenAIUnavailableAccount(ctx, account, resp.StatusCode, resp.Header, message)
}

const openAIWSAvailabilityFallbackReason = "upstream_ws_temporarily_unavailable"

type openAIWSAvailabilityHTTPFallbackKey struct{}

func WithOpenAIWSAvailabilityHTTPFallback(ctx context.Context, accountID int64) context.Context {
	return context.WithValue(ctx, openAIWSAvailabilityHTTPFallbackKey{}, accountID)
}

func isOpenAIWSAvailabilityHTTPFallback(ctx context.Context, account *Account) bool {
	if ctx == nil {
		return false
	}
	id, _ := ctx.Value(openAIWSAvailabilityHTTPFallbackKey{}).(int64)
	return account != nil && account.IsOpenAI() && id == account.ID
}

func openAIWSFallbackHTTPStatusUnavailable(status int) bool {
	return status == http.StatusTooManyRequests || status >= 500 && status <= 599
}

func IsOpenAIWSAvailabilityFallback(err error) bool {
	var fallback *openAIWSFallbackError
	return errors.As(err, &fallback) && fallback.Reason == openAIWSAvailabilityFallbackReason
}

func (s *OpenAIGatewayService) openAIUpstreamUnavailableWSFallback(
	c *gin.Context, account *Account, headers http.Header, err error, passthrough bool,
) error {
	if !account.IsOpenAI() {
		return nil
	}
	var closeErr coderws.CloseError
	var dialErr *openAIWSDialError
	var message string
	var upstreamStatus, closeCode int
	switch {
	case errors.As(err, &closeErr) && closeErr.Code == coderws.StatusTryAgainLater:
		closeCode = int(closeErr.Code)
		message = closeErr.Reason
	case errors.As(err, &dialErr):
		upstreamStatus = dialErr.StatusCode
		switch upstreamStatus {
		case http.StatusNotFound, http.StatusMethodNotAllowed, http.StatusUpgradeRequired:
		default:
			if !openAIWSFallbackHTTPStatusUnavailable(upstreamStatus) {
				return nil
			}
		}
		headers = dialErr.ResponseHeaders
		message = extractUpstreamErrorMessage(dialErr.ResponseBody)
		if strings.TrimSpace(message) == "" {
			message = http.StatusText(upstreamStatus)
		}
	default:
		return nil
	}
	s.markOpenAIWSFallbackCooling(account.ID, openAIWSAvailabilityFallbackReason)
	proxyID, proxyName := opsUpstreamWSProxyAttribution(account)
	appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
		Platform: account.Platform, AccountID: account.ID, AccountName: account.Name, ProxyID: proxyID, ProxyName: proxyName,
		UpstreamStatusCode: upstreamStatus, UpstreamWSCloseCode: closeCode, UpstreamRequestID: headers.Get("x-request-id"),
		Passthrough: passthrough, Kind: "ws_fallback", Reason: openAIWSAvailabilityFallbackReason, Message: sanitizeUpstreamErrorMessage(message),
	})
	return wrapOpenAIWSFallback(openAIWSAvailabilityFallbackReason, err)
}

func (s *OpenAIGatewayService) cooldownOpenAIUnavailableAccount(
	ctx context.Context, account *Account, upstreamStatus int, headers http.Header, message string,
) *UpstreamFailoverError {
	until := time.Now().Add(time.Duration(accountbalance.OpenAIUpstreamUnavailableCooldownSeconds) * time.Second)
	s.BlockAccountScheduling(account, until, string(openAIUpstreamUnavailableReason))
	stateCtx, cancel := openAIAccountStateContext(ctx)
	defer cancel()
	if s.accountRepo != nil {
		if err := s.accountRepo.SetTempUnschedulable(stateCtx, account.ID, until, string(openAIUpstreamUnavailableReason)); err != nil {
			slog.WarnContext(stateCtx, "openai.upstream_unavailable_cooldown_write_failed", "account_id", account.ID, "error", err)
		}
	}
	return &UpstreamFailoverError{
		StatusCode:        upstreamStatus,
		ResponseHeaders:   headers.Clone(),
		Stage:             GatewayFailureStageInference,
		Scope:             GatewayFailureScopeAccount,
		Reason:            openAIUpstreamUnavailableReason,
		NextAccountAction: NextAccountRetry,
		ClientStatusCode:  http.StatusServiceUnavailable,
		ClientMessage:     sanitizeOpenAIPassthroughClientErrorText(account, message),
	}
}

func (e *UpstreamFailoverError) IsOpenAIUpstreamTemporarilyUnavailable() bool {
	return e != nil && e.Reason == openAIUpstreamUnavailableReason
}

func (e *UpstreamFailoverError) isOpenAIAvailabilityFailover() bool {
	return e.Reason == openAIRelayIncompatibleReason || e.IsOpenAIUpstreamTemporarilyUnavailable()
}
