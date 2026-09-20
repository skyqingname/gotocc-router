package videoconfig

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

const MaxProviderResponseBytes = 1024 * 1024

var taskIdentifier = regexp.MustCompile(`^[A-Za-z0-9_-]{1,255}$`)

// RequestSpec contains a relative path and body, never a Base URL or credential.
// The application host owns account selection, security audit, holds, timeouts,
// transport, retry policy and result downloads. Building a spec performs no IO.
type RequestSpec struct {
	Method string
	Path string
	ContentType string
	Body []byte
}

type TaskState string
const (
	TaskPending TaskState = "pending"
	TaskSucceeded TaskState = "succeeded"
	TaskFailed TaskState = "failed"
	TaskExpired TaskState = "expired"
)

type TaskResult struct {
	ID string `json:"id"`
	State TaskState `json:"state"`
	ResultURL string `json:"result_url,omitempty"`
	ContentPath string `json:"content_path,omitempty"`
	DurationSeconds int `json:"duration_seconds,omitempty"`
	ProviderModel string `json:"provider_model,omitempty"`
}

func BuildCreate(snapshot Snapshot, prompt string) (RequestSpec, error) {
	if strings.TrimSpace(prompt) == "" || len(prompt) > 32000 { return RequestSpec{}, invalid("prompt", "invalid_prompt") }
	if !identifier.MatchString(snapshot.UpstreamModel) { return RequestSpec{}, invalid("model", "invalid_identifier") }
	allowed := protocolParameters(snapshot.Protocol)
	if allowed == nil { return RequestSpec{}, invalid("protocol", "unsupported_protocol") }
	body := map[string]any{"model": snapshot.UpstreamModel, "prompt": prompt}
	for name, raw := range snapshot.Parameters {
		typeName, ok := allowed[name]
		if !ok { return RequestSpec{}, invalid("parameters", "unknown_parameter") }
		value, err := parameterValue(name, Parameter{Type: typeName}, raw)
		if err != nil { return RequestSpec{}, err }
		switch name {
		case "seconds":
			seconds := value.(int)
			if snapshot.Protocol == OpenAIJSON { body["seconds"] = strconv.Itoa(seconds) } else { body["duration"] = seconds }
		case "reference_image":
			if snapshot.Protocol == OpenAIJSON {
				body["input_reference"] = map[string]string{"image_url": value.(string)}
			} else { body["image"] = map[string]string{"url": value.(string)} }
		default:
			body[name] = value
		}
	}
	if _, exists := snapshot.Parameters["seconds"]; !exists { return RequestSpec{}, invalid("seconds", "required") }
	encoded, err := json.Marshal(body)
	if err != nil { return RequestSpec{}, invalid("body", "invalid_json") }
	path := "/v1/videos"
	if snapshot.Protocol == XAIVideo { path += "/generations" }
	return RequestSpec{Method: http.MethodPost, Path: path, ContentType: "application/json", Body: encoded}, nil
}

// BuildPoll never creates or retries a paid task. A resumed task must keep its
// original protocol and credential-owning account even after configuration edits.
func BuildPoll(protocol, taskID string) (RequestSpec, error) {
	if protocolParameters(protocol) == nil { return RequestSpec{}, invalid("protocol", "unsupported_protocol") }
	if !taskIdentifier.MatchString(taskID) { return RequestSpec{}, invalid("task_id", "invalid_identifier") }
	return RequestSpec{Method: http.MethodGet, Path: "/v1/videos/" + taskID}, nil
}

func ParseCreate(protocol string, raw []byte) (TaskResult, error) {
	object, err := responseObject(raw)
	if err != nil { return TaskResult{}, err }
	if protocol == XAIVideo {
		var id string
		if json.Unmarshal(object["request_id"], &id) != nil || !taskIdentifier.MatchString(id) { return TaskResult{}, invalid("response", "missing_task_id") }
		return TaskResult{ID: id, State: TaskPending}, nil
	}
	if protocol != OpenAIJSON { return TaskResult{}, invalid("protocol", "unsupported_protocol") }
	var id string
	if json.Unmarshal(object["id"], &id) != nil || !taskIdentifier.MatchString(id) { return TaskResult{}, invalid("response", "missing_task_id") }
	return ParsePoll(protocol, id, raw)
}

