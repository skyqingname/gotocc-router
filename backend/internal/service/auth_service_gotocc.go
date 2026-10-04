package service

import (
	"context"
	"errors"
	"strings"
	"time"

	dbent "github.com/LuckyKuang/sub2api-plus/ent"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/logger"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/resellersite"
)

type registrationInvitation struct {
	reseller *ResellerProfile
	redeem   *RedeemCode
	reusable *ReusableInvitationCode
}

func (s *AuthService) SetReusableInvitationCodeRepository(repo ReusableInvitationCodeRepository) {
	if s != nil {
		s.reusableInvitationRepo = repo
	}
}

func (s *AuthService) resolveRegistrationInvitation(ctx context.Context, invitationCode string, missingErr error) (*registrationInvitation, error) {
	invitationCode = strings.TrimSpace(invitationCode)
	if IsResellerInvitation(invitationCode) {
		profile, err := s.resellerService.Repo.Invitation(ctx, invitationCode)
		if err != nil {
			return nil, err
		}
		return &registrationInvitation{reseller: profile}, nil
	}
	if resellersite.IsCustomer(ctx) {
		if invitationCode == "" {
			return nil, missingErr
		}
		return nil, ErrInvitationCodeInvalid
	}
	if invitationCode == "" {
		if s.settingService != nil && s.settingService.IsInvitationCodeEnabled(ctx) {
			return nil, missingErr
		}
		return nil, nil
	}
	if s.redeemRepo != nil {
		redeemCode, err := s.redeemRepo.GetByCode(ctx, invitationCode)
		if err == nil && redeemCode.Type == RedeemTypeInvitation && redeemCode.CanUse() {
			return &registrationInvitation{redeem: redeemCode}, nil
		}
	}
	if s.reusableInvitationRepo != nil {
		reusableCode, err := s.reusableInvitationRepo.GetByCode(ctx, invitationCode)
		if err == nil {
			if !reusableCode.IsUsableAt(time.Now()) {
				return nil, ErrInvitationCodeInvalid
			}
			return &registrationInvitation{reusable: reusableCode}, nil
		}
		if !errors.Is(err, ErrReusableInvitationCodeNotFound) {
			return nil, err
		}
	}
	return nil, ErrInvitationCodeInvalid
}

func (s *AuthService) useRegistrationInvitation(ctx context.Context, invitation *registrationInvitation, user *User, authSource string, failOpenOneTime bool) error {
	if invitation == nil || user == nil {
		return nil
	}
	if invitation.reseller != nil {
		return s.resellerService.BindRegistration(ctx, user, invitation.reseller.UserID)
	}
	if invitation.redeem != nil {
		if s.redeemRepo == nil {
			return ErrInvitationCodeInvalid
		}
		err := s.redeemRepo.Use(ctx, invitation.redeem.ID, user.ID)
		if err != nil && failOpenOneTime {
			logger.LegacyPrintf("service.auth", "[Auth] Failed to mark one-time invitation as used for user %d: %v", user.ID, err)
			return nil
		}
		return err
	}
	if invitation.reusable != nil {
		if s.reusableInvitationRepo == nil {
			return ErrInvitationCodeInvalid
		}
		return s.reusableInvitationRepo.Use(ctx, invitation.reusable.ID, user.ID, user.Email, authSource)
	}
	return nil
}

func (s *AuthService) cleanupCreatedUserAfterInvitationFailure(ctx context.Context, user *User) {
	if s == nil || s.userRepo == nil || user == nil || user.ID <= 0 {
		return
	}
	if err := s.userRepo.Delete(ctx, user.ID); err != nil {
		logger.LegacyPrintf("service.auth", "[Auth] Failed to delete user %d after invitation consumption failure: %v", user.ID, err)
	}
}

func (s *AuthService) createUserWithRegistrationInvitation(ctx context.Context, user *User, invitation *registrationInvitation, authSource string) error {
	if invitation != nil && invitation.reseller != nil {
		user.Balance = 0
	}
	if invitation == nil {
		return s.createUserAndClaimInvitation(ctx, user, nil)
	}
	if invitation.redeem != nil {
		return s.createUserAndClaimInvitation(ctx, user, invitation.redeem)
	}
	if invitation.reusable == nil && invitation.reseller == nil {
		return ErrInvitationCodeInvalid
	}
	if s.entClient == nil {
		if err := s.createUserWithRegistrationEmailGuard(ctx, user); err != nil {
			return err
		}
		if err := s.useRegistrationInvitation(ctx, invitation, user, authSource, false); err != nil {
			s.cleanupCreatedUserAfterInvitationFailure(ctx, user)
			return ErrInvitationCodeInvalid
		}
		return nil
	}

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)
	if err := s.createUserWithRegistrationEmailGuard(txCtx, user); err != nil {
		return err
	}
	if err := s.useRegistrationInvitation(txCtx, invitation, user, authSource, false); err != nil {
		return ErrInvitationCodeInvalid
	}
	return tx.Commit()
}

// ensureSignupInvitation initializes the user default invitation. Registration and
// referral binding already happen together through the unified invitation resolver.
func (s *AuthService) ensureSignupInvitation(ctx context.Context, userID int64) {
	if s.affiliateService == nil || userID <= 0 {
		return
	}
	_, err := s.affiliateService.EnsureUserAffiliate(ctx, userID)
	if err != nil {
		logger.LegacyPrintf("service.auth", "[Auth] Failed to initialize affiliate profile for user %d: %v", userID, err)
		return
	}

}

// ValidateRegistrationInvitation shares code resolution with email and OAuth signup.
func (s *AuthService) ValidateRegistrationInvitation(ctx context.Context, code string) error {
	_, err := s.resolveRegistrationInvitation(ctx, code, ErrInvitationCodeRequired)
	return err
}

// resellerSignupGrantPlan keeps platform signup gifts separate from station credits.
func resellerSignupGrantPlan(plan signupGrantPlan, invitation *registrationInvitation) signupGrantPlan {
	if invitation != nil && invitation.reseller != nil {
		plan.Balance = 0
		plan.Subscriptions = nil
	}
	return plan
}
