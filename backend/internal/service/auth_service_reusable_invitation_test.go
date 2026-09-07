//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type reusableInvitationRepoStub struct {
	codesByCode  map[string]*ReusableInvitationCode
	getErr       error
	useErr       error
	releaseErr   error
	useCalls     []reusableInvitationUseCall
	releaseCalls []struct {
		id     int64
		userID int64
	}
}

type reusableInvitationUseCall struct {
	id         int64
	userID     int64
	email      string
	authSource string
}

func (s *reusableInvitationRepoStub) Create(context.Context, *ReusableInvitationCode) error {
	panic("unexpected Create call")
}

func (s *reusableInvitationRepoStub) GetByID(context.Context, int64) (*ReusableInvitationCode, error) {
	panic("unexpected GetByID call")
}

func (s *reusableInvitationRepoStub) GetByCode(_ context.Context, code string) (*ReusableInvitationCode, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	got, ok := s.codesByCode[code]
	if !ok {
		return nil, ErrReusableInvitationCodeNotFound
	}
	cloned := *got
	return &cloned, nil
}

func (s *reusableInvitationRepoStub) GetUsableByCode(ctx context.Context, code string) (*ReusableInvitationCode, error) {
	got, err := s.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if !got.IsUsableAt(testNow()) {
		return nil, ErrReusableInvitationCodeInvalid
	}
	return got, nil
}

func (s *reusableInvitationRepoStub) List(context.Context, pagination.PaginationParams) ([]ReusableInvitationCode, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}

func (s *reusableInvitationRepoStub) Disable(context.Context, int64) (*ReusableInvitationCode, error) {
	panic("unexpected Disable call")
}

func (s *reusableInvitationRepoStub) Use(_ context.Context, id, userID int64, email, authSource string) error {
	if s.useErr != nil {
		return s.useErr
	}
	s.useCalls = append(s.useCalls, reusableInvitationUseCall{
		id: id, userID: userID, email: email, authSource: authSource,
	})
	return nil
}

func (s *reusableInvitationRepoStub) Release(_ context.Context, id, userID int64) error {
	s.releaseCalls = append(s.releaseCalls, struct {
		id     int64
		userID int64
	}{id: id, userID: userID})
	return s.releaseErr
}

func (s *reusableInvitationRepoStub) ListUsesByCodeID(context.Context, int64, int) ([]ReusableInvitationCodeUse, error) {
	panic("unexpected ListUsesByCodeID call")
}

func testNow() time.Time {
	return time.Now()
}

func TestRegisterWithVerificationUsesReusableInvitation(t *testing.T) {
	userRepo := &userRepoStub{nextID: 101}
	redeemRepo := &redeemCodeRepoStub{}
	reusableRepo := &reusableInvitationRepoStub{codesByCode: map[string]*ReusableInvitationCode{
		"JOIN": {ID: 77, Code: "JOIN", Status: ReusableInvitationCodeStatusActive},
	}}
	authService := newAuthService(userRepo, map[string]string{
		SettingKeyRegistrationEnabled:   "true",
		SettingKeyInvitationCodeEnabled: "true",
	}, nil, nil)
	authService.redeemRepo = redeemRepo
	authService.SetReusableInvitationCodeRepository(reusableRepo)

	_, user, err := authService.RegisterWithVerification(context.Background(), "user@test.com", "password", "", "", "JOIN", "")

	require.NoError(t, err)
	require.Equal(t, int64(101), user.ID)
	require.Empty(t, redeemRepo.useCalls)
	require.Equal(t, []reusableInvitationUseCall{{
		id: 77, userID: 101, email: "user@test.com", authSource: "email",
	}}, reusableRepo.useCalls)
}

func TestRegisterWithVerificationDeletesCreatedUserWhenReusableInvitationUseFails(t *testing.T) {
	userRepo := &userRepoStub{nextID: 102}
	reusableRepo := &reusableInvitationRepoStub{
		codesByCode: map[string]*ReusableInvitationCode{
			"JOIN": {ID: 78, Code: "JOIN", Status: ReusableInvitationCodeStatusActive},
		},
		useErr: errors.New("audit write failed"),
	}
	authService := newAuthService(userRepo, map[string]string{
		SettingKeyRegistrationEnabled:   "true",
		SettingKeyInvitationCodeEnabled: "true",
	}, nil, nil)
	authService.redeemRepo = &redeemCodeRepoStub{}
	authService.SetReusableInvitationCodeRepository(reusableRepo)

	_, _, err := authService.RegisterWithVerification(context.Background(), "user@test.com", "password", "", "", "JOIN", "")

	require.ErrorIs(t, err, ErrInvitationCodeInvalid)
	require.Equal(t, []int64{102}, userRepo.deletedIDs)
}

func TestRegisterWithVerificationPrefersUsableOneTimeInvitation(t *testing.T) {
	userRepo := &userRepoStub{nextID: 103}
	redeemRepo := &redeemCodeRepoStub{codesByCode: map[string]*RedeemCode{
		"JOIN": {ID: 88, Code: "JOIN", Type: RedeemTypeInvitation, Status: StatusUnused},
	}}
	reusableRepo := &reusableInvitationRepoStub{codesByCode: map[string]*ReusableInvitationCode{
		"JOIN": {ID: 79, Code: "JOIN", Status: ReusableInvitationCodeStatusActive},
	}}
	authService := newAuthService(userRepo, map[string]string{
		SettingKeyRegistrationEnabled:   "true",
		SettingKeyInvitationCodeEnabled: "true",
	}, nil, nil)
	authService.redeemRepo = redeemRepo
	authService.SetReusableInvitationCodeRepository(reusableRepo)

	_, user, err := authService.RegisterWithVerification(context.Background(), "user@test.com", "password", "", "", "JOIN", "")

	require.NoError(t, err)
	require.NotNil(t, user)
	require.Len(t, redeemRepo.useCalls, 1)
	require.Equal(t, int64(88), redeemRepo.useCalls[0].id)
	require.Empty(t, reusableRepo.useCalls)
}

