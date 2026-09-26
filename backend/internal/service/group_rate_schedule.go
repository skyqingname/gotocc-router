package service

import (
	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/rateschedule"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/timezone"
)

func cloneGroupRateSchedule(value rateschedule.Config) rateschedule.Config {
	value.Rules = append([]rateschedule.Rule{}, value.Rules...)
	return value
}

func normalizeGroupRateSchedule(input *rateschedule.Config, subscriptionType string, enabled bool, start, end string, multiplier float64) (rateschedule.Config, error) {
	value := rateschedule.Config{Rules: []rateschedule.Rule{}}
	if input != nil {
		value = cloneGroupRateSchedule(*input)
	} else if subscriptionType == SubscriptionTypeSubscription && start != "" && end != "" {
		value.Enabled = enabled
		value.Rules = []rateschedule.Rule{{ID: "legacy-peak", Enabled: true, Start: start, End: end, Multiplier: multiplier}}
	}
	if _, err := rateschedule.Compile(value, timezone.Name()); err != nil {
		return value, infraerrors.BadRequest("INVALID_RATE_SCHEDULE", err.Error())
	}
	return value, nil
}
