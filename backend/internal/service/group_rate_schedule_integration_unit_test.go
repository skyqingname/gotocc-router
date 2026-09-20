//go:build unit

package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/rateschedule"
)

// These are service-level unit tests, not PostgreSQL transaction or end-to-end
// billing tests. Embed the unchanged repository contract for unused methods.
type groupFeatureMemoryStore struct {
	SettingRepository
	mu sync.Mutex
	values map[string]string
}

func (s *groupFeatureMemoryStore) GetValue(_ context.Context, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, exists := s.values[key]
	if !exists { return "", ErrSettingNotFound }
	return value, nil
}

func (s *groupFeatureMemoryStore) CompareAndSwapGroupFeature(_ context.Context, _ int64, key, expected, replacement string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.values[key] != expected { return ErrGroupFeatureConflict }
	s.values[key] = replacement
	return nil
}

func testGroupSchedule() rateschedule.Config {
	return rateschedule.Config{Enabled: true, Timezone: "UTC", Rules: []rateschedule.Rule{
		{ID: "night", Enabled: true, Start: "22:00", End: "06:00", Multiplier: 0.5},
	}}
}

func TestGroupRateScheduleSaveHydrateAndFreeze(t *testing.T) {
	ctx := context.Background()
	store := &groupFeatureMemoryStore{values: map[string]string{}}
	settings := &SettingService{settingRepo: store}
	group := &Group{ID: 7, RateMultiplier: 0.4, SubscriptionType: "standard"}
	key := &APIKey{ID: 9, Group: group}
	at := time.Date(2026, 9, 20, 23, 0, 0, 0, time.UTC)
	unchanged, err := settings.PrepareGroupRateSchedule(ctx, key, at, false)
	if err != nil || unchanged != key { t.Fatalf("missing config changed existing request: %v", err) }
	saved, err := settings.SaveGroupRateSchedule(ctx, group.ID, 0, testGroupSchedule())
	if err != nil || saved.Version != 1 { t.Fatalf("save: %+v, %v", saved, err) }
	prepared, err := settings.PrepareGroupRateSchedule(ctx, key, at, false)
	if err != nil { t.Fatal(err) }
	if prepared == key || prepared.Group == group || group.RateScheduleRuntime != nil { t.Fatal("shared authentication object was mutated") }
	later := at.Add(9 * time.Hour)
	if got := prepared.Group.PeakMultiplierAt(later); got != 0.5 { t.Fatalf("HTTP request repriced across boundary: %v", got) }
	text, image := computePeakAwareMultipliers(prepared, 0.25, later)
	if text != 0.125 || image != resolveImageRateMultiplier(key, 0.25) { t.Fatalf("user base/media separation: %v, %v", text, image) }

	changed := testGroupSchedule()
	changed.Rules[0].Multiplier = 2
	if _, err := settings.SaveGroupRateSchedule(ctx, group.ID, 1, changed); err != nil { t.Fatal(err) }
	fresh, err := settings.PrepareGroupRateSchedule(ctx, key, at, false)
	if err != nil { t.Fatal(err) }
	if fresh.Group.PeakMultiplierAt(at) != 2 || prepared.Group.PeakMultiplierAt(at) != 0.5 { t.Fatal("new request missed configuration, or old snapshot changed") }
	if _, err := settings.SaveGroupRateSchedule(ctx, group.ID, 1, changed); !errors.Is(err, ErrGroupFeatureConflict) { t.Fatalf("stale version accepted: %v", err) }
}

func TestGroupRateScheduleSupersedesLegacyAndKeepsDisabledRules(t *testing.T) {
	settings := &SettingService{settingRepo: &groupFeatureMemoryStore{values: map[string]string{}}}
	group := &Group{ID: 7, SubscriptionType: SubscriptionTypeSubscription, RateMultiplier: 0.4,
		PeakRateEnabled: true, PeakStart: "22:00", PeakEnd: "23:59", PeakRateMultiplier: 3}
	config := testGroupSchedule()
	config.Enabled = false
	ctx := context.Background()
	if _, err := settings.SaveGroupRateSchedule(ctx, group.ID, 0, config); err != nil { t.Fatal(err) }
	view, err := settings.GetGroupRateSchedule(ctx, group)
	if err != nil || view.Source != "schedule" || len(view.Config.Rules) != 1 { t.Fatalf("disabled rules erased: %+v, %v", view, err) }
	at := time.Date(2026, 9, 20, 23, 0, 0, 0, time.UTC)
	key, err := settings.PrepareGroupRateSchedule(ctx, &APIKey{Group: group}, at, false)
	if err != nil { t.Fatal(err) }
	if key.Group.PeakMultiplierAt(at) != 1 { t.Fatal("legacy factor stacked with disabled new schedule") }
}

func TestGroupRateScheduleConcurrentCASAndInvalidStorage(t *testing.T) {
	store := &groupFeatureMemoryStore{values: map[string]string{}}
	settings := &SettingService{settingRepo: store}
	ctx := context.Background()
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := settings.SaveGroupRateSchedule(ctx, 7, 0, testGroupSchedule()); results <- err }()
	}
	wg.Wait()
	close(results)
	success := 0
	for err := range results { if err == nil { success++ } else if !errors.Is(err, ErrGroupFeatureConflict) { t.Fatal(err) } }
	if success != 1 { t.Fatalf("expected one successful version-zero write, got %d", success) }
	store.mu.Lock()
	store.values[groupRateScheduleKey(7)] = `{"version":1,"config":{"timezone":"Invalid/Zone"}}`
	store.mu.Unlock()
	if _, err := settings.GetGroupRateSchedule(ctx, &Group{ID: 7}); err == nil { t.Fatal("corrupt configuration silently treated as missing") }
}
