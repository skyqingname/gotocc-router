package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/util/responseheaders"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

const openAIPlatformVideosURL = "https://api.openai.com/v1/videos"

var openAIVideoAllowedHeaders = map[string]bool{
	"accept":          true,
	"accept-language": true,
	"content-type":    true,
}

// OpenAIVideoForwardInput describes the OpenAI-compatible video task surface.
type OpenAIVideoForwardInput struct {
	Method             string
	Path               string
	Body               []byte
	Model              string
	UpstreamModel      string
	LocalRequestID     string
	DeferResponseWrite bool
}

func (s *OpenAIGatewayService) buildOpenAIVideoUpstreamRequest(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	input OpenAIVideoForwardInput,
	token string,
) (*http.Request, error) {
	targetURL := openAIPlatformVideosURL
	if account != nil {
		baseURL := account.GetOpenAIBaseURL()
		if baseURL != "" {
			validatedURL, err := s.validateUpstreamBaseURL(baseURL)
			if err != nil {
				return nil, err
			}
			targetURL = joinOpenAIVideoURL(buildOpenAIVideosBaseURL(validatedURL), input.Path)
		}
	}

	method := strings.ToUpper(strings.TrimSpace(input.Method))
	if method == "" {
		method = http.MethodPost
	}
	var bodyReader io.Reader
	if len(input.Body) > 0 {
		bodyReader = bytes.NewReader(input.Body)
	}
	upstreamCtx, releaseUpstreamCtx := detachUpstreamContext(ctx)
	request, err := http.NewRequestWithContext(upstreamCtx, method, targetURL, bodyReader)
	releaseUpstreamCtx()
	if err != nil {
		return nil, err
	}
	request = request.WithContext(WithHTTPUpstreamProfile(request.Context(), HTTPUpstreamProfileOpenAI))
	if c != nil && c.Request != nil {
		for key, values := range c.Request.Header {
			if !openAIVideoAllowedHeaders[strings.ToLower(strings.TrimSpace(key))] {
				continue
			}
			for _, value := range values {
				request.Header.Add(key, value)
			}
		}
	}
	request.Header.Del("Authorization")
	request.Header.Del("X-Api-Key")
	request.Header.Set("Authorization", "Bearer "+token)
	if idempotencyKey := strings.TrimSpace(input.LocalRequestID); idempotencyKey != "" {
		request.Header.Set("Idempotency-Key", idempotencyKey)
	}
	if method == http.MethodPost && request.Header.Get("Content-Type") == "" {
		request.Header.Set("Content-Type", "application/json")
	}
	account.applyOpenAIHeaderOverrides(request.Header)
	s.applyOpenAIOutboundIdentity(ctx, account, request.Header, false)
	return request, nil
}

