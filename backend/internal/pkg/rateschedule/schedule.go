// Package rateschedule validates and evaluates recurring, local-clock rate windows.
// It does not read a database, mutate group prices, or perform billing writes.
package rateschedule

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

const MaxRules = 64

var (
	ErrNotCompiled = errors.New("rate schedule is not compiled")
	ErrInvalidBase = errors.New("base multiplier must be finite and non-negative")
	ErrInvalidTime = errors.New("request time must be supplied")
	ErrOverflow    = errors.New("effective multiplier is not finite")
	ErrUnderflow   = errors.New("positive effective multiplier underflows to zero")
)

// Rule is a daily, half-open [start,end) window. End may be 24:00;
// start > end crosses midnight, but start == end is invalid, not all-day.
// Multiplier is an additional factor, not an absolute group price.
type Rule struct {
	ID         string  `json:"id"`
	Enabled    bool    `json:"enabled"`
	Start      string  `json:"start"`
	End        string  `json:"end"`
	Multiplier float64 `json:"multiplier"`
}

type Config struct {
	Enabled  bool   `json:"enabled"`
	Timezone string `json:"timezone"`
	Rules    []Rule `json:"rules"`
}

// ValidationError contains stable codes/field paths, never raw configuration.
type ValidationError struct {
	Field string
	Code  string
}

func (e *ValidationError) Error() string {
	return "rate schedule " + e.Field + ": " + e.Code
}

type interval struct {
	start, end int
	id         string
	factor     float64
}

// Schedule owns copies of all values and is immutable after Compile. A single
// instance can be shared by concurrent requests. Configuration changes require
// compiling a replacement, not changing a request's existing snapshot.
type Schedule struct {
	enabled  bool
	location *time.Location
	hash     string
	windows  []interval
}

// Snapshot fixes the factor at the trusted request acceptance time. Integrators
// must use this same value for profit admission and settlement, including retries.
// BaseMultiplier must already have resolved the user's override/group default.
// This object contains multipliers, not a monetary charge.
type Snapshot struct {
	ConfigHash          string    `json:"config_hash"`
	Timezone            string    `json:"timezone"`
	RuleID              string    `json:"rule_id,omitempty"`
	Applied             bool      `json:"applied"`
	ReceivedAt          time.Time `json:"received_at"`
	BaseMultiplier      float64   `json:"base_multiplier"`
	ScheduleMultiplier  float64   `json:"schedule_multiplier"`
	EffectiveMultiplier float64   `json:"effective_multiplier"`
}

// Compile validates disabled rules too, so later enabling them cannot expose
// malformed data. Only enabled rules participate in overlap checks. An omitted
// timezone inherits an explicit serverTimezone; neither the host nor browser
// local timezone is guessed. The resolved timezone is bound into ConfigHash.
func Compile(config Config, serverTimezone string) (*Schedule, error) {
	zone := config.Timezone
	if zone == "" {
		zone = serverTimezone
	}
	if !validTimezoneName(zone) {
		return nil, &ValidationError{Field: "timezone", Code: "invalid_timezone"}
	}
	location, err := time.LoadLocation(zone)
	if err != nil {
		return nil, &ValidationError{Field: "timezone", Code: "invalid_timezone"}
	}
	if len(config.Rules) > MaxRules {
		return nil, &ValidationError{Field: "rules", Code: "too_many_rules"}
	}

	// Never retain or sort the caller's slice.
	normalized := Config{Enabled: config.Enabled, Timezone: zone, Rules: make([]Rule, 0, len(config.Rules))}
	windows := make([]interval, 0, len(config.Rules)*2)
	ids := make(map[string]bool, len(config.Rules))
	for i, rule := range config.Rules {
		field := fmt.Sprintf("rules[%d]", i)
		if !validID(rule.ID) {
			return nil, &ValidationError{Field: field + ".id", Code: "invalid_id"}
		}
		if ids[rule.ID] {
			return nil, &ValidationError{Field: field + ".id", Code: "duplicate_id"}
		}
		ids[rule.ID] = true
		start, ok := parseClock(rule.Start, false)
		if !ok {
			return nil, &ValidationError{Field: field + ".start", Code: "invalid_time"}
		}
		end, ok := parseClock(rule.End, true)
		if !ok {
			return nil, &ValidationError{Field: field + ".end", Code: "invalid_time"}
		}
		if start == end {
			return nil, &ValidationError{Field: field + ".end", Code: "empty_window"}
		}
		if !finiteNonNegative(rule.Multiplier) {
			return nil, &ValidationError{Field: field + ".multiplier", Code: "invalid_multiplier"}
		}
		if rule.Multiplier == 0 {
			rule.Multiplier = 0 // Canonicalize negative zero in the configuration hash.
		}
		rule.Start = formatClock(start)
		rule.End = formatClock(end)
		normalized.Rules = append(normalized.Rules, rule)
		if !rule.Enabled {
			continue
		}
		if start < end {
			windows = append(windows, interval{start, end, rule.ID, rule.Multiplier})
		} else {
			windows = append(windows, interval{start, 1440, rule.ID, rule.Multiplier})
			if end > 0 {
				windows = append(windows, interval{0, end, rule.ID, rule.Multiplier})
			}
		}
	}
	sort.Slice(windows, func(i, j int) bool { return windows[i].start < windows[j].start })
	for i := 1; i < len(windows); i++ {
		if windows[i].start < windows[i-1].end {
			return nil, &ValidationError{Field: "rules", Code: "overlapping_windows"}
		}
	}
	// Rule order is not priority: reordering the same normalized configuration
	// must not change its version, matching, or price.
	sort.Slice(normalized.Rules, func(i, j int) bool { return normalized.Rules[i].ID < normalized.Rules[j].ID })
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("encode validated rate schedule: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return &Schedule{
		enabled: config.Enabled, location: location,
		hash: hex.EncodeToString(digest[:]), windows: windows,
	}, nil
}

