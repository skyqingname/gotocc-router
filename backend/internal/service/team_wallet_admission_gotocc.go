package service

import "context"

// A subscription may grant access to a group, but team requests do not spend
// personal subscription quota. Recheck the entitlement after ownership changes.
func (s *APIKeyService) ValidateTeamGroupEntitlement(ctx context.Context, key *APIKey) error {
	if key.TeamID == nil || key.Group == nil || key.User.ResellerCustomer != nil || !key.Group.IsSubscriptionType() {
		return nil
	}
	_, err := s.userSubRepo.GetActiveByUserIDAndGroupID(ctx, key.User.ID, key.Group.ID)
	return err
}

// Personal identity still supplies group permissions and pricing. The balance
// belongs to the team and is never published to the personal balance cache.
func bindTeamWallet(key *APIKey) error {
	if key.TeamID == nil {
		return nil
	}
	if key.Team == nil || key.User == nil {
		return ErrTeamMembershipRequired
	}
	var source int64
	if key.User.ResellerCustomer != nil {
		source = key.User.ResellerCustomer.OwnerID
	}
	if source != valueOrZero(key.Team.ResellerOwnerID) {
		return ErrTeamFundingSource
	}
	user := *key.User
	user.TeamWalletID = key.TeamID
	user.Balance = key.Team.Balance
	user.FrozenBalance = key.Team.FrozenBalance
	if user.ResellerCustomer != nil {
		account := *user.ResellerCustomer
		account.CreditBalance, account.FrozenCredit = user.Balance, user.FrozenBalance
		user.ResellerCustomer = &account
	}
	key.User = &user
	return nil
}

func (s *BillingCacheService) checkTeamWalletEligibility(ctx context.Context, user *User) error {
	wallet, err := s.teamWalletRepo.GetWallet(ctx, *user.TeamWalletID)
	if err != nil {
		return err
	}
	if s.balanceBelowEligibilityThreshold(wallet.Balance) {
		return ErrTeamBalanceInsufficient
	}
	if user.ResellerCustomer != nil {
		account, err := s.resellerRepo.CustomerAccount(ctx, user.ID)
		if err != nil {
			return err
		}
		if account == nil || account.OwnerID != valueOrZero(wallet.ResellerOwnerID) {
			return ErrTeamFundingSource
		}
		account.CreditBalance, account.FrozenCredit = wallet.Balance, wallet.FrozenBalance
		return checkResellerCustomerFunds(account)
	}
	return nil
}

func (s *BillingCacheService) balanceForRequest(ctx context.Context, user *User) (float64, error) {
	if user.TeamWalletID != nil {
		wallet, err := s.teamWalletRepo.GetWallet(ctx, *user.TeamWalletID)
		if err != nil {
			return 0, err
		}
		return wallet.Balance, nil
	}
	return s.GetUserBalance(ctx, user.ID)
}

// User IDs are positive; negative IDs reserve a separate team namespace in the
// existing inflight-only cache. They are never sent to user storage or billing.
func inflightBalanceScopeID(user *User) int64 {
	if user.TeamWalletID != nil {
		return -*user.TeamWalletID
	}
	return user.ID
}