func (s *OpenAIGatewayService) ForwardVideo(ctx context.Context, c *gin.Context, account *Account, input OpenAIVideoForwardInput) (*OpenAIForwardResult, error) {
	startTime := time.Now()
	if account == nil {
		return nil, errors.New("account is required")
	}
	if !account.IsOpenAIApiKey() {
		writeOpenAIVideoError(c, http.StatusBadGateway, "upstream_error", "Video generation requires an OpenAI-compatible API key account")
		return nil, errors.New("video generation requires an OpenAI-compatible API key account")
	}

	input.Path = normalizeOpenAIVideoForwardPath(input.Path)
	if input.UpstreamModel == "" {
		input.UpstreamModel = input.Model
	}
	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return nil, err
	}
	request, err := s.buildOpenAIVideoUpstreamRequest(ctx, c, account, input, token)
	if err != nil {
		return nil, err
	}

	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	upstreamStart := time.Now()
	response, err := s.httpUpstream.Do(request, proxyURL, account.ID, account.Concurrency)
	SetOpsLatencyMs(c, OpsUpstreamLatencyMsKey, time.Since(upstreamStart).Milliseconds())
	if err != nil {
		safeErr := sanitizeUpstreamErrorMessage(err.Error())
		setOpsUpstreamError(c, 0, safeErr, "")
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
			Platform: account.Platform, AccountID: account.ID, AccountName: account.Name,
			Kind: "request_error", Message: safeErr,
		})
		writeOpenAIVideoError(c, http.StatusBadGateway, "upstream_error", "Upstream request failed")
		return nil, fmt.Errorf("upstream request failed: %s", safeErr)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode >= http.StatusBadRequest {
		body := s.readUpstreamErrorBody(response)
		_ = response.Body.Close()
		response.Body = io.NopCloser(bytes.NewReader(body))
		upstreamMessage := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(body)))
		if s.shouldFailoverOpenAIUpstreamResponse(response.StatusCode, upstreamMessage, body) {
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
				Platform: account.Platform, AccountID: account.ID, AccountName: account.Name,
				UpstreamStatusCode: response.StatusCode, UpstreamRequestID: response.Header.Get("x-request-id"),
				Kind: "failover", Message: upstreamMessage,
			})
			s.handleFailoverSideEffects(ctx, response, account, body, input.UpstreamModel)
			return nil, &UpstreamFailoverError{
				StatusCode: response.StatusCode, ResponseBody: body,
				RetryableOnSameAccount: account.IsPoolMode() && account.IsPoolModeRetryableStatus(response.StatusCode),
			}
		}
		return s.handleErrorResponse(ctx, response, c, account, input.Body, input.Model)
	}

	result := &OpenAIForwardResult{
		RequestID: response.Header.Get("x-request-id"), Usage: OpenAIUsage{},
		Model: input.Model, UpstreamModel: input.UpstreamModel,
		UpstreamEndpoint: input.Path, ResponseHeaders: response.Header.Clone(),
		StatusCode: response.StatusCode, Duration: time.Since(startTime),
	}
	if input.DeferResponseWrite {
		body, readErr := s.readOpenAIVideoJSONResponse(response.Body)
		if readErr != nil {
			return nil, readErr
		}
		result.ResponseBody = body
		return result, nil
	}
	if c != nil && c.Writer != nil {
		responseheaders.WriteFilteredHeaders(c.Writer.Header(), response.Header, s.responseHeaderFilter)
		c.Status(response.StatusCode)
		if _, err := io.Copy(c.Writer, response.Body); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (s *OpenAIGatewayService) WriteOpenAIVideoForwardResponse(c *gin.Context, result *OpenAIForwardResult) error {
	if c == nil || c.Writer == nil || result == nil {
		return errors.New("openai video response is incomplete")
	}
	responseheaders.WriteFilteredHeaders(c.Writer.Header(), result.ResponseHeaders, s.responseHeaderFilter)
	status := result.StatusCode
	if status <= 0 {
		status = http.StatusOK
	}
	c.Status(status)
	_, err := c.Writer.Write(result.ResponseBody)
	return err
}

func (s *OpenAIGatewayService) readOpenAIVideoJSONResponse(body io.Reader) ([]byte, error) {
	if body == nil {
		return nil, errors.New("upstream video response body is empty")
	}
	maxBytes := int64(0)
	if s != nil && s.cfg != nil {
		maxBytes = s.cfg.VideoTask.MaxResponseBytes
	}
	if maxBytes <= 0 {
		return nil, errors.New("video_task.max_response_bytes is not configured")
	}
	limited := io.LimitReader(body, maxBytes+1)
	payload, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(payload)) > maxBytes {
		return nil, errors.New("upstream video response exceeds configured limit")
	}
	return payload, nil
}

type OpenAIVideoPollResult struct {
	ProviderStatus string
	ErrorCode      string
	ErrorMessage   string
	Body           []byte
	StatusCode     int
}

