//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/LuckyKuang/sub2api-plus/internal/config"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type providerReusableInvitationRepoStub struct{}

func (*providerReusableInvitationRepoStub) Create(context.Context, *ReusableInvitationCode) error {
	return nil
}
func (*providerReusableInvitationRepoStub) GetByID(context.Context, int64) (*ReusableInvitationCode, error) {
	return nil, ErrReusableInvitationCodeNotFound
}
func (*providerReusableInvitationRepoStub) GetByCode(context.Context, string) (*ReusableInvitationCode, error) {
	return nil, ErrReusableInvitationCodeNotFound
}
func (*providerReusableInvitationRepoStub) GetUsableByCode(context.Context, string) (*ReusableInvitationCode, error) {
	return nil, ErrReusableInvitationCodeNotFound
}
func (*providerReusableInvitationRepoStub) List(context.Context, pagination.PaginationParams) ([]ReusableInvitationCode, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (*providerReusableInvitationRepoStub) Disable(context.Context, int64) (*ReusableInvitationCode, error) {
	return nil, ErrReusableInvitationCodeNotFound
}
func (*providerReusableInvitationRepoStub) Use(context.Context, int64, int64, string, string) error {
	return nil
}
func (*providerReusableInvitationRepoStub) Release(context.Context, int64, int64) error {
	return nil
}
func (*providerReusableInvitationRepoStub) ListUsesByCodeID(context.Context, int64, int) ([]ReusableInvitationCodeUse, error) {
	return nil, nil
}

func TestProvideAuthServiceWiresReusableInvitationRepository(t *testing.T) {
	reusableRepo := &providerReusableInvitationRepoStub{}

	authService := ProvideAuthService(
		nil, nil, nil, nil, &config.Config{}, nil, nil, nil, nil, nil,
		reusableRepo,
		nil, nil, nil, nil, nil,
	)

	require.Same(t, reusableRepo, authService.reusableInvitationRepo)
}
