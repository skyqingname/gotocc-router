package service

import (
	_ "embed"
	"encoding/json"
	"strings"
)

//go:embed video_yingce_url_gen.json
var yingceURLJSON []byte
var yingceURLDefaults struct {
	Prefix   string   `json:"default_prefix"`
	Prefixes []string `json:"prefixes"`
}

func init() {
	if err := json.Unmarshal(yingceURLJSON, &yingceURLDefaults); err != nil {
		panic(err)
	}
}
func yingceAPIURLWithDefaultPrefix(baseURL string, path string, defaultPrefix string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	requestPath := strings.TrimSpace(path)
	if requestPath == "" {
		return base
	}
	if !strings.HasPrefix(requestPath, "/") {
		requestPath = "/" + requestPath
	}

	requestPrefix := yingceRequestAPIPathPrefix(requestPath)
	basePrefix := yingceBaseAPIPathPrefix(base)
	if requestPrefix != "" {
		if basePrefix == requestPrefix {
			return base + strings.TrimPrefix(requestPath, requestPrefix)
		}
		// 请求路径显式版本优先于 baseURL 残留版本，例如 base=/v1、path=/v2/... 时必须切到 /v2。
		return strings.TrimSuffix(base, basePrefix) + requestPath
	}
	if basePrefix != "" {
		return base + requestPath
	}
	return base + defaultPrefix + requestPath
}

func yingceRequestAPIPathPrefix(value string) string {
	lower := strings.ToLower(value)
	for _, prefix := range yingceURLDefaults.Prefixes {
		if lower == prefix || strings.HasPrefix(lower, prefix+"/") || strings.HasPrefix(lower, prefix+"?") || strings.HasPrefix(lower, prefix+"#") {
			return prefix
		}
	}
	return ""
}

func yingceBaseAPIPathPrefix(value string) string {
	lower := strings.ToLower(strings.TrimRight(value, "/"))
	for _, prefix := range yingceURLDefaults.Prefixes {
		if lower == prefix || strings.HasSuffix(lower, prefix) {
			return prefix
		}
	}
	return ""
}
