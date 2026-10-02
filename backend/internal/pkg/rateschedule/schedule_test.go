//go:build unit

package rateschedule

import (
	"errors"
	"math"
	"reflect"
	"sync"
	"testing"
	"time"
)

func mustCompile(t *testing.T, config Config) *Schedule {
	t.Helper()
	s, err := Compile(config, "UTC")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()
	at, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatal(err)
	}
	return at
}

func TestResolveWindows(t *testing.T) {
	s := mustCompile(t, Config{Enabled: true, Timezone: "Asia/Shanghai", Rules: []Rule{
		{ID: "night", Enabled: true, Start: "22:00", End: "06:00", Multiplier: 0.5},
		{ID: "peak", Enabled: true, Start: "18:00", End: "22:00", Multiplier: 1.5},
		{ID: "free", Enabled: true, Start: "12:00", End: "13:00", Multiplier: 0},
	}})
	cases := []struct {
		at, id string
		factor float64
	}{
		{"2026-09-20T21:59:59+08:00", "peak", 1.5},
		{"2026-09-20T22:00:00+08:00", "night", 0.5},
		{"2026-09-21T00:00:00+08:00", "night", 0.5},
		{"2026-09-21T05:59:59+08:00", "night", 0.5},
		{"2026-09-21T06:00:00+08:00", "", 1},
		{"2026-09-21T12:00:00+08:00", "free", 0},
		{"2026-09-21T13:00:00+08:00", "", 1},
		{"2026-09-21T18:00:00+08:00", "peak", 1.5},
		{"2026-09-20T14:00:00Z", "night", 0.5},
	}
	for _, tc := range cases {
		t.Run(tc.at, func(t *testing.T) {
			at := mustTime(t, tc.at)
			got, err := s.Resolve(at, 0.4)
			if err != nil {
				t.Fatal(err)
			}
			if got.RuleID != tc.id || got.Applied != (tc.id != "") || got.ScheduleMultiplier != tc.factor || math.Abs(got.EffectiveMultiplier-0.4*tc.factor) > 1e-12 {
				t.Fatalf("unexpected snapshot: %+v", got)
			}
			if got.Timezone != "Asia/Shanghai" || !got.ReceivedAt.Equal(at) || got.ReceivedAt.Location() != time.UTC || len(got.ConfigHash) != 64 {
				t.Fatalf("incomplete snapshot: %+v", got)
			}
		})
	}
}

func TestDisabledAndEmptySchedulesKeepBase(t *testing.T) {
	for _, config := range []Config{
		{Enabled: false, Rules: []Rule{{ID: "off", Enabled: true, Start: "00:00", End: "24:00", Multiplier: 9}}},
		{Enabled: true},
		{Enabled: true, Rules: []Rule{{ID: "off", Enabled: false, Start: "00:00", End: "24:00", Multiplier: 9}}},
	} {
		s := mustCompile(t, config)
		for _, base := range []float64{0, 0.2, 1, 8} {
			got, err := s.Resolve(mustTime(t, "2026-09-20T18:00:00Z"), base)
			if err != nil || got.Applied || got.EffectiveMultiplier != base || got.ScheduleMultiplier != 1 {
				t.Fatalf("disabled/empty schedule changed base: %+v, %v", got, err)
			}
		}
	}
}

func TestMidnightAndAllDay(t *testing.T) {
	for _, rule := range []Rule{
		{ID: "all", Enabled: true, Start: "00:00", End: "24:00", Multiplier: 0.5},
		{ID: "evening", Enabled: true, Start: "18:00", End: "00:00", Multiplier: 0.5},
	} {
		s := mustCompile(t, Config{Enabled: true, Rules: []Rule{rule}})
		for minute := 0; minute < 1440; minute++ {
			at := time.Date(2026, 9, 20, minute/60, minute%60, 0, 0, time.UTC)
			got, err := s.Resolve(at, 0.4)
			wantApplied := rule.ID == "all" || minute >= 18*60
			if err != nil || got.Applied != wantApplied {
				t.Fatalf("rule=%s minute=%d: %+v, %v", rule.ID, minute, got, err)
			}
		}
	}
}

