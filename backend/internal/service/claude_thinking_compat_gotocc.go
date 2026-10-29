package service

import (
	"fmt"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/claude"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type claudeLegacyThinkingAdaptation struct {
	Type   string
	Effort string
}

// prepareClaude55MessagesRequest adapts native Messages parameters after the
// upstream model is resolved. The parsed request belongs to this account attempt;
// ReplaceBody keeps derived thinking/effort fields and raw ranges in sync.
func prepareClaude55MessagesRequest(parsed *ParsedRequest, model string) error {
	body, changed, err := normalizeClaudeOpus55LegacyThinking(parsed.Body.Bytes(), model)
	if err != nil {
		return fmt.Errorf("adapt Claude thinking: %w", err)
	}
	if changed {
		if err := parsed.ReplaceBody(body); err != nil {
			return fmt.Errorf("parse adapted Claude request: %w", err)
		}
	}
	return validateClaude55Request(parsed.Body.Bytes(), model)
}

// normalizeClaudeOpus55LegacyThinking only changes the two legacy modes selected
// in upstream-request-adaptations.json. Existing adaptive requests and other
// models remain byte-for-byte unchanged. Conversation blocks are never rewritten.
func normalizeClaudeOpus55LegacyThinking(body []byte, model string) ([]byte, bool, error) {
	if !claude.IsOpus55(model) {
		return body, false, nil
	}
	rule, ok := claudeOpus55LegacyThinking[gjson.GetBytes(body, "thinking.type").String()]
	if !ok {
		return body, false, nil
	}
	adapted, err := sjson.SetBytes(body, "thinking.type", rule.Type)
	if err != nil {
		return nil, false, err
	}
	adapted, err = sjson.DeleteBytes(adapted, "thinking.budget_tokens")
	if err != nil {
		return nil, false, err
	}
	// An empty configured effort preserves the caller's value or the model default.
	// Explicit disabled maps to low even if a contradictory higher effort was sent.
	if rule.Effort != "" {
		adapted, err = sjson.SetBytes(adapted, "output_config.effort", rule.Effort)
		if err != nil {
			return nil, false, err
		}
	}
	return adapted, true, nil
}
