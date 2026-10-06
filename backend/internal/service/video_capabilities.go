package service

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"slices"
	"strconv"
	"strings"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/videoprotocol"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// Unified /v1/videos request fields: the documented name comes first, followed
// by the accepted aliases.
var (
	videoSecondsFields     = []string{"seconds", "duration", "duration_seconds"}
	videoAspectRatioFields = []string{"aspect_ratio", "aspectRatio", "ratio"}
	videoAudioOutputFields = []string{"generate_audio", "generateAudio"}
	videoImageFields       = []string{"reference_images", "images", "image_urls", "input_reference", "first_frame_image", "last_frame_image"}
	videoVideoFields       = []string{"reference_videos", "videos", "video_urls"}
	videoAudioFields       = []string{"reference_audios", "audios", "audio_urls"}
)

// applyVideoCapabilities fills the duration, resolution and aspect ratio a
// request omits, rejects values the model does not support and resolves the
// upstream model's resolution placeholder. It returns the fields it wrote so a
// multipart request can carry them upstream.
func applyVideoCapabilities(config *videoprotocol.Config, parameters, body []byte, contentType string) ([]byte, []string, error) {
	written := []string{}
	set := func(field string, value any) error {
		var err error
		parameters, err = sjson.SetBytes(parameters, field, value)
		written = append(written, field)
		return err
	}
	caps := config.Capabilities

	if !gjson.GetBytes(parameters, "seconds").Exists() {
		if alias := firstVideoField(parameters, videoSecondsFields); alias.Exists() {
			if err := set("seconds", alias.Value()); err != nil {
				return nil, nil, err
			}
		} else if caps != nil {
			if err := set("seconds", caps.DefaultSeconds()); err != nil {
				return nil, nil, err
			}
		}
	}
	if caps == nil {
		return parameters, written, nil
	}

	seconds, err := videoRequestSeconds(gjson.GetBytes(parameters, "seconds"))
	if err != nil {
		return nil, nil, err
	}
	if len(caps.FixedSeconds) > 0 && !slices.Contains(caps.FixedSeconds, seconds) {
		return nil, nil, fmt.Errorf("该模型时长只支持 %s 秒", joinVideoSeconds(caps.FixedSeconds))
	}
	if len(caps.FixedSeconds) == 0 && ((caps.MinSeconds > 0 && seconds < caps.MinSeconds) || (caps.MaxSeconds > 0 && seconds > caps.MaxSeconds)) {
		return nil, nil, fmt.Errorf("该模型时长需在 %s 之间", videoSecondsRange(caps))
	}

	resolution, requested := openAIVideoResolutionFromBody(parameters)
	// size also carries aspect ratios such as 16:9 for some clients; only WxH names a resolution.
	if requested && !firstVideoField(parameters, []string{"resolution", "resolution_name", "metadata.resolution"}).Exists() && !strings.Contains(resolution, "x") && !strings.HasSuffix(resolution, "p") {
		requested = false
	}
	if len(caps.Resolutions) > 0 {
		matched := caps.Resolutions[0]
		if requested {
			index := slices.IndexFunc(caps.Resolutions, func(allowed string) bool { return videoResolutionKey(allowed) == videoResolutionKey(resolution) })
			if index < 0 {
				return nil, nil, fmt.Errorf("该模型分辨率只支持 %s", strings.Join(caps.Resolutions, " / "))
			}
			matched = caps.Resolutions[index]
		}
		resolution = matched
		if gjson.GetBytes(parameters, "resolution").String() != resolution {
			if err := set("resolution", resolution); err != nil {
				return nil, nil, err
			}
		}
	}

	if len(caps.AspectRatios) > 0 {
		ratio := strings.TrimSpace(firstVideoField(parameters, videoAspectRatioFields).String())
		if ratio == "" {
			ratio = caps.AspectRatios[0]
		}
		if !slices.Contains(caps.AspectRatios, ratio) {
			return nil, nil, fmt.Errorf("该模型画面比例只支持 %s", strings.Join(caps.AspectRatios, " / "))
		}
		if gjson.GetBytes(parameters, "aspect_ratio").String() != ratio {
			if err := set("aspect_ratio", ratio); err != nil {
				return nil, nil, err
			}
		}
	}

	if !caps.AudioOutput && firstVideoField(parameters, videoAudioOutputFields).Bool() {
		return nil, nil, fmt.Errorf("该模型不支持生成音频，请去掉 generate_audio")
	}

	uploadedImages, uploadedVideos, uploadedAudios, err := countVideoUploads(body, contentType)
	if err != nil {
		return nil, nil, err
	}
	images := countVideoReferences(parameters, videoImageFields) + uploadedImages
	videos := countVideoReferences(parameters, videoVideoFields) + uploadedVideos
	audios := countVideoReferences(parameters, videoAudioFields) + uploadedAudios
	for _, check := range []struct {
		label, unit string
		count, max  int
		supported   bool
	}{
		{"参考图", "张", images, caps.MaxReferenceImages, caps.ReferenceImages},
		{"参考视频", "个", videos, caps.MaxReferenceVideos, caps.ReferenceVideos},
		{"参考音频", "段", audios, caps.MaxReferenceAudios, caps.ReferenceAudios},
	} {
		if check.count > 0 && !check.supported {
			return nil, nil, fmt.Errorf("该模型不支持%s", check.label)
		}
		if check.max > 0 && check.count > check.max {
			return nil, nil, fmt.Errorf("该模型%s最多 %d %s", check.label, check.max, check.unit)
		}
	}
	if caps.MaxReferenceTotal > 0 && images+videos+audios > caps.MaxReferenceTotal {
		return nil, nil, fmt.Errorf("该模型参考素材合计最多 %d 个", caps.MaxReferenceTotal)
	}

	config.UpstreamModel = strings.ReplaceAll(config.UpstreamModel, videoprotocol.ResolutionPlaceholder, resolution)
	return parameters, written, nil
}

