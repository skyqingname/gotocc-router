package service

// GoToCC：内容审计接入 TypeSafe System One。一次请求按合并后的风险类别提问，各类别 noul
// 概率作为分数进入内容审计原有的阈值、拦截、通知与封号流程。

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/outboundidentity"
)

const (
	ContentModerationProtocolOpenAIModeration = "openai_moderation"
	ContentModerationProtocolTypeSafeJev      = "typesafe_jev"
)

//go:embed content_audit_jev_gen.json
var contentAuditJevJSON []byte

type contentAuditJevRisk struct {
	Category        string   `json:"category"`
	QuestionID      string   `json:"question_id"`
	Description     string   `json:"description"`
	Threshold       float64  `json:"threshold"`
	ScoreCategories []string `json:"score_categories"`
}

type contentAuditJevParameters struct {
	BaseURL           string                `json:"base_url"`
	Model             string                `json:"model"`
	InputLimitRunes   int                   `json:"input_limit_runes"`
	Policy            string                `json:"policy"`
	Question          string                `json:"question"`
	Criteria          map[string]string     `json:"criteria"`
	Risks             []contentAuditJevRisk `json:"risks"`
	RetiredCategories []string              `json:"retired_categories"`
}

var contentAuditJev = func() contentAuditJevParameters {
	var parameters contentAuditJevParameters
	if err := json.Unmarshal(contentAuditJevJSON, &parameters); err != nil {
		panic(fmt.Sprintf("content audit Jev parameters: %v", err))
	}
	return parameters
}()

func contentModerationInputText(input any) string {
	switch value := input.(type) {
	case string:
		return value
	case []moderationAPIInputPart:
		texts := make([]string, 0, len(value))
		for _, part := range value {
			if part.Type == "text" {
				texts = append(texts, part.Text)
			}
		}
		return strings.Join(texts, "\n")
	default:
		return ""
	}
}

// Jev 只评估文字；超过单段上限的文本切段并行提交，每个类别取各段最高概率。
func (s *ContentModerationService) callJevOnce(ctx context.Context, cfg *ContentModerationConfig, apiKey string, input any, httpStatus *int) (*moderationAPIResult, error) {
	text := []rune(strings.TrimSpace(contentModerationInputText(input)))
	if len(text) == 0 {
		return nil, errors.New("typesafe jev requires text input")
	}
	limit := contentAuditJev.InputLimitRunes
	chunks := make([]string, 0, (len(text)+limit-1)/limit)
	for start := 0; start < len(text); start += limit {
		chunks = append(chunks, string(text[start:min(start+limit, len(text))]))
	}
	type outcome struct {
		scores map[string]float64
		status int
		err    error
	}
	outcomes := make([]outcome, len(chunks))
	var wg sync.WaitGroup
	for index, chunk := range chunks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result := &outcomes[index]
			result.scores, result.err = s.callJevChunk(ctx, cfg, apiKey, chunk, &result.status)
		}()
	}
	wg.Wait()
	scores := make(map[string]float64, len(contentAuditJev.Risks))
	for _, result := range outcomes {
		if httpStatus != nil {
			*httpStatus = result.status
		}
		if result.err != nil {
			return nil, result.err
		}
		for category, score := range result.scores {
			scores[category] = math.Max(scores[category], score)
		}
	}
	return &moderationAPIResult{CategoryScores: scores}, nil
}

func (s *ContentModerationService) callJevChunk(ctx context.Context, cfg *ContentModerationConfig, apiKey, content string, httpStatus *int) (map[string]float64, error) {
	questions := make(map[string]any, len(contentAuditJev.Risks))
	for _, risk := range contentAuditJev.Risks {
		questions[risk.QuestionID] = map[string]any{
			"type":         "noul",
			"instructions": cfg.JevPolicy + "\n\n" + strings.ReplaceAll(contentAuditJev.Question, "{risk}", risk.Description),
			"criteria":     contentAuditJev.Criteria,
		}
	}
	raw, err := json.Marshal(map[string]any{"model": cfg.Model, "state": map[string]string{"content": content}, "questions": questions})
	if err != nil {
		return nil, err
	}
	target := strings.TrimRight(cfg.BaseURL, "/")
	if !strings.HasSuffix(target, "/v1/systemone") {
		target = strings.TrimSuffix(target, "/v1") + "/v1/systemone"
	}
	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.TimeoutMS)*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, target, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	outboundidentity.ApplyContext(req)
	client, err := s.moderationHTTPClient(ctx, cfg)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	*httpStatus = resp.StatusCode
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read typesafe jev response (HTTP %d): %w", resp.StatusCode, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("typesafe jev status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out struct {
		Answers map[string]struct {
			Type string   `json:"type"`
			Noul *float64 `json:"noul"`
		} `json:"answers"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode typesafe jev response: %w; response: %s", err, string(body))
	}
	scores := make(map[string]float64, len(contentAuditJev.Risks))
	for _, risk := range contentAuditJev.Risks {
		answer, ok := out.Answers[risk.QuestionID]
		if !ok || answer.Type != "noul" || answer.Noul == nil || math.IsNaN(*answer.Noul) || *answer.Noul < 0 || *answer.Noul > 1 {
			return nil, fmt.Errorf("typesafe jev returned no valid noul for %s; response: %s", risk.QuestionID, string(body))
		}
		scores[risk.Category] = *answer.Noul
	}
	return scores, nil
}
