package securityaudit

import (
	"encoding/json"
	"math"

	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
)

func validateAuditResponsePolicy(format string, threshold float64) error {
	switch format {
	case "", "qwen3guard": // Legacy settings retain the original classifier.
		return nil
	case "confidence_json":
		if math.IsNaN(threshold) || math.IsInf(threshold, 0) || threshold < 0 || threshold > 1 {
			return infraerrors.BadRequest("prompt_audit_invalid_confidence_threshold", "评分拦截阈值必须在 0 到 1 之间")
		}
		return nil
	default:
		return infraerrors.BadRequest("prompt_audit_invalid_response_format", "审核输出格式必须为 qwen3guard 或 confidence_json")
	}
}

// ParseConfidenceJSON applies the configured threshold to the model's score.
// The explicit format selection keeps JSON scores separate from Qwen categories.
func ParseConfidenceJSON(content string, threshold float64) (*NormalizedResult, error) {
	var response struct {
		Confidence *float64 `json:"confidence"`
		Reason     string   `json:"reason"`
	}
	if err := json.Unmarshal([]byte(content), &response); err != nil || response.Confidence == nil {
		return nil, &GuardError{Code: ErrorCodeInvalidResponse}
	}
	score := *response.Confidence
	if score < 0 || score > 1 {
		return nil, &GuardError{Code: ErrorCodeInvalidResponse}
	}
	result := &NormalizedResult{
		Decision: EventPass, RiskLevel: RiskLow, Action: ActionAllow, Safety: "Safe",
		Categories: []string{}, MatchedScanners: []string{},
		ScannerScores:   map[string]float64{"confidence": score},
		ScannerEvidence: map[string]string{"confidence": RedactPreview(response.Reason, 160)},
		ScannerBackend:  "confidence-json-openai", PolicyID: "confidence", PolicyVersion: 1,
	}
	if score >= threshold {
		result.Decision, result.RiskLevel, result.Action, result.Safety = EventCritical, RiskCritical, ActionBlock, "Unsafe"
		result.MatchedScanners = []string{"confidence"}
	}
	return result, nil
}
