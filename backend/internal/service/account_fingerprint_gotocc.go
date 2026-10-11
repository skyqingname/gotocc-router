package service

import (
	"context"
	"encoding/json"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type accountFingerprintContextKey struct{}

var accountFingerprintNonTextModel = regexp.MustCompile(accountFingerprintNonTextModelPattern)

func accountFingerprintPrompt(ctx context.Context) string {
	prompt, _ := ctx.Value(accountFingerprintContextKey{}).(string)
	return prompt
}

func (s *AccountTestService) TestAccountFingerprint(c *gin.Context, accountID int64, modelID, prompt string) error {
	modelID, prompt = strings.TrimSpace(modelID), strings.TrimSpace(prompt)
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(accountFingerprintTimeoutSeconds)*time.Second)
	defer cancel()
	c.Request = c.Request.WithContext(context.WithValue(ctx, accountFingerprintContextKey{}, prompt))
	return s.TestAccountConnection(c, accountID, modelID, prompt, "fingerprint")
}

func accountFingerprintUnsupported(account *Account, modelID string) bool {
	return slices.Contains(accountFingerprintUnsupportedPlatforms, account.Platform) ||
		accountFingerprintNonTextModel.MatchString(modelID) ||
		accountFingerprintNonTextModel.MatchString(account.GetMappedModel(modelID)) ||
		account.IsSyntheticUITest()
}

func createAccountFingerprintMessagesPayload(ctx context.Context, modelID string) (map[string]any, error) {
	payload, err := createTestPayload(modelID)
	if err != nil {
		return nil, err
	}
	applyAccountFingerprintMessages(ctx, payload)
	return payload, nil
}

// Keep the account's existing endpoint, authentication, identity and required
// provider instructions. Only the probe's user input and output budget change.
func applyAccountFingerprintMessages(ctx context.Context, payload map[string]any) {
	if prompt := accountFingerprintPrompt(ctx); prompt != "" {
		payload["messages"] = []map[string]any{{"role": "user", "content": prompt}}
		payload["max_tokens"] = accountFingerprintMaxOutputTokens
	}
}

func applyAccountFingerprintResponses(ctx context.Context, payload map[string]any, codex bool) {
	if prompt := accountFingerprintPrompt(ctx); prompt != "" {
		payload["input"] = []map[string]any{{"role": "user", "content": []map[string]any{{"type": "input_text", "text": prompt}}}}
		if !codex {
			payload["max_output_tokens"] = accountFingerprintMaxOutputTokens
		}
	}
}

func applyAccountFingerprintGemini(ctx context.Context, payload map[string]any) {
	if prompt := accountFingerprintPrompt(ctx); prompt != "" {
		payload["contents"] = []map[string]any{{"role": "user", "parts": []map[string]any{{"text": prompt}}}}
		payload["generationConfig"] = map[string]any{"maxOutputTokens": accountFingerprintMaxOutputTokens}
	}
}

// These byte payloads are produced by the existing provider builders. Apply
// the challenge before provider wrapping/signing, never after authentication.
func accountFingerprintJSON(ctx context.Context, body []byte, protocol string) []byte {
	if accountFingerprintPrompt(ctx) == "" {
		return body
	}
	var payload map[string]any
	_ = json.Unmarshal(body, &payload)
	switch protocol {
	case "responses":
		applyAccountFingerprintResponses(ctx, payload, false)
	case "gemini":
		applyAccountFingerprintGemini(ctx, payload)
	case "antigravity":
		applyAccountFingerprintGemini(ctx, payload["request"].(map[string]any))
	}
	result, _ := json.Marshal(payload)
	return result
}
