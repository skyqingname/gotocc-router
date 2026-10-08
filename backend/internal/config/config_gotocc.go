package config

import (
	"fmt"
	"strings"
)

// TeamConfig controls whether users can create and operate a single-owner team.
type TeamConfig struct {
	Enabled            bool `mapstructure:"enabled"`
	SelfServiceEnabled bool `mapstructure:"self_service_enabled"`
	DefaultMemberLimit int  `mapstructure:"default_member_limit"`
}

// AsyncImageConfig bounds multipart uploads accepted by asynchronous image edits.
type AsyncImageConfig struct {
	EditMaxInputImages int `mapstructure:"edit_max_input_images"`
}

// VideoTaskConfig controls durable polling and terminal billing for
// OpenAI-compatible asynchronous video tasks.
type VideoTaskConfig struct {
	Enabled               bool     `mapstructure:"enabled"`
	ScanIntervalSeconds   int      `mapstructure:"scan_interval_seconds"`
	TaskTimeoutSeconds    int      `mapstructure:"task_timeout_seconds"`
	LeaseSeconds          int      `mapstructure:"lease_seconds"`
	RequestTimeoutSeconds int      `mapstructure:"request_timeout_seconds"`
	ClaimBatchSize        int      `mapstructure:"claim_batch_size"`
	MaxResponseBytes      int64    `mapstructure:"max_response_bytes"`
	SuccessStatuses       []string `mapstructure:"success_statuses"`
	FailureStatuses       []string `mapstructure:"failure_statuses"`
	CancelledStatuses     []string `mapstructure:"cancelled_statuses"`
}

func validateVideoTaskTerminalStatuses(cfg VideoTaskConfig) error {
	sets := []struct {
		name   string
		values []string
	}{
		{name: "success_statuses", values: cfg.SuccessStatuses},
		{name: "failure_statuses", values: cfg.FailureStatuses},
		{name: "cancelled_statuses", values: cfg.CancelledStatuses},
	}
	seen := make(map[string]string)
	for _, set := range sets {
		if len(set.values) == 0 {
			return fmt.Errorf("video_task.%s must not be empty", set.name)
		}
		for _, value := range set.values {
			normalized := strings.ToLower(strings.TrimSpace(value))
			if normalized == "" {
				return fmt.Errorf("video_task.%s must not contain empty status", set.name)
			}
			if previous, ok := seen[normalized]; ok {
				return fmt.Errorf("video_task status %q appears in both %s and %s", normalized, previous, set.name)
			}
			seen[normalized] = set.name
		}
	}
	return nil
}
