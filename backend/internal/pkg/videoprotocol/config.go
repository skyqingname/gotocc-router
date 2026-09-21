// Package videoprotocol describes each channel model's wire contract.
package videoprotocol

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type Parameter struct {
	Name     string   `json:"name"`
	Label    string   `json:"label,omitempty"`
	Disabled bool     `json:"disabled,omitempty"`
	Type     string   `json:"type"`
	Required bool     `json:"required"`
	Values   []string `json:"values"`
	Min      *float64 `json:"min,omitempty"`
	Max      *float64 `json:"max,omitempty"`
	Step     *float64 `json:"step,omitempty"`
}

type Config struct {
	CreateStatus  string            `json:"create_status"`
	Enabled       bool              `json:"enabled"`
	Protocol      string            `json:"protocol"`
	UpstreamModel string            `json:"upstream_model"`
	CreatePath    string            `json:"create_path"`
	StatusPath    string            `json:"status_path"`
	ContentPath   string            `json:"content_path"`
	Headers       map[string]string `json:"headers"`
	Defaults      map[string]any    `json:"defaults"`
	Parameters    []Parameter       `json:"parameters"`
	RequestFields map[string]string `json:"request_fields"`
	IDField       string            `json:"id_field"`
	StatusField   string            `json:"status_field"`
	VideoURLField string            `json:"video_url_field"`
	Statuses      map[string]string `json:"statuses"`
}

func (c Config) Validate() error {
	if c.Protocol != "openai" && c.Protocol != "custom_json" {
		return fmt.Errorf("video protocol must be openai or custom_json")
	}
	if strings.TrimSpace(c.UpstreamModel) == "" {
		return fmt.Errorf("video upstream model is required")
	}
	for _, path := range []string{c.CreatePath, c.StatusPath, c.ContentPath} {
		if path == "" {
			continue
		}
		u, err := url.Parse(path)
		if err != nil || !strings.HasPrefix(path, "/") || u.IsAbs() || u.Host != "" {
			return fmt.Errorf("video paths must be relative to the account base URL")
		}
	}
	if c.CreatePath == "" || c.StatusPath == "" || !strings.Contains(c.StatusPath, "{task_id}") {
		return fmt.Errorf("video create path and status path with {task_id} are required")
	}
	if c.ContentPath == "" && c.VideoURLField == "" {
		return fmt.Errorf("configure video content path or result URL field")
	}
	if c.IDField == "" || c.StatusField == "" {
		return fmt.Errorf("video response id and status fields are required")
	}
	for key := range c.Headers {
		switch strings.ToLower(key) {
		case "authorization", "x-api-key", "host", "cookie", "user-agent":
			return fmt.Errorf("video header %s is owned by the account", key)
		}
	}
	for key := range c.Defaults {
		switch strings.Split(key, ".")[0] {
		case "prompt", "input", "messages", "model":
			return fmt.Errorf("video defaults cannot replace audited content or model")
		}
	}
	names := map[string]bool{}
	for _, p := range c.Parameters {
		if p.Name == "" || names[p.Name] {
			return fmt.Errorf("video parameter names must be nonempty and unique")
		}
		names[p.Name] = true
		switch p.Type {
		case "string", "number", "integer", "boolean", "array", "object":
		default:
			return fmt.Errorf("invalid video parameter type for %s", p.Name)
		}
		if p.Step != nil && *p.Step <= 0 {
			return fmt.Errorf("video parameter step must be positive for %s", p.Name)
		}
		if p.Min != nil && p.Max != nil && *p.Min > *p.Max {
			return fmt.Errorf("invalid video parameter range for %s", p.Name)
		}
	}
	if len(c.Statuses) == 0 {
		return fmt.Errorf("video status mapping is required")
	}
	if c.CreateStatus != "" && c.CreateStatus != "pending" && c.CreateStatus != "processing" && c.CreateStatus != "completed" {
		return fmt.Errorf("video create_status must be pending, processing or completed")
	}
	if c.Protocol == "custom_json" && len(c.RequestFields) == 0 {
		return fmt.Errorf("custom video request field mapping is required")
	}
	targets := []string{}
	for source, target := range c.RequestFields {
		if source == "" || target == "" {
			return fmt.Errorf("video field mapping paths must not be empty")
		}
		for _, previous := range targets {
			if target == previous || strings.HasPrefix(target, previous+".") || strings.HasPrefix(previous, target+".") {
				return fmt.Errorf("video request target paths overlap")
			}
		}
		targets = append(targets, target)
	}
	for _, status := range c.Statuses {
		switch status {
		case "pending", "processing", "completed", "failed", "cancelled", "expired":
		default:
			return fmt.Errorf("invalid normalized video status")
		}
	}
	defaultsConfig := c
	defaultsConfig.Parameters = append([]Parameter(nil), c.Parameters...)
	for i := range defaultsConfig.Parameters {
		defaultsConfig.Parameters[i].Required = false
	}
	_, err := defaultsConfig.Prepare([]byte(`{}`))
	return err
}

