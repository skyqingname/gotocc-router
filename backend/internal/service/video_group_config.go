package service

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/videoconfig"
)

const VideoGroupConfigKeyPrefix = "video_group_config:"

type VideoGroupConfigView struct {
	GroupID int64 `json:"group_id"`
	Version int64 `json:"version"`
	Config videoconfig.Config `json:"config"`
	UpdatedAt time.Time `json:"updated_at"`
	// Configuration management and pure adapters are available; this flag must
	// remain false until durable task routing/settlement uses their snapshots.
	ExecutionReady bool `json:"execution_ready"`
}

func (s *SettingService) GetVideoGroupConfig(ctx context.Context, groupID int64) (VideoGroupConfigView, error) {
	view, _, err := s.loadVideoGroupConfig(ctx, groupID)
	return view, err
}

func (s *SettingService) loadVideoGroupConfig(ctx context.Context, groupID int64) (VideoGroupConfigView, string, error) {
	view := VideoGroupConfigView{GroupID: groupID, Config: videoconfig.Config{Bindings: []videoconfig.Binding{}}}
	if s == nil || s.settingRepo == nil || groupID <= 0 { return view, "", ErrGroupFeatureUnavailable }
	raw, err := s.settingRepo.GetValue(ctx, VideoGroupConfigKeyPrefix+strconv.FormatInt(groupID, 10))
	if errors.Is(err, ErrSettingNotFound) { return view, "", nil }
	if err != nil { return view, "", ErrGroupFeatureUnavailable }
	if json.Unmarshal([]byte(raw), &view) != nil || view.Version < 1 || view.GroupID != groupID { return VideoGroupConfigView{}, "", ErrGroupFeatureUnavailable }
	if _, err := videoconfig.Compile(view.Config); err != nil { return VideoGroupConfigView{}, "", ErrGroupFeatureUnavailable }
	view.ExecutionReady = false
	return view, raw, nil
}

func (s *SettingService) SaveVideoGroupConfig(ctx context.Context, groupID, expectedVersion int64, config videoconfig.Config) (VideoGroupConfigView, error) {
	if _, err := videoconfig.Compile(config); err != nil { return VideoGroupConfigView{}, infraerrors.BadRequest("invalid_video_config", err.Error()) }
	current, raw, err := s.loadVideoGroupConfig(ctx, groupID)
	if err != nil { return VideoGroupConfigView{}, err }
	if expectedVersion < 0 || current.Version != expectedVersion || expectedVersion == int64(^uint64(0)>>1) { return VideoGroupConfigView{}, ErrGroupFeatureConflict }
	cas, ok := s.settingRepo.(GroupFeatureSettingsStore)
	if !ok { return VideoGroupConfigView{}, ErrGroupFeatureUnavailable }
	next := VideoGroupConfigView{GroupID: groupID, Version: expectedVersion+1, Config: config, UpdatedAt: time.Now().UTC()}
	encoded, err := json.Marshal(next)
	if err != nil { return VideoGroupConfigView{}, ErrGroupFeatureUnavailable }
	if err := cas.CompareAndSwapGroupFeature(ctx, groupID, VideoGroupConfigKeyPrefix+strconv.FormatInt(groupID, 10), raw, string(encoded)); err != nil { return VideoGroupConfigView{}, err }
	return next, nil
}

func (s *SettingService) PreviewVideoBinding(ctx context.Context, groupID int64, bindingID string, input map[string]json.RawMessage) (videoconfig.Snapshot, error) {
	view, err := s.GetVideoGroupConfig(ctx, groupID)
	if err != nil { return videoconfig.Snapshot{}, err }
	compiled, err := videoconfig.Compile(view.Config)
	if err != nil { return videoconfig.Snapshot{}, ErrGroupFeatureUnavailable }
	quote, err := compiled.Resolve(bindingID, input)
	if err != nil { return videoconfig.Snapshot{}, infraerrors.BadRequest("invalid_video_parameters", err.Error()) }
	return quote, nil
}
