package service

import (
	"maps"

	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
)

const accountExtraImageEditsMultipart = "image_edits_multipart"

// OpenAIImageEditsMultipartEnabled is an upstream account capability shared by
// all its image models and API addresses. OAuth uses its own image protocol.
func (a *Account) OpenAIImageEditsMultipartEnabled() bool {
	if a == nil || a.Platform != PlatformOpenAI || a.Type != AccountTypeAPIKey {
		return false
	}
	if enabled, ok := a.Extra[accountExtraImageEditsMultipart].(bool); ok {
		return enabled
	}
	return openAIImageEditsMultipartDefault
}

func validateImageEditsMultipartExtra(extra map[string]any) error {
	if value, exists := extra[accountExtraImageEditsMultipart]; exists {
		if _, ok := value.(bool); !ok {
			return infraerrors.BadRequest("IMAGE_EDITS_MULTIPART_INVALID", "image_edits_multipart must be a boolean")
		}
	}
	return nil
}

// Omitted fields in an older account editor must not erase an enabled protocol.
func normalizeImageEditsMultipartExtra(platform, accountType string, extra, current map[string]any) (map[string]any, error) {
	if err := validateImageEditsMultipartExtra(extra); err != nil {
		return nil, err
	}
	if platform != PlatformOpenAI || accountType != AccountTypeAPIKey {
		return extra, nil
	}
	normalized := maps.Clone(extra)
	if normalized == nil {
		normalized = make(map[string]any)
	}
	if _, provided := normalized[accountExtraImageEditsMultipart]; !provided {
		normalized[accountExtraImageEditsMultipart] = openAIImageEditsMultipartDefault
		if enabled, ok := current[accountExtraImageEditsMultipart].(bool); ok {
			normalized[accountExtraImageEditsMultipart] = enabled
		}
	}
	return normalized, nil
}
