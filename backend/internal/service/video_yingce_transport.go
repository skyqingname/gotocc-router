package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	yp "github.com/LuckyKuang/sub2api-plus/internal/pkg/yingceprotocol"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
)

func (s *OpenAIGatewayService) buildYingceVideoUpstreamRequest(ctx context.Context, c *gin.Context, account *Account, input OpenAIVideoForwardInput, token string) (*http.Request, error) {
	config := input.ProviderConfig
	adapter, err := config.Adapter()
	if err != nil {
		return nil, err
	}
	poll := yp.PollContext{BaseURL: account.GetOpenAIBaseURL(), Model: config.UpstreamModel, TaskID: input.TaskID}
	if config.PollRequest != nil {
		poll.Request = *config.PollRequest
	}
	var spec yp.RequestSpec
	publicResult := false
	absoluteTarget := ""
	switch {
	case strings.HasSuffix(input.Path, "/content"):
		if capability, ok := adapter.(yp.ResultCapability); ok && capability.ResultAvailable() {
			spec, err = adapter.(yp.ResultAdapter).BuildResult(ctx, poll)
		} else {
			statusInput := input
			statusInput.Path = "/v1/videos/" + url.PathEscape(input.TaskID)
			statusInput.Method = http.MethodGet
			statusRequest, e := s.buildYingceVideoUpstreamRequest(ctx, nil, account, statusInput, token)
			if e != nil {
				return nil, e
			}
			proxy := ""
			if account.Proxy != nil {
				proxy = account.Proxy.URL()
			}
			response, e := s.httpUpstream.Do(statusRequest, proxy, account.ID, account.Concurrency)
			if e != nil {
				return nil, e
			}
			defer response.Body.Close()
			if response.StatusCode != http.StatusOK {
				return nil, fmt.Errorf("video result lookup returned HTTP %d", response.StatusCode)
			}
			raw, e := s.readOpenAIVideoJSONResponse(response.Body)
			if e != nil {
				return nil, e
			}
			body, e := config.NormalizeResponse(raw, input.TaskID, false)
			if e != nil {
				return nil, e
			}
			absoluteTarget, e = s.validateUpstreamBaseURL(gjson.GetBytes(body, "url").String())
			if e != nil {
				return nil, e
			}
			apiBase, _ := url.Parse(account.GetOpenAIBaseURL())
			mediaURL, _ := url.Parse(absoluteTarget)
			publicResult = apiBase.Host != mediaURL.Host
			spec = yp.RequestSpec{Method: http.MethodGet, Path: "/", Auth: pollAuth(config.ProviderManifest)}
		}
	case input.Method == http.MethodPost:
		if config.CreateRequest == nil {
			return nil, fmt.Errorf("video create request was not prepared")
		}
		spec, err = adapter.BuildCreate(ctx, yp.RequestContext{BaseURL: account.GetOpenAIBaseURL(), Request: *config.CreateRequest})
	default:
		spec, err = adapter.BuildPoll(ctx, poll)
	}
	if err != nil {
		return nil, err
	}
	if err = spec.Validate(); err != nil {
		return nil, err
	}
	target := absoluteTarget
	if target == "" {
		base, e := s.validateUpstreamBaseURL(account.GetOpenAIBaseURL())
		if e != nil {
			return nil, e
		}
		target, err = yingceProtocolURL(base, spec)
		if err != nil {
			return nil, err
		}
	}
	body, contentType, err := s.yingceRequestBody(ctx, account, spec)
	if err != nil {
		return nil, err
	}
	upstreamCtx, release := detachUpstreamContext(ctx)
	request, err := http.NewRequestWithContext(upstreamCtx, spec.Method, target, body)
	release()
	if err != nil {
		return nil, err
	}
	request = request.WithContext(WithHTTPUpstreamProfile(request.Context(), HTTPUpstreamProfileOpenAI))
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	if c != nil {
		for _, name := range []string{"Range", "Accept"} {
			if v := c.GetHeader(name); v != "" {
				request.Header.Set(name, v)
			}
		}
	}
	if !publicResult {
		for name, value := range spec.Headers {
			request.Header.Set(name, value)
		}
		for name, value := range config.Headers {
			request.Header.Set(name, value)
		}
		if input.LocalRequestID != "" {
			request.Header.Set("Idempotency-Key", input.LocalRequestID)
		}
		account.applyOpenAIHeaderOverrides(request.Header)
	}
	s.applyOpenAIOutboundIdentity(ctx, account, request.Header, false)
	request = prepareAccountOutboundRequest(request, account)
	if publicResult {
		request.Header.Del("Authorization")
		request.Header.Del("X-Api-Key")
		request.Header.Del("X-Goog-Api-Key")
	} else if err = applyYingceProtocolAuth(request, account, token, spec.Auth); err != nil {
		return nil, err
	}
	return request, nil
}
func pollAuth(raw json.RawMessage) yp.ManifestAuth {
	var m yp.Manifest
	_ = json.Unmarshal(raw, &m)
	return m.Contributes.Providers[0].Auth
}
func yingceProtocolURL(base string, spec yp.RequestSpec) (string, error) {
	origin, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	relative, err := url.Parse(spec.Path)
	if err != nil {
		return "", err
	}
	target := origin.ResolveReference(relative)
	if !spec.OriginPath {
		target, err = url.Parse(yingceAPIURLWithDefaultPrefix(base, spec.Path, yingceURLDefaults.Prefix))
		if err != nil {
			return "", err
		}
	}
	query := target.Query()
	for name, values := range spec.Query {
		for _, v := range values {
			query.Add(name, v)
		}
	}
	target.RawQuery = query.Encode()
	return target.String(), nil
}
func (s *OpenAIGatewayService) yingceRequestBody(ctx context.Context, account *Account, spec yp.RequestSpec) (io.Reader, string, error) {
	if spec.Body == nil && len(spec.Files) == 0 {
		return nil, "", nil
	}
	switch spec.ContentType {
	case "", "application/json":
		data, err := json.Marshal(spec.Body)
		return bytes.NewReader(data), "application/json", err
	case "application/x-www-form-urlencoded":
		values := url.Values{}
		for key, value := range spec.Body.(map[string]any) {
			items, err := yingceFormValues(value)
			if err != nil {
				return nil, "", err
			}
			for _, item := range items {
				values.Add(key, item)
			}
		}
		return strings.NewReader(values.Encode()), spec.ContentType, nil
	case "multipart/form-data":
		var buffer bytes.Buffer
		writer := multipart.NewWriter(&buffer)
		if spec.Body != nil {
			for key, value := range spec.Body.(map[string]any) {
				items, err := yingceFormValues(value)
				if err != nil {
					return nil, "", err
				}
				for _, item := range items {
					if err := writer.WriteField(key, item); err != nil {
						return nil, "", err
					}
				}
			}
		}
		for _, file := range spec.Files {
			data, mimeType, err := s.yingceMediaBytes(ctx, account, file.Reference)
			if err != nil {
				return nil, "", err
			}
			if file.MIMEType != "" {
				mimeType = file.MIMEType
			}
			header := textproto.MIMEHeader{}
			header.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{"name": file.Name, "filename": yingceDefault(file.Filename, "reference")}))
			header.Set("Content-Type", yingceDefault(mimeType, "application/octet-stream"))
			part, err := writer.CreatePart(header)
			if err != nil {
				return nil, "", err
			}
			if _, err = part.Write(data); err != nil {
				return nil, "", err
			}
		}
		if err := writer.Close(); err != nil {
			return nil, "", err
		}
		return bytes.NewReader(buffer.Bytes()), writer.FormDataContentType(), nil
	default:
		return nil, "", fmt.Errorf("unsupported video request content type %s", spec.ContentType)
	}
}
func (s *OpenAIGatewayService) yingceMediaBytes(ctx context.Context, account *Account, reference yp.MediaReference) ([]byte, string, error) {
	if reference.DataURL != "" {
		header, encoded, ok := strings.Cut(reference.DataURL, ",")
		if !ok {
			return nil, "", fmt.Errorf("invalid media data URL")
		}
		kind := strings.TrimPrefix(strings.Split(header, ";")[0], "data:")
		if strings.HasSuffix(header, ";base64") {
			data, err := base64.StdEncoding.DecodeString(encoded)
			return data, kind, err
		}
		data, err := url.QueryUnescape(encoded)
		return []byte(data), kind, err
	}
	target, err := s.validateUpstreamBaseURL(reference.URL)
	if err != nil {
		return nil, "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, "", err
	}
	s.applyOpenAIOutboundIdentity(ctx, account, request.Header, false)
	proxy := ""
	if account.Proxy != nil {
		proxy = account.Proxy.URL()
	}
	response, err := s.httpUpstream.Do(request, proxy, account.ID, account.Concurrency)
	if err != nil {
		return nil, "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("video reference returned HTTP %d", response.StatusCode)
	}
	data, err := io.ReadAll(response.Body)
	return data, response.Header.Get("Content-Type"), err
}

func yingceFormValues(value any) ([]string, error) {
	switch v := value.(type) {
	case nil:
		return nil, nil
	case string:
		return []string{v}, nil
	case []any:
		var out []string
		for _, item := range v {
			values, err := yingceFormValues(item)
			if err != nil {
				return nil, err
			}
			out = append(out, values...)
		}
		return out, nil
	default:
		data, err := json.Marshal(v)
		return []string{string(data)}, err
	}
}
