package service

import "context"

func (s *BillingCacheService) reserveResellerInflight(ctx context.Context, user *User, group *Group, estimate float64, renew bool) (*InflightReservation, error) {
	account := user.ResellerCustomer
	quote := ResellerPriceFromContext(ctx, user.ID, group.ID)
	if quote == nil || !quote.ManagedCredits {
		return nil, ErrBillingServiceUnavailable
	}
	customer := *user
	customer.ID = account.UserID
	customer.ResellerCustomer = nil
	creditReservation, err := s.reserveInflight(ctx, &customer, group, nil, estimate, renew)
	if err != nil {
		return nil, err
	}
	owner := &User{ID: account.OwnerID}
	costReservation, err := s.reserveInflight(ctx, owner, group, nil, estimate/quote.Multiplier, renew)
	if err != nil {
		creditReservation.Release()
		return nil, err
	}
	if creditReservation == nil {
		return costReservation, nil
	}
	creditReservation.resellerCost = costReservation
	return creditReservation, nil
}
