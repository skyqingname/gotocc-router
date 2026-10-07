package service

import "context"

func (s *APIKeyService) attachResellerCustomer(ctx context.Context, key *APIKey) error {
	// The current owner supplies team pricing; the wallet supplies its independent balance.
	account, err := s.resellerRepo.CustomerAccount(ctx, key.User.ID)
	if err != nil {
		return err
	}
	if account == nil {
		return bindTeamWallet(key)
	}
	customer, err := s.userRepo.GetByID(ctx, account.UserID)
	if err != nil {
		return err
	}
	key.User.Status = customer.Status
	applyResellerCustomerAccount(key.User, account)
	key.ResellerPrices, err = s.resellerRepo.Pricing(ctx, account.UserID)
	if err != nil {
		return err
	}
	groups, err := s.GetAvailableGroups(ctx, account.UserID)
	if err != nil {
		return err
	}
	key.User.RestrictPublicGroups = true
	key.User.AllowedGroups = make([]int64, 0, len(groups))
	for _, group := range groups {
		key.User.AllowedGroups = append(key.User.AllowedGroups, group.ID)
	}
	return bindTeamWallet(key)
}

func (s *APIKeyService) resellerAvailableGroups(ctx context.Context, customerID int64, account *ResellerCustomerAccount) ([]Group, error) {
	ownerGroups, err := s.GetAvailableGroups(ctx, account.OwnerID)
	if err != nil {
		return nil, err
	}
	customer, err := s.userRepo.GetByID(ctx, customerID)
	if err != nil {
		return nil, err
	}
	prices, err := s.resellerRepo.Pricing(ctx, customerID)
	if err != nil {
		return nil, err
	}
	pricingContext := WithResellerPrices(ctx, customerID, prices)
	groups := make([]Group, 0, len(ownerGroups))
	for _, group := range ownerGroups {
		if customer.CanBindGroup(group.ID, group.IsExclusive) {
			ApplyResellerGroupPricing(pricingContext, customerID, &group)
			groups = append(groups, group)
		}
	}
	return groups, nil
}

func (s *UserService) hydrateResellerCustomer(ctx context.Context, user *User) error {
	account, err := s.resellerRepo.CustomerAccount(ctx, user.ID)
	if err != nil {
		return err
	}
	applyResellerCustomerAccount(user, account)
	return nil
}

func (s *BillingCacheService) checkResellerBillingEligibility(ctx context.Context, user *User) error {
	account, err := s.resellerRepo.CustomerAccount(ctx, user.ResellerCustomer.UserID)
	if err != nil {
		return err
	}
	return checkResellerCustomerFunds(account)
}

func ProvideResellerUserService(users UserRepository, settings SettingRepository, authCache APIKeyAuthCacheInvalidator, billing BillingCache, resellers ResellerRepository) *UserService {
	svc := NewUserService(users, settings, authCache, billing)
	svc.resellerRepo = resellers
	return svc
}
