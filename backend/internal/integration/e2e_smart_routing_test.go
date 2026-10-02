//go:build e2e

package integration

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

// Set E2E_ROUTING_API_KEY only for an isolated fixture with gpt-4.1 and
// claude-sonnet-4-20250514 configured. One key must work across both protocols.
func TestSmartRoutingProtocolFlow(t *testing.T) {
	key := strings.TrimSpace(os.Getenv("E2E_ROUTING_API_KEY"))
	if key == "" {
		t.Skip("E2E_ROUTING_API_KEY is required for the routing fixture")
	}
	requireUserFlowInstance(t)
	for _, tc := range []struct {
		name, path, model string
		stream            bool
	}{
		{"openai_chat", "/v1/chat/completions", "gpt-4.1", false},
		{"anthropic_messages", "/v1/messages", "claude-sonnet-4-20250514", false},
		{"openai_stream", "/v1/chat/completions", "gpt-4.1", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body, err := json.Marshal(map[string]any{"model": tc.model, "messages": []map[string]string{{"role": "user", "content": "local routing verification"}}, "max_tokens": 32, "stream": tc.stream})
			if err != nil {
				t.Fatal(err)
			}
			resp, err := doRequest(t, http.MethodPost, tc.path, body, key)
			if err != nil {
				t.Fatalf("gateway request: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("gateway returned %d, want 200", resp.StatusCode)
			}
			data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
			if err != nil {
				t.Fatal(err)
			}
			if tc.stream {
				if !strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") || !strings.Contains(string(data), "[DONE]") {
					t.Fatal("expected complete SSE stream")
				}
				return
			}
			var response map[string]json.RawMessage
			if err := json.Unmarshal(data, &response); err != nil {
				t.Fatalf("gateway returned invalid JSON: %v", err)
			}
			var model string
			if err := json.Unmarshal(response["model"], &model); err != nil || model != tc.model {
				t.Fatalf("gateway returned unexpected model %q", model)
			}
			if len(response["usage"]) == 0 {
				t.Fatal("gateway response is missing usage")
			}
			if tc.path == "/v1/messages" && len(response["content"]) == 0 {
				t.Fatal("Anthropic response is missing content")
			}
			if tc.path == "/v1/chat/completions" && len(response["choices"]) == 0 {
				t.Fatal("OpenAI response is missing choices")
			}
		})
	}
}
