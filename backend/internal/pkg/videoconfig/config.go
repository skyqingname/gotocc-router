// Package videoconfig owns the declarative video model configuration contract.
// It does not execute scripts, choose credentials, call providers or move money.
package videoconfig

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/shopspring/decimal"
)

const (
	OpenAIJSON = "openai_video_json"
	XAIVideo = "xai_video"
	MaxBindings = 128
	MaxConfigBytes = 120 * 1024
)

var identifier = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:/-]{0,127}$`)
var priceNumber = regexp.MustCompile(`^(0|[1-9][0-9]{0,5})(\.[0-9]{1,10})?$`)

type Parameter struct {
	Type string `json:"type"`
	Required bool `json:"required"`
	Default json.RawMessage `json:"default,omitempty"`
	Locked bool `json:"locked"`
	Minimum *int `json:"minimum,omitempty"`
	Maximum *int `json:"maximum,omitempty"`
	Enum []json.RawMessage `json:"enum,omitempty"`
}

type Price struct {
	Unit string `json:"unit"`
	USD string `json:"usd"`
	Seconds *int `json:"seconds,omitempty"`
	Resolution string `json:"resolution,omitempty"`
	Size string `json:"size,omitempty"`
	GenerateAudio *bool `json:"generate_audio,omitempty"`
}

type Binding struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Enabled bool `json:"enabled"`
	AccountID int64 `json:"account_id"`
	PublicModel string `json:"public_model"`
	UpstreamModel string `json:"upstream_model"`
	Protocol string `json:"protocol"`
	Parameters map[string]Parameter `json:"parameters"`
	Prices []Price `json:"prices"`
}

type Config struct {
	Bindings []Binding `json:"bindings"`
}

type ValidationError struct { Field, Code string }
func (e *ValidationError) Error() string { return "video configuration " + e.Field + ": " + e.Code }
func invalid(field, code string) error { return &ValidationError{Field: field, Code: code} }

// Compiled owns its configuration. Accessors return copies, not mutable maps.
type Compiled struct {
	hash string
	bindings map[string]Binding
}

// Snapshot must be stored with the durable task before a billable submission.
// It describes a quote; it is NOT proof that a balance hold or audit succeeded.
type Snapshot struct {
	ConfigHash string `json:"config_hash"`
	BindingID string `json:"binding_id"`
	AccountID int64 `json:"account_id"`
	PublicModel string `json:"public_model"`
	UpstreamModel string `json:"upstream_model"`
	Protocol string `json:"protocol"`
	Parameters map[string]json.RawMessage `json:"parameters"`
	BaseCostUSD string `json:"base_cost_usd"`
}

func Decode(raw []byte) (Config, error) {
	if len(raw) > MaxConfigBytes { return Config{}, invalid("config", "too_large") }
	var config Config
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&config); err != nil { return Config{}, invalid("config", "invalid_json") }
	if err := d.Decode(new(any)); err != io.EOF { return Config{}, invalid("config", "trailing_json") }
	if _, err := Compile(config); err != nil { return Config{}, err }
	return config, nil
}

func Compile(config Config) (*Compiled, error) {
	encoded, err := json.Marshal(config)
	if err != nil || len(encoded) > MaxConfigBytes { return nil, invalid("config", "too_large_or_invalid") }
	var owned Config
	if json.Unmarshal(encoded, &owned) != nil { return nil, invalid("config", "invalid_json") }
	if len(owned.Bindings) > MaxBindings { return nil, invalid("bindings", "too_many_bindings") }
	compiled := &Compiled{bindings: make(map[string]Binding, len(owned.Bindings))}
	for i, binding := range owned.Bindings {
		field := fmt.Sprintf("bindings[%d]", i)
		if !identifier.MatchString(binding.ID) || !identifier.MatchString(binding.PublicModel) || !identifier.MatchString(binding.UpstreamModel) {
			return nil, invalid(field, "invalid_identifier")
		}
		if strings.TrimSpace(binding.Name) == "" || len(binding.Name) > 200 || binding.AccountID <= 0 {
			return nil, invalid(field, "invalid_name_or_account")
		}
		if _, exists := compiled.bindings[binding.ID]; exists { return nil, invalid(field, "duplicate_binding") }
		allowed := protocolParameters(binding.Protocol)
		if allowed == nil { return nil, invalid(field+".protocol", "unsupported_protocol") }
		if len(binding.Parameters) == 0 || len(binding.Parameters) > len(allowed) { return nil, invalid(field+".parameters", "invalid_parameters") }
		for name, param := range binding.Parameters {
			expectedType, ok := allowed[name]
			if !ok || param.Type != expectedType { return nil, invalid(field+".parameters", "unsupported_parameter") }
			if param.Type != "integer" && (param.Minimum != nil || param.Maximum != nil) { return nil, invalid(field+".parameters", "invalid_range") }
			if param.Minimum != nil && param.Maximum != nil && *param.Minimum > *param.Maximum { return nil, invalid(field+".parameters", "invalid_range") }
			if len(param.Enum) > 128 { return nil, invalid(field+".parameters", "too_many_options") }
			for _, option := range param.Enum {
				withoutEnum := param
				withoutEnum.Enum = nil
				if _, err := parameterValue(name, withoutEnum, option); err != nil { return nil, invalid(field+".parameters", "invalid_option") }
			}
			if param.Locked && len(param.Default) == 0 { return nil, invalid(field+".parameters", "locked_default_required") }
			if len(param.Default) > 0 {
				if _, err := parameterValue(name, param, param.Default); err != nil { return nil, invalid(field+".parameters", "invalid_default") }
			}
		}
		seconds, hasSeconds := binding.Parameters["seconds"]
		if !hasSeconds || (!seconds.Required && len(seconds.Default) == 0) { return nil, invalid(field, "duration_required") }
		if len(binding.Prices) == 0 || len(binding.Prices) > 128 { return nil, invalid(field+".prices", "prices_required") }
		for j, price := range binding.Prices {
			if (price.Unit != "per_job" && price.Unit != "per_second") || !priceNumber.MatchString(price.USD) { return nil, invalid(field+".prices", "invalid_price") }
			for name, value := range priceSelectors(price) {
				param, exists := binding.Parameters[name]
				if !exists { return nil, invalid(field+".prices", "unknown_price_dimension") }
				raw, _ := json.Marshal(value)
				if _, err := parameterValue(name, param, raw); err != nil { return nil, invalid(field+".prices", "invalid_price_dimension") }
			}
			for k := 0; k < j; k++ {
				if pricesOverlap(price, binding.Prices[k]) { return nil, invalid(field+".prices", "overlapping_prices") }
			}
		}
		compiled.bindings[binding.ID] = binding
	}
	sort.Slice(owned.Bindings, func(i, j int) bool { return owned.Bindings[i].ID < owned.Bindings[j].ID })
	encoded, _ = json.Marshal(owned)
	digest := sha256.Sum256(encoded)
	compiled.hash = hex.EncodeToString(digest[:])
	return compiled, nil
}

func protocolParameters(protocol string) map[string]string {
	switch protocol {
	case OpenAIJSON:
		return map[string]string{"seconds": "integer", "size": "string", "reference_image": "string"}
	case XAIVideo:
		return map[string]string{"seconds": "integer", "aspect_ratio": "string", "resolution": "string", "generate_audio": "boolean", "reference_image": "string"}
	default:
		return nil
	}
}

func parameterValue(name string, parameter Parameter, raw json.RawMessage) (any, error) {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) { return nil, invalid(name, "value_required") }
	var value any
	switch parameter.Type {
	case "integer":
		var n int
		if json.Unmarshal(raw, &n) != nil || (parameter.Minimum != nil && n < *parameter.Minimum) || (parameter.Maximum != nil && n > *parameter.Maximum) {
			return nil, invalid(name, "invalid_integer")
		}
		if name == "seconds" && (n < 1 || n > 3600) { return nil, invalid(name, "invalid_duration") }
		value = n
	case "string":
		var text string
		if json.Unmarshal(raw, &text) != nil || len(text) == 0 || len(text) > 4096 { return nil, invalid(name, "invalid_string") }
		if name == "reference_image" && !validHTTPSURL(text) { return nil, invalid(name, "https_reference_required") }
		value = text
	case "boolean":
		var flag bool
		if json.Unmarshal(raw, &flag) != nil { return nil, invalid(name, "invalid_boolean") }
		value = flag
	default:
		return nil, invalid(name, "invalid_type")
	}
	if len(parameter.Enum) > 0 {
		normalized, _ := json.Marshal(value)
		matched := false
		for _, option := range parameter.Enum {
			var buffer bytes.Buffer
			if json.Compact(&buffer, option) == nil && bytes.Equal(normalized, buffer.Bytes()) { matched = true; break }
		}
		if !matched { return nil, invalid(name, "unsupported_value") }
	}
	return value, nil
}

// Resolve rejects unknown control fields and conflicting locked values instead
// of silently ignoring them. Caller input can never choose a URL or credential.
func (c *Compiled) Resolve(bindingID string, input map[string]json.RawMessage) (Snapshot, error) {
	if c == nil { return Snapshot{}, errors.New("video configuration not compiled") }
	binding, exists := c.bindings[bindingID]
	if !exists || !binding.Enabled { return Snapshot{}, invalid("binding", "unavailable") }
	for key := range input { if _, ok := binding.Parameters[key]; !ok { return Snapshot{}, invalid("parameters", "unknown_parameter") } }
	resolved := make(map[string]json.RawMessage, len(binding.Parameters))
	values := make(map[string]any, len(binding.Parameters))
	for name, param := range binding.Parameters {
		raw, supplied := input[name]
		if !supplied { raw = param.Default }
		if len(raw) == 0 {
			if param.Required { return Snapshot{}, invalid(name, "required") }
			continue
		}
		value, err := parameterValue(name, param, raw)
		if err != nil { return Snapshot{}, err }
		canonical, _ := json.Marshal(value)
		if supplied && param.Locked {
			locked, err := parameterValue(name, param, param.Default)
			if err != nil { return Snapshot{}, err }
			lockedJSON, _ := json.Marshal(locked)
			if !bytes.Equal(canonical, lockedJSON) { return Snapshot{}, invalid(name, "locked") }
		}
		resolved[name], values[name] = canonical, value
	}
	var chosen *Price
	for i := range binding.Prices {
		price := &binding.Prices[i]
		matches := true
		for name, required := range priceSelectors(*price) { if values[name] != required { matches = false; break } }
		if matches { chosen = price; break }
	}
	if chosen == nil { return Snapshot{}, invalid("price", "unpriced_variant") }
	amount, err := decimal.NewFromString(chosen.USD)
	if err != nil { return Snapshot{}, invalid("price", "invalid_price") }
	if chosen.Unit == "per_second" {
		seconds, ok := values["seconds"].(int)
		if !ok { return Snapshot{}, invalid("seconds", "required") }
		amount = amount.Mul(decimal.NewFromInt(int64(seconds)))
	}
	return Snapshot{ConfigHash: c.hash, BindingID: binding.ID, AccountID: binding.AccountID,
		PublicModel: binding.PublicModel, UpstreamModel: binding.UpstreamModel, Protocol: binding.Protocol,
		Parameters: resolved, BaseCostUSD: amount.String()}, nil
}

func priceSelectors(price Price) map[string]any {
	selectors := map[string]any{}
	if price.Seconds != nil { selectors["seconds"] = *price.Seconds }
	if price.Resolution != "" { selectors["resolution"] = price.Resolution }
	if price.Size != "" { selectors["size"] = price.Size }
	if price.GenerateAudio != nil { selectors["generate_audio"] = *price.GenerateAudio }
	return selectors
}

func pricesOverlap(a, b Price) bool {
	left, right := priceSelectors(a), priceSelectors(b)
	for name, value := range left { if other, exists := right[name]; exists && value != other { return false } }
	return true
}

// This is lexical validation only. The host must still resolve DNS and enforce
// its existing SSRF, redirect, size and credential-isolation rules on every fetch.
func validHTTPSURL(value string) bool {
	u, err := url.Parse(value)
	return err == nil && u.Scheme == "https" && u.Hostname() != "" && u.User == nil && u.Fragment == "" && !strings.ContainsAny(value, "\r\n")
}