func ParsePoll(protocol, expectedID string, raw []byte) (TaskResult, error) {
	if !taskIdentifier.MatchString(expectedID) { return TaskResult{}, invalid("task_id", "invalid_identifier") }
	object, err := responseObject(raw)
	if err != nil { return TaskResult{}, err }
	var status, model string
	if json.Unmarshal(object["status"], &status) != nil { return TaskResult{}, invalid("response", "missing_status") }
	if rawModel, exists := object["model"]; exists {
		if json.Unmarshal(rawModel, &model) != nil || !identifier.MatchString(model) { return TaskResult{}, invalid("response", "invalid_model") }
	}
	result := TaskResult{ID: expectedID, ProviderModel: model}
	switch protocol {
	case OpenAIJSON:
		var id string
		if json.Unmarshal(object["id"], &id) != nil || id != expectedID { return TaskResult{}, invalid("response", "task_mismatch") }
		switch status {
		case "queued", "in_progress": result.State = TaskPending
		case "completed":
			result.State = TaskSucceeded
			result.ContentPath = "/v1/videos/" + expectedID + "/content"
		case "failed": result.State = TaskFailed
		default: return TaskResult{}, invalid("response", "unknown_status")
		}
		if seconds, ok := object["seconds"]; ok {
			var text string
			if json.Unmarshal(seconds, &text) != nil { return TaskResult{}, invalid("response", "invalid_duration") }
			n, err := strconv.Atoi(text)
			if err != nil || n < 1 || n > 3600 { return TaskResult{}, invalid("response", "invalid_duration") }
			result.DurationSeconds = n
		}
	case XAIVideo:
		switch status {
		case "pending": result.State = TaskPending
		case "failed": result.State = TaskFailed
		case "expired": result.State = TaskExpired
		case "done":
			video, err := responseObject(object["video"])
			if err != nil { return TaskResult{}, err }
			var location string
			var duration int
			if json.Unmarshal(video["url"], &location) != nil || !validHTTPSURL(location) { return TaskResult{}, invalid("response", "invalid_result_url") }
			if json.Unmarshal(video["duration"], &duration) != nil || duration < 1 || duration > 3600 { return TaskResult{}, invalid("response", "invalid_duration") }
			result.State, result.ResultURL, result.DurationSeconds = TaskSucceeded, location, duration
		default: return TaskResult{}, invalid("response", "unknown_status")
		}
	default: return TaskResult{}, invalid("protocol", "unsupported_protocol")
	}
	return result, nil
}

// Duplicate decision keys and trailing JSON are rejected, while unrelated new
// response fields are retained as compatible provider metadata, not assertions.
func responseObject(raw []byte) (map[string]json.RawMessage, error) {
	if len(raw) == 0 || len(raw) > MaxProviderResponseBytes { return nil, invalid("response", "invalid_size") }
	d := json.NewDecoder(bytes.NewReader(raw))
	token, err := d.Token()
	if err != nil || token != json.Delim('{') { return nil, invalid("response", "invalid_object") }
	out := map[string]json.RawMessage{}
	for d.More() {
		token, err = d.Token()
		if err != nil { return nil, invalid("response", "invalid_json") }
		key, ok := token.(string)
		if !ok { return nil, invalid("response", "invalid_key") }
		if _, exists := out[key]; exists { return nil, invalid("response", "duplicate_key") }
		var value json.RawMessage
		if d.Decode(&value) != nil { return nil, invalid("response", "invalid_json") }
		out[key] = value
	}
	if _, err := d.Token(); err != nil { return nil, invalid("response", "invalid_json") }
	if _, err := d.Token(); !errors.Is(err, io.EOF) { return nil, invalid("response", "trailing_json") }
	return out, nil
}
