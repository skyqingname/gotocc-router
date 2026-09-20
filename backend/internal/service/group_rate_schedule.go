package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/rateschedule"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/timezone"
)

const GroupRateScheduleKeyPrefix = "group_rate_schedule:"

var (
	ErrGroupFeatureConflict = infraerrors.Conflict("group_feature_conflict", "Configuration changed; reload before saving")
	ErrGroupFeatureUnavailable = infraerrors.ServiceUnavailable("group_feature_unavailable", "Group configuration is temporarily unavailable")
)

// GroupFeatureSettingsStore is an optional capability implemented by the existing
// settings repository. The CAS locks the group and setting in one transaction;
// the regular SettingRepository interface and its existing mocks stay unchanged.
type GroupFeatureSettingsStore interface {
	CompareAndSwapGroupFeature(ctx context.Context, groupID int64, key, expected, replacement string) error
}

type GroupRateScheduleSettings struct {
	Version int64 `json:"version"`
	Config rateschedule.Config `json:"config"`
	UpdatedAt time.Time `json:"updated_at"`
}

type GroupRateScheduleView struct {
	GroupID int64 `json:"group_id"`
	Version int64 `json:"version"`
	Config rateschedule.Config `json:"config"`
	ServerTimezone string `json:"server_timezone"`
	Source string `json:"source"`
	UpdatedAt time.Time `json:"updated_at"`
}

// A request owns this object; it is never placed in a shared API-key cache.
// HTTP factors use the accepted instant. Live turns use their supplied turn
// instant with the same validated configuration, not the handshake's factor.
type GroupRateScheduleRuntime struct {
	Schedule *rateschedule.Schedule
	AcceptedAt time.Time
	Live bool
	Version int64
}

func (s *SettingService) GetGroupRateSchedule(ctx context.Context, group *Group) (GroupRateScheduleView, error) {
	if group == nil || group.ID <= 0 { return GroupRateScheduleView{}, ErrGroupNotFound }
	stored, _, exists, err := s.loadGroupRateSchedule(ctx, group.ID)
	if err != nil { return GroupRateScheduleView{}, err }
	view := GroupRateScheduleView{GroupID: group.ID, ServerTimezone: timezone.Location().String(), Source: "legacy", Config: legacyGroupRateSchedule(group)}
	if exists {
		view.Version, view.Config, view.UpdatedAt, view.Source = stored.Version, stored.Config, stored.UpdatedAt, "schedule"
	}
	return view, nil
}

func (s *SettingService) SaveGroupRateSchedule(ctx context.Context, groupID, expectedVersion int64, config rateschedule.Config) (GroupRateScheduleSettings, error) {
	if s == nil || s.settingRepo == nil || groupID <= 0 { return GroupRateScheduleSettings{}, ErrGroupFeatureUnavailable }
	if expectedVersion < 0 { return GroupRateScheduleSettings{}, ErrGroupFeatureConflict }
	config.Rules = append([]rateschedule.Rule{}, config.Rules...)
	if config.Timezone == "" { config.Timezone = timezone.Location().String() }
	if _, err := rateschedule.Compile(config, config.Timezone); err != nil {
		return GroupRateScheduleSettings{}, infraerrors.BadRequest("invalid_rate_schedule", err.Error())
	}
	current, raw, exists, err := s.loadGroupRateSchedule(ctx, groupID)
	if err != nil { return GroupRateScheduleSettings{}, err }
	if (!exists && expectedVersion != 0) || (exists && current.Version != expectedVersion) || expectedVersion == int64(^uint64(0)>>1) {
		return GroupRateScheduleSettings{}, ErrGroupFeatureConflict
	}
	cas, ok := s.settingRepo.(GroupFeatureSettingsStore)
	if !ok { return GroupRateScheduleSettings{}, ErrGroupFeatureUnavailable }
	next := GroupRateScheduleSettings{Version: expectedVersion + 1, Config: config, UpdatedAt: time.Now().UTC()}
	encoded, err := json.Marshal(next)
	if err != nil { return GroupRateScheduleSettings{}, ErrGroupFeatureUnavailable }
	if err := cas.CompareAndSwapGroupFeature(ctx, groupID, groupRateScheduleKey(groupID), raw, string(encoded)); err != nil { return GroupRateScheduleSettings{}, err }
	return next, nil
}

