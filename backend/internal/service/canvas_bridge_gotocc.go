package service

import (
	"context"
	"errors"
	"strings"
)

// CanvasBridgeVerifyInput is retained for existing GoToCC-backed Canvas logins.
// Canvas funding is handled exclusively by the explicit card exchange flow.
type CanvasBridgeVerifyInput struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type CanvasBridgeAccount struct {
	UserID           int64  `json:"user_id"`
	Email            string `json:"email"`
	Username         string `json:"username"`
	Status           string `json:"status"`
	ResellerCustomer bool   `json:"reseller_customer"`
	PayerUserID      int64  `json:"payer_user_id"`
	TeamID           *int64 `json:"team_id,omitempty"`
}

type CanvasBridgeService struct {
	users     UserRepository
	resellers ResellerRepository
}

func NewCanvasBridgeService(users UserRepository, resellers ResellerRepository) *CanvasBridgeService {
	return &CanvasBridgeService{users: users, resellers: resellers}
}

func (s *CanvasBridgeService) Verify(ctx context.Context, in CanvasBridgeVerifyInput) (*CanvasBridgeAccount, error) {
	user, err := s.users.GetByEmail(ctx, strings.TrimSpace(in.Email))
	if errors.Is(err, ErrUserNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	if !user.CheckPassword(in.Password) {
		return nil, ErrInvalidCredentials
	}
	customer, err := s.resellers.CustomerAccount(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	return &CanvasBridgeAccount{UserID: user.ID, Email: user.Email, Username: user.Username, Status: user.Status,
		ResellerCustomer: customer != nil, PayerUserID: user.ID}, nil
}