func TestOAuthFirstRegistrationUsesReusableInvitation(t *testing.T) {
	userRepo := &userRepoStub{nextID: 104}
	reusableRepo := &reusableInvitationRepoStub{codesByCode: map[string]*ReusableInvitationCode{
		"OAUTH-JOIN": {ID: 80, Code: "OAUTH-JOIN", Status: ReusableInvitationCodeStatusActive},
	}}
	authService := newAuthService(userRepo, map[string]string{
		SettingKeyRegistrationEnabled:   "true",
		SettingKeyInvitationCodeEnabled: "true",
	}, nil, nil)
	authService.refreshTokenCache = &refreshTokenCacheStub{}
	authService.redeemRepo = &redeemCodeRepoStub{}
	authService.SetReusableInvitationCodeRepository(reusableRepo)

	tokenPair, user, err := authService.LoginOrRegisterOAuthWithTokenPair(
		context.Background(), "oauth-user@test.com", "oauth_user", "OAUTH-JOIN", "", "oidc",
	)

	require.NoError(t, err)
	require.NotNil(t, tokenPair)
	require.Equal(t, int64(104), user.ID)
	require.Equal(t, []reusableInvitationUseCall{{
		id: 80, userID: 104, email: "oauth-user@test.com", authSource: "oidc",
	}}, reusableRepo.useCalls)
}

func TestOAuthFirstRegistrationDeletesCreatedUserWhenReusableInvitationUseFails(t *testing.T) {
	userRepo := &userRepoStub{nextID: 105}
	reusableRepo := &reusableInvitationRepoStub{
		codesByCode: map[string]*ReusableInvitationCode{
			"OAUTH-JOIN": {ID: 81, Code: "OAUTH-JOIN", Status: ReusableInvitationCodeStatusActive},
		},
		useErr: errors.New("audit write failed"),
	}
	authService := newAuthService(userRepo, map[string]string{
		SettingKeyRegistrationEnabled:   "true",
		SettingKeyInvitationCodeEnabled: "true",
	}, nil, nil)
	authService.refreshTokenCache = &refreshTokenCacheStub{}
	authService.redeemRepo = &redeemCodeRepoStub{}
	authService.SetReusableInvitationCodeRepository(reusableRepo)

	tokenPair, user, err := authService.LoginOrRegisterOAuthWithTokenPair(
		context.Background(), "oauth-fail@test.com", "oauth_user", "OAUTH-JOIN", "", "oidc",
	)

	require.Nil(t, tokenPair)
	require.Nil(t, user)
	require.ErrorIs(t, err, ErrInvitationCodeInvalid)
	require.Equal(t, []int64{105}, userRepo.deletedIDs)
}

func TestRollbackOAuthEmailAccountCreationReleasesReusableInvitation(t *testing.T) {
	userRepo := &userRepoStub{}
	reusableRepo := &reusableInvitationRepoStub{codesByCode: map[string]*ReusableInvitationCode{
		"OAUTH-EMAIL": {ID: 82, Code: "OAUTH-EMAIL", Status: ReusableInvitationCodeStatusActive},
	}}
	authService := newOAuthEmailFlowAuthService(
		userRepo,
		&redeemCodeRepoStub{},
		&refreshTokenCacheStub{},
		map[string]string{
			SettingKeyRegistrationEnabled:   "true",
			SettingKeyInvitationCodeEnabled: "true",
		},
		&emailCacheStub{},
		nil,
	)
	authService.SetReusableInvitationCodeRepository(reusableRepo)

	err := authService.RollbackOAuthEmailAccountCreation(context.Background(), 42, "OAUTH-EMAIL")

	require.NoError(t, err)
	require.Equal(t, []struct {
		id     int64
		userID int64
	}{{id: 82, userID: 42}}, reusableRepo.releaseCalls)
	require.Equal(t, []int64{42}, userRepo.deletedIDs)
}

func TestRollbackOAuthEmailAccountCreationStopsWhenReusableReleaseFails(t *testing.T) {
	userRepo := &userRepoStub{}
	reusableRepo := &reusableInvitationRepoStub{
		codesByCode: map[string]*ReusableInvitationCode{
			"OAUTH-EMAIL": {ID: 83, Code: "OAUTH-EMAIL", Status: ReusableInvitationCodeStatusActive},
		},
		releaseErr: errors.New("release failed"),
	}
	authService := newOAuthEmailFlowAuthService(
		userRepo,
		&redeemCodeRepoStub{},
		&refreshTokenCacheStub{},
		map[string]string{
			SettingKeyRegistrationEnabled:   "true",
			SettingKeyInvitationCodeEnabled: "true",
		},
		&emailCacheStub{},
		nil,
	)
	authService.SetReusableInvitationCodeRepository(reusableRepo)

	err := authService.RollbackOAuthEmailAccountCreation(context.Background(), 42, "OAUTH-EMAIL")

	require.ErrorContains(t, err, "restore reusable invitation code")
	require.Empty(t, userRepo.deletedIDs)
}