// Prepare operates on canonical fields before billing. Defaults never override
// supplied values. Mapping happens later, after the same content has been audited.
func (c Config) Prepare(body []byte) ([]byte, error) {
	result := append([]byte(nil), body...)
	for name, value := range c.Defaults {
		disabled := false
		for _, p := range c.Parameters {
			if p.Name == name && p.Disabled {
				disabled = true
				break
			}
		}
		if disabled {
			continue
		}
		if !gjson.GetBytes(result, name).Exists() {
			var err error
			result, err = sjson.SetBytes(result, name, value)
			if err != nil {
				return nil, err
			}
		}
	}
	for _, p := range c.Parameters {
		value := gjson.GetBytes(result, p.Name)
		if p.Disabled {
			if value.Exists() {
				return nil, fmt.Errorf("video parameter %s is disabled for this channel model", p.Name)
			}
			continue
		}
		if !value.Exists() || value.Type == gjson.Null {
			if p.Required {
				return nil, fmt.Errorf("video parameter %s is required", p.Name)
			}
			continue
		}
		valid := false
		switch p.Type {
		case "string":
			valid = value.Type == gjson.String
		case "number":
			valid = value.Type == gjson.Number
		case "integer":
			valid = value.Type == gjson.Number && value.Float() == float64(value.Int())
		case "boolean":
			valid = value.Type == gjson.True || value.Type == gjson.False
		case "array":
			valid = value.IsArray()
		case "object":
			valid = value.IsObject()
		}
		if !valid {
			return nil, fmt.Errorf("video parameter %s must be %s", p.Name, p.Type)
		}
		if len(p.Values) > 0 {
			found := false
			for _, allowed := range p.Values {
				if value.String() == allowed {
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("video parameter %s is not an allowed value", p.Name)
			}
		}
		if value.Type == gjson.Number && p.Step != nil {
			origin := 0.0
			if p.Min != nil {
				origin = *p.Min
			}
			steps := (value.Float() - origin) / *p.Step
			if math.Abs(steps-math.Round(steps)) > 1e-8 {
				return nil, fmt.Errorf("video parameter %s does not match its step", p.Name)
			}
		}
		if value.Type == gjson.Number && ((p.Min != nil && value.Float() < *p.Min) || (p.Max != nil && value.Float() > *p.Max)) {
			return nil, fmt.Errorf("video parameter %s is outside its range", p.Name)
		}
	}
	return result, nil
}

func (c Config) MapRequest(body []byte) ([]byte, error) {
	if c.Protocol == "openai" {
		return body, nil
	}
	result := []byte(`{}`)
	for source, target := range c.RequestFields {
		value := gjson.GetBytes(body, source)
		if !value.Exists() {
			continue
		}
		var err error
		result, err = sjson.SetRawBytes(result, target, []byte(value.Raw))
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (c Config) NormalizeResponse(body []byte, taskID string, create bool) ([]byte, error) {
	id := gjson.GetBytes(body, c.IDField).String()
	if id == "" {
		id = taskID
	}
	if id == "" {
		return nil, fmt.Errorf("video provider response has no task id")
	}
	rawStatus := gjson.GetBytes(body, c.StatusField).String()
	status := c.Statuses[rawStatus]

	if status == "" && create && rawStatus == "" {
		status = c.CreateStatus
	}
	if status == "" {
		return nil, fmt.Errorf("unmapped video provider status")
	}
	normalized := map[string]any{"id": id, "object": "video", "status": status, "provider_status": rawStatus}
	if value := gjson.GetBytes(body, c.VideoURLField); c.VideoURLField != "" && value.Type == gjson.String {
		normalized["url"] = value.String()
	}
	if value := gjson.GetBytes(body, "error"); value.Exists() {
		normalized["error"] = value.Value()
	}
	return json.Marshal(normalized)
}