func TestValidation(t *testing.T) {
	valid := Rule{ID: "one", Enabled: true, Start: "01:00", End: "02:00", Multiplier: 1}
	cases := []struct {
		name, code string
		mutate     func(*Config)
	}{
		{"local", "invalid_timezone", func(c *Config) { c.Timezone = "Local" }},
		{"path", "invalid_timezone", func(c *Config) { c.Timezone = "../etc/passwd" }},
		{"unknown zone", "invalid_timezone", func(c *Config) { c.Timezone = "Invalid/Zone" }},
		{"empty id", "invalid_id", func(c *Config) { c.Rules[0].ID = "" }},
		{"unsafe id", "invalid_id", func(c *Config) { c.Rules[0].ID = "one\n" }},
		{"duplicate", "duplicate_id", func(c *Config) { c.Rules = append(c.Rules, valid) }},
		{"negative", "invalid_multiplier", func(c *Config) { c.Rules[0].Multiplier = -1 }},
		{"nan", "invalid_multiplier", func(c *Config) { c.Rules[0].Multiplier = math.NaN() }},
		{"infinity", "invalid_multiplier", func(c *Config) { c.Rules[0].Multiplier = math.Inf(1) }},
		{"start 24", "invalid_time", func(c *Config) { c.Rules[0].Start = "24:00" }},
		{"end 24:01", "invalid_time", func(c *Config) { c.Rules[0].End = "24:01" }},
		{"short minute", "invalid_time", func(c *Config) { c.Rules[0].Start = "1:1" }},
		{"whitespace", "invalid_time", func(c *Config) { c.Rules[0].Start = " 01:00" }},
		{"equal", "empty_window", func(c *Config) { c.Rules[0].End = "01:00" }},
		{"overlap", "overlapping_windows", func(c *Config) {
			c.Rules = append(c.Rules, Rule{ID: "two", Enabled: true, Start: "01:30", End: "03:00", Multiplier: 2})
		}},
		{"overnight overlap", "overlapping_windows", func(c *Config) {
			c.Rules = append(c.Rules, Rule{ID: "two", Enabled: true, Start: "22:00", End: "01:30", Multiplier: 2})
		}},
		{"too many", "too_many_rules", func(c *Config) { c.Rules = make([]Rule, MaxRules+1) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := Config{Enabled: true, Timezone: "UTC", Rules: []Rule{valid}}
			tc.mutate(&c)
			_, err := Compile(c, "UTC")
			var validation *ValidationError
			if !errors.As(err, &validation) || validation.Code != tc.code {
				t.Fatalf("want %s, got %v", tc.code, err)
			}
		})
	}
	if _, err := Compile(Config{}, ""); err == nil {
		t.Fatal("missing server timezone silently accepted")
	}
	mustCompile(t, Config{Enabled: true, Rules: []Rule{valid, {ID: "off", Enabled: false, Start: "00:00", End: "24:00", Multiplier: 1}}})
}

func TestRequestErrors(t *testing.T) {
	at := mustTime(t, "2026-09-20T00:00:00Z")
	s := mustCompile(t, Config{Enabled: true, Rules: []Rule{{ID: "all", Enabled: true, Start: "00:00", End: "24:00", Multiplier: 2}}})
	for _, base := range []float64{-1, math.NaN(), math.Inf(1), math.Inf(-1)} {
		if _, err := s.Resolve(at, base); !errors.Is(err, ErrInvalidBase) {
			t.Fatalf("invalid base %v: %v", base, err)
		}
	}
	if _, err := s.Resolve(time.Time{}, 1); !errors.Is(err, ErrInvalidTime) {
		t.Fatalf("zero request time: %v", err)
	}
	if _, err := s.Resolve(at, math.MaxFloat64); !errors.Is(err, ErrOverflow) {
		t.Fatalf("overflow: %v", err)
	}
	var uncompiled *Schedule
	if _, err := uncompiled.Resolve(at, 1); !errors.Is(err, ErrNotCompiled) {
		t.Fatalf("nil schedule: %v", err)
	}
	tiny := mustCompile(t, Config{Enabled: true, Rules: []Rule{{ID: "tiny", Enabled: true, Start: "00:00", End: "24:00", Multiplier: 0.1}}})
	if _, err := tiny.Resolve(at, math.SmallestNonzeroFloat64); !errors.Is(err, ErrUnderflow) {
		t.Fatalf("underflow: %v", err)
	}
}