// PrepareGroupRateSchedule only reads configuration and creates request-owned
// values. It never changes the cached group, account selection, balance or quota.
// A fresh read means another instance's successful save affects the next request
// without relying on a stale distributed auth-cache entry.
func (s *SettingService) PrepareGroupRateSchedule(ctx context.Context, apiKey *APIKey, acceptedAt time.Time, live bool) (*APIKey, error) {
	if apiKey == nil || apiKey.Group == nil { return apiKey, nil }
	stored, _, exists, err := s.loadGroupRateSchedule(ctx, apiKey.Group.ID)
	if err != nil { return nil, err }
	if !exists { return apiKey, nil }
	compiled, err := rateschedule.Compile(stored.Config, timezone.Location().String())
	if err != nil { return nil, ErrGroupFeatureUnavailable }
	if acceptedAt.IsZero() { return nil, ErrGroupFeatureUnavailable }
	// Check the group-default preview now. User overrides retain their existing
	// resolution order; the runtime stores only the additional schedule factor.
	if _, err := compiled.Resolve(acceptedAt, apiKey.Group.RateMultiplier); err != nil { return nil, ErrGroupFeatureUnavailable }
	keyCopy, groupCopy := *apiKey, *apiKey.Group
	groupCopy.RateScheduleRuntime = &GroupRateScheduleRuntime{Schedule: compiled, AcceptedAt: acceptedAt, Live: live, Version: stored.Version}
	keyCopy.Group = &groupCopy
	return &keyCopy, nil
}

func (s *SettingService) loadGroupRateSchedule(ctx context.Context, groupID int64) (GroupRateScheduleSettings, string, bool, error) {
	if s == nil || s.settingRepo == nil { return GroupRateScheduleSettings{}, "", false, ErrGroupFeatureUnavailable }
	raw, err := s.settingRepo.GetValue(ctx, groupRateScheduleKey(groupID))
	if errors.Is(err, ErrSettingNotFound) { return GroupRateScheduleSettings{}, "", false, nil }
	if err != nil { return GroupRateScheduleSettings{}, "", false, ErrGroupFeatureUnavailable }
	var stored GroupRateScheduleSettings
	if strings.TrimSpace(raw) == "" || json.Unmarshal([]byte(raw), &stored) != nil || stored.Version < 1 {
		return GroupRateScheduleSettings{}, "", false, ErrGroupFeatureUnavailable
	}
	if _, err := rateschedule.Compile(stored.Config, timezone.Location().String()); err != nil { return GroupRateScheduleSettings{}, "", false, ErrGroupFeatureUnavailable }
	return stored, raw, true, nil
}

func legacyGroupRateSchedule(group *Group) rateschedule.Config {
	config := rateschedule.Config{Timezone: timezone.Location().String(), Rules: []rateschedule.Rule{}}
	if group == nil || !group.IsSubscriptionType() { return config }
	start, validStart := parseMinutes(group.PeakStart)
	end, validEnd := parseMinutes(group.PeakEnd)
	if validStart && validEnd && start < end {
		config.Enabled = group.PeakRateEnabled
		config.Rules = []rateschedule.Rule{{ID: "legacy", Enabled: true, Start: group.PeakStart, End: group.PeakEnd, Multiplier: group.PeakRateMultiplier}}
		if _, err := rateschedule.Compile(config, config.Timezone); err != nil { config.Enabled, config.Rules = false, []rateschedule.Rule{} }
	}
	return config
}

func groupRateScheduleKey(groupID int64) string { return GroupRateScheduleKeyPrefix + strconv.FormatInt(groupID, 10) }

func (r *GroupRateScheduleRuntime) FactorAt(turnAt time.Time) float64 {
	if r == nil || r.Schedule == nil { return 1 }
	at := r.AcceptedAt
	if r.Live { at = turnAt }
	snapshot, err := r.Schedule.Resolve(at, 1)
	if err != nil {
		// Only trusted, precompiled runtime values reach here. Do not guess a
		// discounted price on programming errors; callers receive a nonfinite
		// sentinel rather than silently treating a malformed schedule as free.
		panic(fmt.Errorf("validated group rate snapshot failed: %w", err))
	}
	return snapshot.ScheduleMultiplier
}