// Resolve uses only the supplied instant, never time.Now(). Local-clock rules
// repeat in both occurrences of a repeated DST hour; a skipped hour contains no
// real instants. An unmatched or disabled schedule uses factor 1, not price 1.
func (s *Schedule) Resolve(receivedAt time.Time, baseMultiplier float64) (Snapshot, error) {
	if s == nil || s.location == nil {
		return Snapshot{}, ErrNotCompiled
	}
	if !finiteNonNegative(baseMultiplier) {
		return Snapshot{}, ErrInvalidBase
	}
	if receivedAt.IsZero() {
		return Snapshot{}, ErrInvalidTime
	}
	result := Snapshot{
		ConfigHash: s.hash, Timezone: s.location.String(), ReceivedAt: receivedAt.UTC(),
		BaseMultiplier: baseMultiplier, ScheduleMultiplier: 1, EffectiveMultiplier: baseMultiplier,
	}
	if !s.enabled {
		return result, nil
	}
	local := receivedAt.In(s.location)
	minute := local.Hour()*60 + local.Minute()
	for _, window := range s.windows {
		if minute >= window.start && minute < window.end {
			result.RuleID = window.id
			result.Applied = true
			result.ScheduleMultiplier = window.factor
			result.EffectiveMultiplier = baseMultiplier * window.factor
			if !finiteNonNegative(result.EffectiveMultiplier) {
				return Snapshot{}, ErrOverflow
			}
			if baseMultiplier > 0 && window.factor > 0 && result.EffectiveMultiplier == 0 {
				return Snapshot{}, ErrUnderflow
			}
			break
		}
	}
	return result, nil
}

func finiteNonNegative(value float64) bool {
	return value >= 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}

func validID(id string) bool {
	if len(id) == 0 || len(id) > 64 {
		return false
	}
	for _, c := range id {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

func validTimezoneName(zone string) bool {
	if zone == "UTC" {
		return true
	}
	if len(zone) > 128 || !strings.Contains(zone, "/") || strings.HasPrefix(zone, "/") || strings.Contains(zone, "..") {
		return false
	}
	for _, c := range zone {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("/_+-", c)) {
			return false
		}
	}
	return true
}

func parseClock(value string, allowEndOfDay bool) (int, bool) {
	colon := strings.IndexByte(value, ':')
	if (colon != 1 && colon != 2) || len(value)-colon-1 != 2 {
		return 0, false
	}
	hour := 0
	for i := 0; i < colon; i++ {
		if value[i] < '0' || value[i] > '9' {
			return 0, false
		}
		hour = hour*10 + int(value[i]-'0')
	}
	if value[colon+1] < '0' || value[colon+1] > '9' || value[colon+2] < '0' || value[colon+2] > '9' {
		return 0, false
	}
	minute := int(value[colon+1]-'0')*10 + int(value[colon+2]-'0')
	if allowEndOfDay && hour == 24 && minute == 0 {
		return 1440, true
	}
	if hour > 23 || minute > 59 {
		return 0, false
	}
	return hour*60 + minute, true
}

func formatClock(minute int) string {
	return fmt.Sprintf("%02d:%02d", minute/60, minute%60)
}
