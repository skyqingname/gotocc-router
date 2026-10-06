package videoprotocol

import (
	"fmt"
	"strings"
)

// ResolutionPlaceholder in an upstream model name is replaced by the request's
// resolution, so one public model can serve upstream models split by resolution.
const ResolutionPlaceholder = "{resolution}"

// Capabilities is a model's public contract: the docs page and model plaza show
// it, and the gateway enforces it before billing and the upstream request.
// Field names follow the unified /v1/videos catalogue of the reference upstream.
type Capabilities struct {
	MinSeconds         int      `json:"min_seconds,omitempty"`
	MaxSeconds         int      `json:"max_seconds,omitempty"`
	FixedSeconds       []int    `json:"fixed_seconds,omitempty"`
	Resolutions        []string `json:"resolutions,omitempty"`
	AspectRatios       []string `json:"aspect_ratios,omitempty"`
	ReferenceImages    bool     `json:"reference_images"`
	ReferenceVideos    bool     `json:"reference_videos"`
	ReferenceAudios    bool     `json:"reference_audios"`
	MaxReferenceImages int      `json:"max_reference_images,omitempty"`
	MaxReferenceVideos int      `json:"max_reference_videos,omitempty"`
	MaxReferenceAudios int      `json:"max_reference_audios,omitempty"`
	MaxReferenceTotal  int      `json:"max_reference_total,omitempty"`
	AudioOutput        bool     `json:"audio_output"`
}

func (c Capabilities) Validate() error {
	if c.MinSeconds < 0 || c.MaxSeconds < 0 || c.MaxReferenceImages < 0 || c.MaxReferenceVideos < 0 || c.MaxReferenceAudios < 0 || c.MaxReferenceTotal < 0 {
		return fmt.Errorf("能力数值不能为负数")
	}
	if c.MinSeconds > 0 && c.MaxSeconds > 0 && c.MinSeconds > c.MaxSeconds {
		return fmt.Errorf("最短时长不能大于最长时长")
	}
	for _, seconds := range c.FixedSeconds {
		if seconds <= 0 {
			return fmt.Errorf("固定时长必须为正整数")
		}
	}
	seen := map[string]bool{}
	for _, resolution := range c.Resolutions {
		key := strings.ToLower(strings.TrimSpace(resolution))
		if key == "" || seen[key] {
			return fmt.Errorf("分辨率不能为空或重复")
		}
		seen[key] = true
	}
	for _, ratio := range c.AspectRatios {
		if strings.TrimSpace(ratio) == "" {
			return fmt.Errorf("画面比例不能为空")
		}
	}
	return nil
}

// DefaultSeconds is the duration used when a request omits it: the first fixed
// duration, otherwise 5 seconds kept inside the configured range.
func (c Capabilities) DefaultSeconds() int {
	if len(c.FixedSeconds) > 0 {
		return c.FixedSeconds[0]
	}
	seconds := 5
	if c.MinSeconds > 0 && seconds < c.MinSeconds {
		seconds = c.MinSeconds
	}
	if c.MaxSeconds > 0 && seconds > c.MaxSeconds {
		seconds = c.MaxSeconds
	}
	return seconds
}

// UpstreamModels lists every upstream model name this configuration can send,
// expanding the resolution placeholder over the configured resolutions.
func (c Config) UpstreamModels() []string {
	if !strings.Contains(c.UpstreamModel, ResolutionPlaceholder) {
		return []string{c.UpstreamModel}
	}
	models := []string{}
	if c.Capabilities != nil {
		for _, resolution := range c.Capabilities.Resolutions {
			models = append(models, strings.ReplaceAll(c.UpstreamModel, ResolutionPlaceholder, resolution))
		}
	}
	return models
}
