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

func TestJoinOpenAIVideoURLPreservesTaskSurface(t *testing.T) {
	t.Parallel()

	base := "https://video.example.test/v1/videos"
	require.Equal(t, base, joinOpenAIVideoURL(base, "/v1/videos"))
	require.Equal(t, base+"/task-123", joinOpenAIVideoURL(base, "/v1/videos/task-123"))
	require.Equal(t, base+"/task-123/content", joinOpenAIVideoURL(base, "/v1/videos/task-123/content"))
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
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewReader([]byte(`{"model":"video-ds-2.0-fast"}`)))
	c.Request.Header.Set("Authorization", "Bearer user-secret")
	c.Request.Header.Set("X-Api-Key", "user-secret")
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-Unsafe-Forward", "must-not-pass")

	request, err := service.buildOpenAIVideoUpstreamRequest(
		context.Background(),
		c,
		account,
		OpenAIVideoForwardInput{Method: http.MethodPost, Path: "/v1/videos", Body: []byte(`{"model":"video-ds-2.0-fast"}`)},
		"upstream-secret",
	)
	require.NoError(t, err)
	require.Equal(t, "http://video.example.test/v1/videos", request.URL.String())
	require.Equal(t, "Bearer upstream-secret", request.Header.Get("Authorization"))
	require.Empty(t, request.Header.Get("X-Api-Key"))
	require.Empty(t, request.Header.Get("X-Unsafe-Forward"))
	require.NotEmpty(t, request.Header.Get("User-Agent"))
}
