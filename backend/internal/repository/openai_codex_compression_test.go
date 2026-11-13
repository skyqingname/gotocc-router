//go:build unit

package repository

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/stretchr/testify/require"
)

// Capture the production HTTPUpstream wire. Locally valid Zstd, including a
// 512 KiB window, is rejected for some bodies; every JSON byte must reach the
// upstream without request compression, including retries and boundary sizes.
func TestCodexHTTPIdentityBodyWire(t *testing.T) {
	type wireRequest struct {
		header http.Header
		body   []byte
		length int64
	}
	captured := make(chan wireRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		captured <- wireRequest{r.Header.Clone(), body, r.ContentLength}
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer server.Close()
	client := NewHTTPUpstream(nil)
	for _, tc := range []struct {
		name, kind, path, authMode string
	}{
		{"oauth", service.AccountTypeOAuth, "/backend-api/codex/responses", ""},
		{"setup token", service.AccountTypeSetupToken, "/backend-api/codex/responses", ""},
		{"api key", service.AccountTypeAPIKey, "/v1/responses", ""},
		{"compact", service.AccountTypeOAuth, "/backend-api/codex/responses/compact", ""},
		{"models", service.AccountTypeOAuth, "/backend-api/codex/models", ""},
		{"signed identity", service.AccountTypeOAuth, "/backend-api/codex/responses", "agentIdentity"},
		{"signed shadow", service.AccountTypeOAuth, "/backend-api/codex/responses", ""},
	} {
		for _, size := range []int{64, 131071, 131072, 131073, 140132, 262143, 262144, 262145, 524287, 524288, 524289, 1577051, 2497413} {
			text := strings.Repeat("a", size-len(`{"model":"gpt-6.1-sol","stream":true,"input":""}`))
			t.Run(tc.name+"/bytes="+strconv.Itoa(size), func(t *testing.T) {
				body := []byte(`{"model":"gpt-6.1-sol","stream":true,"input":"` + text + `"}`)
				req, err := http.NewRequest(http.MethodPost, server.URL+tc.path, bytes.NewReader(body))
				require.NoError(t, err)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer fixture")
				if strings.HasPrefix(tc.name, "signed") {
					req.Header.Set("Authorization", "AgentAssertion fixture")
				}
				req.Header.Set("User-Agent", "codex_cli_rs/0.163.0 (Ubuntu 24.04; x86_64) xterm-256color")
				req.Header.Set("Originator", "codex_cli_rs")
				req.Header.Set("Version", "0.163.0")
				account := &service.Account{ID: 91, Platform: service.PlatformOpenAI, Type: tc.kind, Credentials: map[string]any{"auth_mode": tc.authMode, "user_agent": "codex_cli_rs/0.163.0 (Ubuntu 24.04; x86_64) xterm-256color"}}
				service.ApplyAccountOutboundIdentity(req.Context(), account, req)
				for attempt := 0; attempt < 2; attempt++ {
					if attempt > 0 {
						req.Body, err = req.GetBody()
						require.NoError(t, err)
					}
					resp, err := client.Do(req, "", 91, 1)
					require.NoError(t, err)
					_, err = io.Copy(io.Discard, resp.Body)
					require.NoError(t, err)
					require.NoError(t, resp.Body.Close())
					wire := <-captured
					require.Empty(t, wire.header.Get("Content-Encoding"))
					require.Equal(t, body, wire.body, "every body byte must survive forwarding and retries")
					require.Equal(t, int64(len(wire.body)), wire.length)
					require.Equal(t, "codex_cli_rs", wire.header.Get("Originator"))
					require.Equal(t, "0.163.0", wire.header.Get("Version"))
				}
			})
		}
	}
}
