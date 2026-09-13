package service

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LuckyKuang/sub2api-plus/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestBuildOpenAIVideosBaseURL(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		base string
		want string
	}{
		{base: "https://video.example.test", want: "https://video.example.test/v1/videos"},
		{base: "https://video.example.test/v1", want: "https://video.example.test/v1/videos"},
		{base: "https://video.example.test/api/v4", want: "https://video.example.test/api/v4/v1/videos"},
		{base: "https://video.example.test/v1/videos", want: "https://video.example.test/v1/videos"},
	} {
		require.Equal(t, test.want, buildOpenAIVideosBaseURL(test.base))
	}
}

func TestNormalizeOpenAIVideoForwardPath(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		path string
		want string
	}{
		{path: "", want: "/v1/videos"},
		{path: "/videos", want: "/v1/videos"},
		{path: "/v1/videos", want: "/v1/videos"},
		{path: "/videos/generations", want: "/v1/videos"},
		{path: "/v1/videos/generations", want: "/v1/videos"},
		{path: "/videos/generations/task-123", want: "/v1/videos/task-123"},
		{path: "/v1/videos/generations/task-123/content", want: "/v1/videos/task-123/content"},
		{path: "/v1/videos/task-123?model=grok-imagine-video", want: "/v1/videos/task-123"},
		{path: "/unsupported", want: "/v1/videos"},
	} {
		require.Equal(t, test.want, normalizeOpenAIVideoForwardPath(test.path), "path=%s", test.path)
	}
}

func TestJoinOpenAIVideoURLUsesCanonicalNewAPISurface(t *testing.T) {
	t.Parallel()

	base := "https://video.example.test/v1/videos"
	require.Equal(t, base, joinOpenAIVideoURL(base, "/v1/videos"))
	require.Equal(t, base+"/task-123", joinOpenAIVideoURL(base, "/v1/videos/task-123"))
	require.Equal(t, base+"/task-123/content", joinOpenAIVideoURL(base, "/v1/videos/task-123/content"))
	require.Equal(t, base, joinOpenAIVideoURL(base, "/v1/videos/generations"))
	require.Equal(t, base+"/task-123", joinOpenAIVideoURL(base, "/videos/generations/task-123"))
	require.Equal(t, base+"/task-123/content", joinOpenAIVideoURL(base, "/videos/generations/task-123/content"))
}

func TestBuildOpenAIVideoUpstreamRequestUsesAccountCredentialOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &OpenAIGatewayService{cfg: &config.Config{
		Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{
			Enabled:           false,
			AllowInsecureHTTP: true,
		}},
	}}
	account := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":  "upstream-secret",
			"base_url": "http://video.example.test/v1",
		},
	}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos/generations", bytes.NewReader([]byte(`{"model":"grok-imagine-video"}`)))
	c.Request.Header.Set("Authorization", "Bearer user-secret")
	c.Request.Header.Set("X-Api-Key", "user-secret")
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-Unsafe-Forward", "must-not-pass")

	request, err := service.buildOpenAIVideoUpstreamRequest(
		context.Background(),
		c,
		account,
		OpenAIVideoForwardInput{Method: http.MethodPost, Path: "/v1/videos/generations", Body: []byte(`{"model":"grok-imagine-video"}`)},
		"upstream-secret",
	)
	require.NoError(t, err)
	require.Equal(t, "http://video.example.test/v1/videos", request.URL.String())
	require.Equal(t, "Bearer upstream-secret", request.Header.Get("Authorization"))
	require.Empty(t, request.Header.Get("X-Api-Key"))
	require.Empty(t, request.Header.Get("X-Unsafe-Forward"))
	require.NotEmpty(t, request.Header.Get("User-Agent"))
}

func TestOpenAIVideoBillingUnitsDistinguishPerRequestAndPerSecond(t *testing.T) {
	t.Parallel()

	perRequestUnits, perRequestMode := openAIVideoBillingUnits(BillingModePerRequest, 2, 15)
	require.Equal(t, float64(2), perRequestUnits)
	require.Equal(t, BillingModePerRequest, perRequestMode)

	perSecondUnits, perSecondMode := openAIVideoBillingUnits(BillingModeVideo, 2, 15)
	require.Equal(t, float64(30), perSecondUnits)
	require.Equal(t, BillingModeVideo, perSecondMode)

	legacyImageUnits, legacyImageMode := openAIVideoBillingUnits(BillingModeImage, 2, 15)
	require.Equal(t, float64(2), legacyImageUnits)
	require.Equal(t, BillingModePerRequest, legacyImageMode)
}

func TestNormalizeOpenAIVideoTaskBillingMode(t *testing.T) {
	t.Parallel()

	mode, err := normalizeOpenAIVideoTaskBillingMode(string(BillingModePerRequest))
	require.NoError(t, err)
	require.Equal(t, string(BillingModePerRequest), mode)

	mode, err = normalizeOpenAIVideoTaskBillingMode(string(BillingModeVideo))
	require.NoError(t, err)
	require.Equal(t, string(BillingModeVideo), mode)

	_, err = normalizeOpenAIVideoTaskBillingMode(string(BillingModeToken))
	require.Error(t, err)
}
