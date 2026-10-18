package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// ImageStorageRoutingConfig is an explicit opt-in to local writes with legacy
// S3 reads. Its JSON file lives outside the binary and is retained on upgrades.
type ImageStorageRoutingConfig struct {
	WriteBackend   string `json:"write_backend"`
	LocalDirectory string `json:"local_directory"`
	PublicBaseURL  string `json:"public_base_url"`
	URLExpiryHours int    `json:"url_expiry_hours"`
}

func loadImageStorageRouting(filename string) (*ImageStorageRoutingConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read image storage routing: %w", err)
	}
	var routing ImageStorageRoutingConfig
	if err := json.Unmarshal(data, &routing); err != nil {
		return nil, fmt.Errorf("parse image storage routing: %w", err)
	}
	if routing.WriteBackend != "local" && routing.WriteBackend != "s3" {
		return nil, fmt.Errorf("image storage write_backend must be local or s3")
	}
	if routing.LocalDirectory == "" || routing.URLExpiryHours <= 0 {
		return nil, fmt.Errorf("image storage local_directory and positive url_expiry_hours are required")
	}
	if !filepath.IsAbs(routing.LocalDirectory) {
		routing.LocalDirectory = filepath.Join(filepath.Dir(filename), routing.LocalDirectory)
	}
	routing.PublicBaseURL = strings.TrimRight(routing.PublicBaseURL, "/")
	base, err := url.Parse(routing.PublicBaseURL)
	if err != nil || base == nil || base.Host == "" || (base.Scheme != "https" && base.Scheme != "http") || base.RawQuery != "" || base.Fragment != "" || base.User != nil {
		return nil, fmt.Errorf("image storage public_base_url must be an HTTP(S) base URL without credentials, query or fragment")
	}
	return &routing, nil
}