func firstVideoField(body []byte, fields []string) gjson.Result {
	for _, field := range fields {
		if value := gjson.GetBytes(body, field); value.Exists() && value.Type != gjson.Null {
			return value
		}
	}
	return gjson.Result{}
}

func videoRequestSeconds(value gjson.Result) (int, error) {
	seconds, err := strconv.Atoi(strings.TrimSpace(value.String()))
	if value.Type == gjson.Number && value.Float() == float64(value.Int()) {
		seconds, err = int(value.Int()), nil
	}
	if err != nil || seconds <= 0 {
		return 0, fmt.Errorf("时长 duration 必须为正整数秒")
	}
	return seconds, nil
}

func joinVideoSeconds(values []int) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = strconv.Itoa(value)
	}
	return strings.Join(parts, " / ")
}

func videoSecondsRange(caps *videoprotocol.Capabilities) string {
	switch {
	case caps.MinSeconds > 0 && caps.MaxSeconds > 0:
		return fmt.Sprintf("%d–%d 秒", caps.MinSeconds, caps.MaxSeconds)
	case caps.MinSeconds > 0:
		return fmt.Sprintf("%d 秒及以上", caps.MinSeconds)
	default:
		return fmt.Sprintf("1–%d 秒", caps.MaxSeconds)
	}
}

// videoResolutionKey compares 720p, 720P and 720 as the same tier.
func videoResolutionKey(resolution string) string {
	if canonical, known := LookupVideoBillingResolution(resolution); known {
		return canonical
	}
	return strings.ToLower(strings.TrimSpace(resolution))
}

func countVideoReferences(body []byte, fields []string) int {
	count := 0
	for _, field := range fields {
		value := gjson.GetBytes(body, field)
		switch {
		case value.IsArray():
			count += len(value.Array())
		case value.IsObject(), value.Type == gjson.String && strings.TrimSpace(value.String()) != "":
			count++
		}
	}
	return count
}

// countVideoUploads counts multipart reference files by their media type.
func countVideoUploads(body []byte, contentType string) (int, int, int, error) {
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType != "multipart/form-data" {
		return 0, 0, 0, err
	}
	images, videos, audios := 0, 0, 0
	reader := multipart.NewReader(bytes.NewReader(body), params["boundary"])
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			return images, videos, audios, nil
		}
		if err != nil {
			return 0, 0, 0, err
		}
		if part.FileName() != "" {
			switch kind := part.Header.Get("Content-Type"); {
			case strings.HasPrefix(kind, "video/"):
				videos++
			case strings.HasPrefix(kind, "audio/"):
				audios++
			default:
				images++
			}
		}
		part.Close()
	}
}
