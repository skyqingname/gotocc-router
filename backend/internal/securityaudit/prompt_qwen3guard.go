package securityaudit

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
)

type ScannerDefinition struct {
	ID string `json:"id"`
	Label string `json:"label"`
	LabelZH string `json:"label_zh"`
	Description string `json:"description"`
}

var AllScannerIDs = []string{
	"violent", "non_violent_illegal_acts", "sexual_content_or_sexual_acts", "pii",
	"suicide_and_self_harm", "unethical_acts", "politically_sensitive_topics", "copyright_violation", "jailbreak",
}

var ScannerCatalog = map[string]ScannerDefinition{
	"violent": {ID: "violent", Label: "Violent", LabelZH: "暴力", Description: "Violence or threats of violence"},
	"non_violent_illegal_acts": {ID: "non_violent_illegal_acts", Label: "Non-violent Illegal Acts", LabelZH: "非暴力违法行为", Description: "Non-violent illegal activity"},
	"sexual_content_or_sexual_acts": {ID: "sexual_content_or_sexual_acts", Label: "Sexual Content or Sexual Acts", LabelZH: "性内容或性行为", Description: "Sexual content or sexual acts"},
	"pii": {ID: "pii", Label: "PII", LabelZH: "个人敏感信息", Description: "Personal identifying information"},
	"suicide_and_self_harm": {ID: "suicide_and_self_harm", Label: "Suicide & Self-Harm", LabelZH: "自杀与自残", Description: "Suicide or self-harm"},
	"unethical_acts": {ID: "unethical_acts", Label: "Unethical Acts", LabelZH: "不道德行为", Description: "Unethical behavior"},
	"politically_sensitive_topics": {ID: "politically_sensitive_topics", Label: "Politically Sensitive Topics", LabelZH: "政治敏感话题", Description: "Politically sensitive topics"},
	"copyright_violation": {ID: "copyright_violation", Label: "Copyright Violation", LabelZH: "版权侵权", Description: "Copyright infringement"},
	"jailbreak": {ID: "jailbreak", Label: "Jailbreak", LabelZH: "越狱攻击", Description: "Prompt injection or jailbreak attempt"},
}

var categoryAliases = map[string]string{
	"violent": "violent", "violence": "violent",
	"non violent illegal acts": "non_violent_illegal_acts", "non-violent illegal acts": "non_violent_illegal_acts",
	"sexual content or sexual acts": "sexual_content_or_sexual_acts", "sexual": "sexual_content_or_sexual_acts",
	"pii": "pii", "personal identifying information": "pii", "personal identifiable information": "pii",
	"suicide self harm": "suicide_and_self_harm", "suicide and self harm": "suicide_and_self_harm", "suicide & self-harm": "suicide_and_self_harm",
	"unethical acts": "unethical_acts", "unethical": "unethical_acts",
	"politically sensitive topics": "politically_sensitive_topics", "political": "politically_sensitive_topics",
	"copyright violation": "copyright_violation", "copyright": "copyright_violation",
	"jailbreak": "jailbreak", "prompt injection": "jailbreak",
}

type GuardError struct {
	Code string
	HTTPStatus int
	Retryable bool
	Timeout bool
	Cause error
}

func (e *GuardError) Error() string {
	if e == nil { return "<nil>" }
	return e.Code
}
func (e *GuardError) Unwrap() error { return e.Cause }

func NormalizeCategory(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.NewReplacer("_", " ", "&", " and ", "/", " ", "-", " ", "–", " ", "—", " ").Replace(normalized)
	normalized = strings.Join(strings.Fields(normalized), " ")
	if canonical, ok := categoryAliases[normalized]; ok { return canonical }
	return strings.ReplaceAll(normalized, " ", "_")
}

