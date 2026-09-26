package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/tidwall/sjson"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/videoprotocol"
	"github.com/tidwall/gjson"
)

func validateGroupVideoModels(platform string, models videoprotocol.Models) error {
	if len(models) > 0 && platform != PlatformVideo {
		return infraerrors.BadRequest("INVALID_VIDEO_PLATFORM", "视频协议只能配置在 Video 分组")
	}
	for name, config := range models {
		if strings.TrimSpace(name) == "" {
			return infraerrors.BadRequest("INVALID_VIDEO_MODEL", "模型名不能为空")
		}
		if err := config.Validate(); err != nil {
			return infraerrors.BadRequest("INVALID_VIDEO_MODEL", name+": "+err.Error())
		}
	}
	return nil
}

func (s *GatewayService) VideoModelIDs(ctx context.Context, groupID int64) ([]string, error) {
	ids := []string{}
	group, err := s.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	models := group.VideoModels
	accounts, err := s.accountRepo.ListSchedulableByGroupIDAndPlatforms(ctx, groupID, []string{PlatformVideo, PlatformOpenAI})
	if err != nil {
		return nil, err
	}
	for model, config := range models {
		if !config.Enabled {
			continue
		}
		for i := range accounts {
			account := &accounts[i]
			if account.IsVideoAPIKey() && account.IsSchedulableForModelWithContext(ctx, config.UpstreamModel) && gatewayAccountSupportsModel(ctx, account, config.UpstreamModel) {
				ids = append(ids, model)
				break
			}
		}
	}
	sort.Strings(ids)
	return ids, nil
}

func (s *OpenAIGatewayService) listVideoAccounts(ctx context.Context, groupID *int64) ([]Account, error) {
	platforms := []string{PlatformVideo, PlatformOpenAI}
	if groupID == nil {
		return s.accountRepo.ListSchedulableUngroupedByPlatforms(ctx, platforms)
	}
	return s.accountRepo.ListSchedulableByGroupIDAndPlatforms(ctx, *groupID, platforms)
}

func (s *OpenAIGatewayService) SelectVideoAccount(ctx context.Context, groupID *int64, sessionHash, model, platform string) (*AccountSelectionResult, error) {
	var accounts []Account
	var err error
	if groupID == nil {
		accounts, err = s.accountRepo.ListSchedulableUngroupedByPlatform(ctx, PlatformOpenAI)
	} else {
		accounts, err = s.accountRepo.ListSchedulableByGroupID(ctx, *groupID)
	}
	if err != nil {
		return nil, err
	}
	excluded := make(map[int64]struct{})
	for i := range accounts {
		if !accounts[i].IsVideoAPIKey() {
			excluded[accounts[i].ID] = struct{}{}
		}
	}
	selected, _, err := s.SelectAccountWithSchedulerForCapability(ctx, groupID, "", sessionHash, model, excluded, OpenAIUpstreamTransportHTTPSSE, "", false, false, false, platform)
	return selected, err
}

func (s *OpenAIGatewayService) ResolveVideoModel(ctx context.Context, key *APIKey, model string) (*videoprotocol.Config, error) {
	if key.Group == nil || key.Group.Platform != PlatformVideo {
		return nil, nil
	}
	if !s.OpenAIVideoTaskLifecycleEnabled() {
		return nil, fmt.Errorf("Video platform requires the video task worker")
	}
	config, exists := key.Group.VideoModels[model]
	if !exists || !config.Enabled {
		return nil, fmt.Errorf("video model %s is not configured or enabled in this group", model)
	}
	if err := config.FreezeCatalog(); err != nil {
		return nil, err
	}
	return &config, nil
}

