package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/config"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/pagination"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type reusableInvitationValidationSettingRepo struct {
	values map[string]string
}

func (s *reusableInvitationValidationSettingRepo) Get(context.Context, string) (*service.Setting, error) {
	panic("unexpected Get call")
}
func (s *reusableInvitationValidationSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", service.ErrSettingNotFound
}
func (s *reusableInvitationValidationSettingRepo) Set(context.Context, string, string) error {
	panic("unexpected Set call")
}
func (s *reusableInvitationValidationSettingRepo) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}
func (s *reusableInvitationValidationSettingRepo) SetMultiple(context.Context, map[string]string) error {
	panic("unexpected SetMultiple call")
}
func (s *reusableInvitationValidationSettingRepo) GetAll(context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}
func (s *reusableInvitationValidationSettingRepo) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

type reusableInvitationValidationRepo struct {
	codes map[string]*service.ReusableInvitationCode
}

type reusableInvitationValidationEmailCache struct {
	service.EmailCache
}

func (*reusableInvitationValidationEmailCache) GetVerificationCode(context.Context, string) (*service.VerificationCodeData, error) {
	return nil, nil
}

func (s *reusableInvitationValidationRepo) Create(context.Context, *service.ReusableInvitationCode) error {
	panic("unexpected Create call")
}
func (s *reusableInvitationValidationRepo) GetByID(context.Context, int64) (*service.ReusableInvitationCode, error) {
	panic("unexpected GetByID call")
}
func (s *reusableInvitationValidationRepo) GetByCode(_ context.Context, code string) (*service.ReusableInvitationCode, error) {
	if got, ok := s.codes[code]; ok {
		return got, nil
	}
	return nil, service.ErrReusableInvitationCodeNotFound
}
func (s *reusableInvitationValidationRepo) GetUsableByCode(_ context.Context, code string) (*service.ReusableInvitationCode, error) {
	got, ok := s.codes[code]
	if !ok || !got.IsUsableAt(time.Now()) {
		return nil, service.ErrReusableInvitationCodeInvalid
	}
	return got, nil
}
func (s *reusableInvitationValidationRepo) List(context.Context, pagination.PaginationParams) ([]service.ReusableInvitationCode, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}
func (s *reusableInvitationValidationRepo) Disable(context.Context, int64) (*service.ReusableInvitationCode, error) {
	panic("unexpected Disable call")
}
func (s *reusableInvitationValidationRepo) Use(context.Context, int64, int64, string, string) error {
	panic("unexpected Use call")
}
func (s *reusableInvitationValidationRepo) Release(context.Context, int64, int64) error {
	panic("unexpected Release call")
}
func (s *reusableInvitationValidationRepo) ListUsesByCodeID(context.Context, int64, int) ([]service.ReusableInvitationCodeUse, error) {
	panic("unexpected ListUsesByCodeID call")
}

func TestValidateInvitationCodeAcceptsReusableCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{}
	settingSvc := service.NewSettingService(&reusableInvitationValidationSettingRepo{values: map[string]string{
		service.SettingKeyInvitationCodeEnabled: "true",
	}}, cfg)
	repo := &reusableInvitationValidationRepo{codes: map[string]*service.ReusableInvitationCode{
		"FOREVER": {ID: 99, Code: "FOREVER", Status: service.ReusableInvitationCodeStatusActive},
	}}
	h := NewAuthHandler(cfg, nil, nil, settingSvc, nil, nil, nil, nil)
	h.SetReusableInvitationCodeRepository(repo)

	router := gin.New()
	router.POST("/validate", h.ValidateInvitationCode)
	req := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewBufferString(`{"code":"FOREVER"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Data ValidateInvitationCodeResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.True(t, body.Data.Valid)
	require.Empty(t, body.Data.ErrorCode)
}

func TestReusableInvitationCodeDoublePathReachesEmailVerification(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{}
	settingRepo := &reusableInvitationValidationSettingRepo{values: map[string]string{
		service.SettingKeyRegistrationEnabled:   "true",
		service.SettingKeyInvitationCodeEnabled: "true",
		service.SettingKeyEmailVerifyEnabled:    "true",
	}}
	settingSvc := service.NewSettingService(settingRepo, cfg)
	repo := &reusableInvitationValidationRepo{codes: map[string]*service.ReusableInvitationCode{
		"FOREVER": {ID: 99, Code: "FOREVER", Status: service.ReusableInvitationCodeStatusActive},
	}}

	authService := service.NewAuthService(
		nil,
		nil,
		nil,
		nil,
		cfg,
		settingSvc,
		service.NewEmailService(settingRepo, &reusableInvitationValidationEmailCache{}),
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	authService.SetReusableInvitationCodeRepository(repo)
	h := NewAuthHandler(cfg, authService, nil, settingSvc, nil, nil, nil, nil)
	h.SetReusableInvitationCodeRepository(repo)

	router := gin.New()
	router.POST("/validate", h.ValidateInvitationCode)
	req := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewBufferString(`{"code":"FOREVER"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Data ValidateInvitationCodeResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.True(t, body.Data.Valid)

	_, _, err := authService.RegisterWithVerification(
		context.Background(),
		"probe@example.com",
		"password",
		"definitely-invalid",
		"",
		"FOREVER",
		"",
	)
	require.ErrorIs(t, err, service.ErrInvalidVerifyCode)
	require.NotErrorIs(t, err, service.ErrInvitationCodeInvalid)
}