// ParseQwen3Guard decodes the historical result format. It is deliberately not
// called by the runtime scanner; saved historical format/decoder tests remain
// readable during the transition. No HTTP request uses the old model protocol.
func ParseQwen3Guard(content string, enabledScanners []string) (*NormalizedResult, error) {
	var safety, categoryLine string
	for _, line := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" { continue }
		lower := strings.ToLower(line)
		switch {
		case strings.HasPrefix(lower, "safety:"):
			if safety != "" { return nil, &GuardError{Code: ErrorCodeInvalidResponse} }
			safety = strings.TrimSpace(line[len("safety:"):])
		case strings.HasPrefix(lower, "categories:"):
			if categoryLine != "" { return nil, &GuardError{Code: ErrorCodeInvalidResponse} }
			categoryLine = strings.TrimSpace(line[len("categories:"):])
		}
	}
	switch strings.ToLower(safety) {
	case "safe": safety = "Safe"
	case "controversial": safety = "Controversial"
	case "unsafe": safety = "Unsafe"
	default: return nil, &GuardError{Code: ErrorCodeInvalidResponse}
	}
	if categoryLine == "" { return nil, &GuardError{Code: ErrorCodeInvalidResponse} }
	enabled := make(map[string]struct{}, len(enabledScanners))
	for _, scanner := range enabledScanners { enabled[NormalizeCategory(scanner)] = struct{}{} }
	known, unknown := map[string]struct{}{}, map[string]struct{}{}
	for _, raw := range strings.Split(categoryLine, ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" || strings.EqualFold(raw, "none") || strings.EqualFold(raw, "n/a") { continue }
		category := NormalizeCategory(raw)
		if _, ok := ScannerCatalog[category]; ok { known[category] = struct{}{} } else { unknown[unknownCategoryID(category)] = struct{}{} }
	}
	knownList, unknownList := orderedScannerKeys(known), sortedKeys(unknown)
	matched := make([]string, 0, len(knownList))
	for _, category := range knownList { if _, ok := enabled[category]; ok { matched = append(matched, category) } }
	result := &NormalizedResult{
		Safety: safety, Categories: knownList, MatchedScanners: matched, UnknownCategories: unknownList,
		ScannerScores: map[string]float64{}, ScannerEvidence: map[string]string{},
		ScannerBackend: "qwen3guard-openai", ScannerVersion: "qwen3guard", PolicyID: "priority", PolicyVersion: 1,
		Decision: EventPass, RiskLevel: RiskLow, Action: ActionAllow,
	}
	score := 0.0
	if safety == "Controversial" { score = 0.5; result.Decision, result.RiskLevel, result.Action = EventFlag, RiskMedium, ActionWarn }
	if safety == "Unsafe" {
		score = 1
		if len(matched) > 0 || len(unknownList) > 0 || len(knownList) == 0 {
			result.Decision, result.RiskLevel, result.Action = EventCritical, RiskCritical, ActionBlock
		} else { result.Decision, result.RiskLevel, result.Action = EventFlag, RiskHigh, ActionWarn }
	}
	for _, category := range matched {
		result.ScannerScores[category] = score
		result.ScannerEvidence[category] = ScannerCatalog[category].Label
		if safety == "Controversial" && isElevatedControversial(category) { result.Decision, result.RiskLevel, result.Action = EventCritical, RiskCritical, ActionBlock }
	}
	return result, nil
}

func unknownCategoryID(value string) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(strings.ToLower(value))))
	return fmt.Sprintf("unknown:%x", digest[:8])
}
func isElevatedControversial(category string) bool { return category == "jailbreak" || category == "pii" || category == "suicide_and_self_harm" }

// OpenAICompatibleScanner keeps the established dependency-injection symbol so
// existing generated wiring remains untouched. Its ONLY live protocol is Jev;
// the old chat-completions request and fallback paths have been removed.
type OpenAICompatibleScanner struct { clients sync.Map }
func NewOpenAICompatibleScanner() *OpenAICompatibleScanner { return &OpenAICompatibleScanner{} }

func (s *OpenAICompatibleScanner) Scan(ctx context.Context, endpoint ActiveEndpoint, auditPrompt, chunk string, enabledScanners []string) (*NormalizedResult, error) {
	return s.scanJev(ctx, endpoint, auditPrompt, chunk, enabledScanners)
}

func formatAuditUserInput(chunk string) string {
	encoded, _ := json.Marshal(chunk)
	return "<user_input>\n" + string(encoded) + "\n</user_input>"
}

func (s *OpenAICompatibleScanner) clientFor(endpoint ActiveEndpoint) (*http.Client, error) {
	key := fmt.Sprintf("%s|%s|%d", endpoint.ID, endpoint.BaseURL, endpoint.TimeoutMS)
	if cached, ok := s.clients.Load(key); ok {
		client, valid := cached.(*http.Client)
		if !valid { s.clients.Delete(key); return nil, errors.New("prompt guard client cache invalid") }
		return client, nil
	}
	client, err := NewSecureHTTPClient(endpoint)
	if err != nil { return nil, err }
	actual, _ := s.clients.LoadOrStore(key, client)
	actualClient, ok := actual.(*http.Client)
	if !ok { s.clients.Delete(key); return nil, errors.New("prompt guard client cache invalid") }
	return actualClient, nil
}

func extractOpenAIContent(body []byte) (string, error) {
	var response struct { Choices []struct { Message struct { Content any `json:"content"` } `json:"message"` } `json:"choices"` }
	if err := json.Unmarshal(body, &response); err != nil || len(response.Choices) == 0 { return "", errors.New("prompt guard response envelope invalid") }
	content := response.Choices[0].Message.Content
	switch typed := content.(type) {
	case string:
		if strings.TrimSpace(typed) == "" { return "", errors.New("prompt guard response content empty") }
		return typed, nil
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			object, ok := item.(map[string]any)
			if !ok { continue }
			if text, ok := object["text"].(string); ok && strings.TrimSpace(text) != "" { parts = append(parts, text) }
		}
		if len(parts) == 0 { return "", errors.New("prompt guard response content empty") }
		return strings.Join(parts, "\n"), nil
	default: return "", errors.New("prompt guard response content invalid")
	}
}

func ScannerDefinitions() []ScannerDefinition {
	result := make([]ScannerDefinition, 0, len(AllScannerIDs))
	for _, id := range AllScannerIDs { result = append(result, ScannerCatalog[id]) }
	sort.SliceStable(result, func(i, j int) bool { return i < j })
	return result
}