func PrepareVideoModelRequest(config *videoprotocol.Config, body []byte, contentType string) ([]byte, string, error) {
	parameters, err := OpenAIVideoRequestParameters(body, contentType)
	if err != nil {
		return nil, "", err
	}
	if config.Protocol == "yingce" {
		parameters, err = normalizeYingceVideoParameters(parameters)
		if err != nil {
			return nil, "", err
		}
	}
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, "", err
	}
	if mediaType == "multipart/form-data" {
		for _, parameter := range config.Parameters {
			value := gjson.GetBytes(parameters, parameter.Name)
			if value.Type != gjson.String {
				continue
			}
			var typed any
			switch parameter.Type {
			case "integer":
				typed, err = strconv.ParseInt(value.String(), 10, 64)
			case "number":
				typed, err = strconv.ParseFloat(value.String(), 64)
			case "boolean":
				typed, err = strconv.ParseBool(value.String())
			case "array", "object":
				err = json.Unmarshal([]byte(value.String()), &typed)
			default:
				continue
			}
			if err != nil {
				return nil, "", fmt.Errorf("invalid video form parameter %s", parameter.Name)
			}
			parameters, err = sjson.SetBytes(parameters, parameter.Name, typed)
			if err != nil {
				return nil, "", err
			}
		}
	}
	prepared, err := config.Prepare(parameters)
	if err != nil {
		return nil, "", err
	}
	if config.Protocol == "yingce" {
		if err := prepareYingceGeneration(config, prepared, body, contentType); err != nil {
			return nil, "", err
		}
		return prepared, "application/json", nil
	}
	if mediaType == "application/json" {
		return prepared, contentType, nil
	}
	if config.Protocol == "custom_json" {
		return nil, "", fmt.Errorf("this video protocol accepts JSON; supply media as URLs")
	}
	var output bytes.Buffer
	writer := multipart.NewWriter(&output)
	reader := multipart.NewReader(bytes.NewReader(body), params["boundary"])
	present := map[string]bool{}
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, "", err
		}
		present[part.FormName()] = true
		target, err := writer.CreatePart(part.Header)
		if err != nil {
			return nil, "", err
		}
		if _, err = io.Copy(target, part); err != nil {
			return nil, "", err
		}
		part.Close()
	}
	for field := range config.Defaults {
		if present[field] || !gjson.GetBytes(prepared, field).Exists() {
			continue
		}
		if err := writer.WriteField(field, gjson.GetBytes(prepared, field).String()); err != nil {
			return nil, "", err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, "", err
	}
	return output.Bytes(), writer.FormDataContentType(), nil
}

func (s *OpenAIGatewayService) videoProviderTarget(ctx context.Context, account *Account, input OpenAIVideoForwardInput, token string) (string, []byte, bool, error) {
	config := input.ProviderConfig
	path := config.CreatePath
	content := strings.HasSuffix(input.Path, "/content")
	if input.Method == http.MethodGet {
		path = config.StatusPath
	}
	if content {
		path = config.ContentPath
	}
	body := input.Body
	var err error
	if input.Method == http.MethodPost {
		body, err = config.MapRequest(body)
		if err != nil {
			return "", nil, false, err
		}
	}
	if content && path == "" {
		target, err := s.videoProviderResultURL(ctx, account, input, token)
		return target, nil, true, err
	}
	path = strings.ReplaceAll(path, "{task_id}", url.PathEscape(input.TaskID))
	path = strings.ReplaceAll(path, "{model}", url.PathEscape(config.UpstreamModel))
	base, err := s.validateUpstreamBaseURL(account.GetOpenAIBaseURL())
	if err != nil {
		return "", nil, false, err
	}
	// Paths are explicitly relative to the configured API origin, including /v1.
	parsed, err := url.Parse(base)
	if err != nil {
		return "", nil, false, err
	}
	relative, err := url.Parse(path)
	if err != nil {
		return "", nil, false, err
	}
	return parsed.ResolveReference(relative).String(), body, false, nil
}

func (s *OpenAIGatewayService) videoProviderResultURL(ctx context.Context, account *Account, input OpenAIVideoForwardInput, token string) (string, error) {
	request, err := s.buildOpenAIVideoUpstreamRequest(ctx, nil, account, OpenAIVideoForwardInput{
		Method: http.MethodGet, Path: "/v1/videos/" + url.PathEscape(input.TaskID), TaskID: input.TaskID,
		ProviderConfig: input.ProviderConfig, Model: input.Model, UpstreamModel: input.UpstreamModel,
	}, token)
	if err != nil {
		return "", err
	}
	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	response, err := s.httpUpstream.Do(request, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("video result lookup returned HTTP %d", response.StatusCode)
	}
	body, err := s.readOpenAIVideoJSONResponse(response.Body)
	if err != nil {
		return "", err
	}
	target := gjson.GetBytes(body, input.ProviderConfig.VideoURLField).String()
	if target == "" {
		return "", fmt.Errorf("video result URL is not ready")
	}
	// Result URLs may belong to provider storage; account credentials stay on
	// the API origin and are never attached to the storage request.
	return s.validateUpstreamBaseURL(target)
}

// Only the local ID is exposed for declarative providers whose task names may contain paths.
func SetVideoPublicTaskID(body []byte, id string) ([]byte, error) {
	return sjson.SetBytes(body, "id", id)
}