func TestExplicitTimezoneAndDST(t *testing.T) {
	s := mustCompile(t, Config{Enabled: true, Timezone: "America/New_York", Rules: []Rule{{ID: "repeat", Enabled: true, Start: "01:00", End: "02:00", Multiplier: 0.5}}})
	for _, value := range []string{"2025-11-02T05:30:00Z", "2025-11-02T06:30:00Z"} {
		got, err := s.Resolve(mustTime(t, value), 0.4)
		if err != nil || got.RuleID != "repeat" || got.EffectiveMultiplier != 0.2 {
			t.Fatalf("repeated DST hour: %+v, %v", got, err)
		}
	}
	spring := mustCompile(t, Config{Enabled: true, Timezone: "America/New_York", Rules: []Rule{{ID: "missing", Enabled: true, Start: "02:00", End: "03:00", Multiplier: 0}}})
	for _, value := range []string{"2025-03-09T06:59:59Z", "2025-03-09T07:00:00Z"} {
		got, err := spring.Resolve(mustTime(t, value), 0.4)
		if err != nil || got.Applied || got.EffectiveMultiplier != 0.4 {
			t.Fatalf("skipped DST hour: %+v, %v", got, err)
		}
	}
}

func TestNormalizedVersionAndImmutableSnapshot(t *testing.T) {
	config := Config{Enabled: true, Timezone: "UTC", Rules: []Rule{
		{ID: "b", Enabled: true, Start: "1:30", End: "02:00", Multiplier: 0.5},
		{ID: "a", Enabled: true, Start: "03:00", End: "04:00", Multiplier: 2},
	}}
	s := mustCompile(t, config)
	at := mustTime(t, "2026-09-20T01:59:59Z")
	before, err := s.Resolve(at, 0.2)
	if err != nil {
		t.Fatal(err)
	}
	if config.Rules[0].Start != "1:30" || config.Rules[0].ID != "b" {
		t.Fatal("Compile mutated caller-owned configuration")
	}
	canonical := Config{Enabled: true, Timezone: "UTC", Rules: []Rule{config.Rules[1], config.Rules[0]}}
	canonical.Rules[1].Start = "01:30"
	if mustCompile(t, canonical).hash != s.hash {
		t.Fatal("equivalent rule order/clock formats changed the version")
	}
	config.Rules[0].Multiplier = 10
	replacement := mustCompile(t, config)
	if replacement.hash == s.hash {
		t.Fatal("changed config reused the version")
	}
	after, err := s.Resolve(at, 0.2)
	if err != nil || !reflect.DeepEqual(before, after) || before.EffectiveMultiplier != 0.1 {
		t.Fatalf("existing schedule/snapshot mutated: %+v -> %+v, %v", before, after, err)
	}
	later, err := s.Resolve(at.Add(time.Second), 0.2)
	if err != nil || later.Applied || before.EffectiveMultiplier != 0.1 {
		t.Fatal("crossing the boundary modified the original snapshot")
	}
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := s.Resolve(at, 0.2)
			if err != nil || !reflect.DeepEqual(before, got) {
				t.Errorf("concurrent resolution differed: %+v, %v", got, err)
			}
		}()
	}
	wg.Wait()
}
