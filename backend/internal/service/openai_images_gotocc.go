package service

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

type openAIGeminiImageData struct {
	B64JSON  string `json:"b64_json"`
	MimeType string `json:"mime_type,omitempty"`
}

type openAIGeminiImageResponse struct {
	Created int64                   `json:"created,omitempty"`
	Data    []openAIGeminiImageData `json:"data"`
}

func normalizeOpenAIGeminiImageResponse(body []byte) ([]byte, bool) {
	if len(body) == 0 || !gjson.ValidBytes(body) || gjson.GetBytes(body, "data").IsArray() {
		return body, false
	}

	root := gjson.ParseBytes(body)
	candidates := root.Get("candidates")
	createTime := root.Get("createTime").String()
	if !candidates.IsArray() {
		candidates = root.Get("response.candidates")
		createTime = root.Get("response.createTime").String()
	}
	if !candidates.IsArray() {
		return body, false
	}

	images := make([]openAIGeminiImageData, 0, 1)
	candidates.ForEach(func(_, candidate gjson.Result) bool {
		candidate.Get("content.parts").ForEach(func(_, part gjson.Result) bool {
			inlineData := part.Get("inlineData")
			if !inlineData.Exists() {
				inlineData = part.Get("inline_data")
			}
			mimeType := strings.TrimSpace(inlineData.Get("mimeType").String())
			if mimeType == "" {
				mimeType = strings.TrimSpace(inlineData.Get("mime_type").String())
			}
			data := strings.TrimSpace(inlineData.Get("data").String())
			if data != "" && strings.HasPrefix(strings.ToLower(mimeType), "image/") {
				images = append(images, openAIGeminiImageData{B64JSON: data, MimeType: mimeType})
			}
			return true
		})
		return true
	})
	if len(images) == 0 {
		return body, false
	}

	response := openAIGeminiImageResponse{Data: images}
	if parsedTime, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(createTime)); err == nil {
		response.Created = parsedTime.Unix()
	}
	normalized, err := json.Marshal(response)
	if err != nil {
		return body, false
	}
	return normalized, true
}
