package service

import (
	"context"
	"strings"

	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
)

// validatePasswordResetSettingsUpdate checks the effective values after a
// partial update, including the existing configuration-file URL alternative.
func (s *SettingService) validatePasswordResetSettingsUpdate(ctx context.Context, updates map[string]string) error {
	keys := []string{SettingKeyEmailVerifyEnabled, SettingKeyPasswordResetEnabled, SettingKeyFrontendURL}
	touched := false
	for _, key := range keys {
		if _, ok := updates[key]; ok {
			touched = true
			break
		}
	}
	if !touched {
		return nil
	}
	values, err := s.settingRepo.GetMultiple(ctx, keys)
	if err != nil {
		return err
	}
	for _, key := range keys {
		if value, ok := updates[key]; ok {
			values[key] = value
		}
	}
	if values[SettingKeyEmailVerifyEnabled] != "true" || values[SettingKeyPasswordResetEnabled] != "true" {
		return nil
	}
	frontendURL := strings.TrimSpace(values[SettingKeyFrontendURL])
	if frontendURL == "" && s.cfg != nil {
		frontendURL = strings.TrimSpace(s.cfg.Server.FrontendURL)
	}
	if frontendURL == "" {
		return infraerrors.BadRequest("PASSWORD_RESET_FRONTEND_URL_REQUIRED", "Configure a frontend URL before enabling password reset")
	}
	return nil
}