func (s *OpenAIGatewayService) PollOpenAIVideoTask(ctx context.Context, task *OpenAIVideoTask, account *Account) (*OpenAIVideoPollResult, error) {
	if task == nil || task.TaskID == nil || strings.TrimSpace(*task.TaskID) == "" {
		return nil, ErrOpenAIVideoTaskIDMissing
	}
	if account == nil || account.ID != task.AccountID {
		return nil, ErrOpenAIVideoTaskConflict
	}
	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return nil, err
	}
	path := "/v1/videos/" + url.PathEscape(strings.TrimSpace(*task.TaskID))
	request, err := s.buildOpenAIVideoUpstreamRequest(ctx, nil, account, OpenAIVideoForwardInput{
		Method: http.MethodGet, Path: path, Model: task.RequestedModel, UpstreamModel: task.UpstreamModel,
	}, token)
	if err != nil {
		return nil, err
	}
	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	response, err := s.httpUpstream.Do(request, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()
	body, err := s.readOpenAIVideoJSONResponse(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= http.StatusBadRequest {
		message := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(body)))
		return nil, fmt.Errorf("video status upstream returned %d: %s", response.StatusCode, message)
	}
	_, providerStatus := parseOpenAIVideoTaskIdentity(body)
	if providerStatus == "" {
		return nil, errors.New("upstream video status response is missing status")
	}
	errorCode, errorMessage := parseOpenAIVideoProviderError(body)
	return &OpenAIVideoPollResult{
		ProviderStatus: providerStatus,
		ErrorCode:      errorCode,
		ErrorMessage:   errorMessage,
		Body:           body,
		StatusCode:     response.StatusCode,
	}, nil
}

func parseOpenAIVideoProviderError(body []byte) (string, string) {
	var code string
	for _, path := range []string{"error.code", "data.error.code", "code", "data.code"} {
		if value := strings.TrimSpace(gjson.GetBytes(body, path).String()); value != "" {
			code = sanitizeUpstreamErrorMessage(value)
			if len(code) > 128 {
				code = code[:128]
			}
			break
		}
	}
	var message string
	for _, path := range []string{"error.message", "data.error.message", "fail_reason", "data.fail_reason", "message", "data.message"} {
		if value := strings.TrimSpace(gjson.GetBytes(body, path).String()); value != "" {
			message = sanitizeUpstreamErrorMessage(value)
			break
		}
	}
	return code, truncateOpenAIVideoError(message)
}

func writeOpenAIVideoError(c *gin.Context, status int, errorType, message string) {
	if c == nil || c.Writer == nil || c.Writer.Written() {
		return
	}
	c.JSON(status, gin.H{"error": gin.H{"type": errorType, "message": message}})
}

func buildOpenAIVideosBaseURL(base string) string {
	normalized := strings.TrimRight(strings.TrimSpace(base), "/")
	if strings.HasSuffix(normalized, "/videos") {
		return normalized
	}
	if strings.HasSuffix(normalized, "/v1") {
		return normalized + "/videos"
	}
	return normalized + "/v1/videos"
}

func joinOpenAIVideoURL(baseURL, path string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	path = normalizeOpenAIVideoForwardPath(path)
	if path == "/v1/videos" {
		return base
	}
	return base + strings.TrimPrefix(path, "/v1/videos")
}

// normalizeOpenAIVideoForwardPath maps the legacy xAI/Canvas generations
// surface onto NewAPI's canonical OpenAI-compatible video task surface.
// Provider-native Grok edits/extensions never enter this service.
func normalizeOpenAIVideoForwardPath(path string) string {
	path = strings.TrimSpace(strings.SplitN(path, "?", 2)[0])
	if path == "" {
		return "/v1/videos"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	videoIndex := -1
	searchFrom := 0
	for searchFrom < len(path) {
		index := strings.Index(path[searchFrom:], "/videos")
		if index < 0 {
			break
		}
		index += searchFrom
		after := index + len("/videos")
		if after == len(path) || path[after] == '/' {
			videoIndex = index
			break
		}
		searchFrom = after
	}
	if videoIndex < 0 {
		return "/v1/videos"
	}

	suffix := path[videoIndex+len("/videos"):]
	if suffix == "" || suffix == "/" || suffix == "/generations" || suffix == "/generations/" {
		return "/v1/videos"
	}
	if strings.HasPrefix(suffix, "/generations/") {
		suffix = strings.TrimPrefix(suffix, "/generations")
	}
	return "/v1/videos" + suffix
}
