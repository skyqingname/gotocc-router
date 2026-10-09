package service

import (
	"strings"

	"github.com/tidwall/gjson"
)

// openAIStreamErrorEventIsProviderInternal reports a provider-side internal
// failure carried by a bare stream error frame, such as the Codex backend's
// "response protection is unavailable". Before any semantic output another
// account can serve the same turn, as with an equivalent response.failed event.
func openAIStreamErrorEventIsProviderInternal(payload []byte) bool {
	for _, path := range []string{"error.type", "response.error.type"} {
		switch strings.ToLower(strings.TrimSpace(gjson.GetBytes(payload, path).String())) {
		case "internal_error", "server_error":
			return true
		}
	}
	return false
}
