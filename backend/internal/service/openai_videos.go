package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/util/responseheaders"
	"github.com/gin-gonic/gin"
)

const openAIPlatformVideosURL = "https://api.openai.com/v1/videos"

var openAIVideoAllowedHeaders = map[string]bool{
	"accept":          true,
	"accept-language": true,
	"content-type":    true,
}

// OpenAIVideoForwardInput describes the OpenAI-compatible video task surface.
type OpenAIVideoForwardInput struct {
	Method        string
	Path          string
	Body          []byte
	Model         string
	UpstreamModel string
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

	input.Path = sanitizeOpenAIVideoForwardPath(input.Path)
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

	if c != nil && c.Writer != nil {
		responseheaders.WriteFilteredHeaders(c.Writer.Header(), response.Header, s.responseHeaderFilter)
		c.Status(response.StatusCode)
		if _, err := io.Copy(c.Writer, response.Body); err != nil {
			return nil, err
		}
	}
	return &OpenAIForwardResult{
		RequestID: response.Header.Get("x-request-id"), Usage: OpenAIUsage{},
		Model: input.Model, UpstreamModel: input.UpstreamModel,
		UpstreamEndpoint: input.Path, ResponseHeaders: response.Header.Clone(), Duration: time.Since(startTime),
	}, nil
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
	path = strings.TrimSpace(path)
	if path == "" || path == "/v1/videos" || path == "/videos" {
		return base
	}
	if index := strings.LastIndex(path, "/videos"); index >= 0 {
		if suffix := strings.TrimLeft(path[index+len("/videos"):], "/"); suffix != "" {
			return base + "/" + suffix
		}
		return base
	}
	return base + "/" + strings.TrimLeft(path, "/")
}

func sanitizeOpenAIVideoForwardPath(path string) string {
	path = strings.TrimSpace(strings.SplitN(path, "?", 2)[0])
	if path == "" || path == "/v1/videos" || path == "/videos" {
		return "/v1/videos"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if !strings.Contains(path, "/videos") {
		return "/v1/videos"
	}
	return path
}
