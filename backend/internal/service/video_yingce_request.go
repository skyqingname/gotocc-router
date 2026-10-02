package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/videoprotocol"
	yp "github.com/LuckyKuang/sub2api-plus/internal/pkg/yingceprotocol"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"io"
	"mime"
	"mime/multipart"
	"strings"
)

// Accept the gateway's canonical video fields and Yingce's equivalent names.
func normalizeYingceVideoParameters(body []byte) ([]byte, error) {
	aliases := [][2]string{{"duration", "seconds"}, {"aspectRatio", "aspect_ratio"}, {"generateAudio", "generate_audio"}, {"providerOptions", "provider_options"}, {"image_urls", "images"}, {"reference_images", "images"}, {"video_urls", "videos"}, {"reference_videos", "videos"}, {"audio_urls", "audios"}, {"reference_audios", "audios"}}
	var err error
	for _, pair := range aliases {
		v := gjson.GetBytes(body, pair[0])
		if v.Exists() && !gjson.GetBytes(body, pair[1]).Exists() {
			body, err = sjson.SetBytes(body, pair[1], v.Value())
			if err != nil {
				return nil, err
			}
		}
	}
	seconds := gjson.GetBytes(body, "seconds")
	if seconds.Type == gjson.String {
		var number json.Number
		if err = json.Unmarshal([]byte(seconds.String()), &number); err != nil {
			return nil, fmt.Errorf("invalid video seconds")
		}
		body, err = sjson.SetBytes(body, "seconds", number)
		if err != nil {
			return nil, err
		}
	}
	return body, nil
}
func prepareYingceGeneration(config *videoprotocol.Config, parameters, original []byte, contentType string) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(parameters, &raw); err != nil {
		return err
	}
	gen := yp.GenerationRequest{Capability: yp.CapabilityVideo, Model: config.UpstreamModel, Prompt: gjson.GetBytes(parameters, "prompt").String(), Duration: int(gjson.GetBytes(parameters, "seconds").Int()), AspectRatio: gjson.GetBytes(parameters, "aspect_ratio").String(), Resolution: gjson.GetBytes(parameters, "resolution").String(), Quality: gjson.GetBytes(parameters, "quality").String(), GenerateAudio: gjson.GetBytes(parameters, "generate_audio").Bool(), Watermark: gjson.GetBytes(parameters, "watermark").Bool(), Operation: gjson.GetBytes(parameters, "operation").String(), ProviderOptions: map[string]map[string]any{}, Extra: map[string]any{}}
	if v := raw["output"]; len(v) > 0 {
		if err := json.Unmarshal(v, &gen.Output); err != nil {
			return err
		}
	}
	if v := raw["extra"]; len(v) > 0 {
		if err := json.Unmarshal(v, &gen.Extra); err != nil {
			return err
		}
	}
	for provider, opts := range config.ProviderOptions {
		gen.ProviderOptions[provider] = map[string]any{}
		for k, v := range opts {
			gen.ProviderOptions[provider][k] = v
		}
	}
	if value := raw["provider_options"]; len(value) > 0 {
		var opts map[string]map[string]any
		if err := json.Unmarshal(value, &opts); err != nil {
			return err
		}
		for id, fields := range opts {
			if gen.ProviderOptions[id] == nil {
				gen.ProviderOptions[id] = map[string]any{}
			}
			for k, v := range fields {
				gen.ProviderOptions[id][k] = v
			}
		}
	}
	known := map[string]bool{}
	for _, key := range []string{"model", "prompt", "seconds", "duration", "aspect_ratio", "aspectRatio", "resolution", "quality", "generate_audio", "generateAudio", "watermark", "images", "image_urls", "reference_images", "input_reference", "videos", "video_urls", "reference_videos", "audios", "audio_urls", "reference_audios", "provider_options", "providerOptions", "output", "extra", "operation"} {
		known[key] = true
	}
	for key, value := range raw {
		if !known[key] {
			var v any
			if err := json.Unmarshal(value, &v); err != nil {
				return err
			}
			gen.Extra[key] = v
		}
	}
	var err error
	gen.Images, err = yingceMediaReferences(raw["images"], "image")
	if err != nil {
		return err
	}
	gen.Videos, err = yingceMediaReferences(raw["videos"], "video")
	if err != nil {
		return err
	}
	gen.Audios, err = yingceMediaReferences(raw["audios"], "audio")
	if err != nil {
		return err
	}
	if v := gjson.GetBytes(parameters, "input_reference"); v.Type == gjson.String {
		gen.Images = append(gen.Images, yp.MediaReference{URL: v.String(), Kind: "image"})
	}
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return err
	}
	if mediaType == "multipart/form-data" {
		reader := multipart.NewReader(bytes.NewReader(original), params["boundary"])
		for {
			part, e := reader.NextPart()
			if e == io.EOF {
				break
			}
			if e != nil {
				return e
			}
			if part.FileName() == "" {
				part.Close()
				continue
			}
			data, e := io.ReadAll(part)
			part.Close()
			if e != nil {
				return e
			}
			mimeType := part.Header.Get("Content-Type")
			reference := yp.MediaReference{DataURL: "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data), Name: part.FileName(), MIMEType: mimeType, Kind: "image"}
			switch {
			case strings.HasPrefix(mimeType, "video/"):
				reference.Kind = "video"
				gen.Videos = append(gen.Videos, reference)
			case strings.HasPrefix(mimeType, "audio/"):
				reference.Kind = "audio"
				gen.Audios = append(gen.Audios, reference)
			default:
				gen.Images = append(gen.Images, reference)
			}
		}
	}
	adapter, err := config.Adapter()
	if err != nil {
		return err
	}
	if _, err = adapter.BuildCreate(context.Background(), yp.RequestContext{Request: gen}); err != nil {
		return err
	}
	config.CreateRequest = &gen
	// Poll/result templates receive model, options and output settings, never a copy
	// of uploaded media or the user's prompt in the persisted task configuration.
	poll := gen
	poll.Prompt = ""
	poll.Images = nil
	poll.Videos = nil
	poll.Audios = nil
	poll.Inputs = nil
	config.PollRequest = &poll
	return nil
}
func yingceMediaReferences(raw json.RawMessage, kind string) ([]yp.MediaReference, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var values []json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, fmt.Errorf("%s references must be an array", kind)
	}
	out := make([]yp.MediaReference, 0, len(values))
	for i, v := range values {
		var ref yp.MediaReference
		if len(v) > 0 && v[0] == '"' {
			if err := json.Unmarshal(v, &ref.URL); err != nil {
				return nil, err
			}
		} else if err := json.Unmarshal(v, &ref); err != nil {
			return nil, err
		}
		if strings.HasPrefix(ref.URL, "data:") {
			ref.DataURL = ref.URL
			ref.URL = ""
		}
		ref.Kind = kind
		ref.Order = i
		out = append(out, ref)
	}
	return out, nil
}
