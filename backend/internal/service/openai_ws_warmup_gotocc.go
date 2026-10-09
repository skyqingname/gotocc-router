package service

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tidwall/gjson"
)

func isOpenAIWSLocalWarmup(payload []byte) bool {
	return gjson.GetBytes(payload, "generate").Type == gjson.False
}

// completeOpenAIWSLocalWarmup acknowledges preparation without running HTTP
// inference. The bridge turn loop retains the input and tool mapping for replay.
func completeOpenAIWSLocalWarmup(model string, write func([]byte) error) (*OpenAIForwardResult, error) {
	started := time.Now()
	id := "resp_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	response := map[string]any{
		"id": id, "object": "response", "created_at": started.Unix(),
		"model": model, "status": "in_progress", "output": []any{},
		"error": nil, "incomplete_details": nil,
	}
	for sequence, eventType := range []string{"response.created", "response.completed"} {
		if eventType == "response.completed" {
			response["status"] = "completed"
			response["usage"] = map[string]any{
				"input_tokens": 0, "output_tokens": 0, "total_tokens": 0,
				"input_tokens_details":  map[string]int{"cached_tokens": 0},
				"output_tokens_details": map[string]int{"reasoning_tokens": 0},
			}
		}
		message, err := json.Marshal(map[string]any{
			"type": eventType, "sequence_number": sequence, "response": response,
		})
		if err != nil {
			return nil, err
		}
		if err := write(message); err != nil {
			return nil, err
		}
	}
	return &OpenAIForwardResult{
		RequestID: id, ResponseID: id, Model: model,
		Stream: true, OpenAIWSMode: true, UpstreamTerminalEvent: "response.completed",
		Duration: time.Since(started),
	}, nil
}
