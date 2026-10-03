package service

import (
	"context"
	"math"
	"strings"
	"time"

	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/pagination"
)

var ErrResellerCustomerNotFound = infraerrors.NotFound("RESELLER_CUSTOMER_NOT_FOUND", "客户不属于当前站长")
var ErrResellerCreditInsufficient = infraerrors.Forbidden("RESELLER_CREDIT_INSUFFICIENT", "客户额度不足，请联系站长充值")
var ErrResellerBalanceInsufficient = infraerrors.Forbidden("RESELLER_BALANCE_INSUFFICIENT", "站长可用余额不足，请联系站长")
var ErrResellerOwnerInactive = infraerrors.Forbidden("RESELLER_OWNER_INACTIVE", "所属站长账户已停用，请联系站长")

type ResellerCustomerAccount struct {
	UserID         int64   `json:"user_id"`
	OwnerID        int64   `json:"owner_id"`
	OwnerName      string  `json:"owner_name"`
	CreditBalance  float64 `json:"credit_balance"`
	FrozenCredit   float64 `json:"frozen_credit"`
	OwnerBalance   float64 `json:"-"`
	OwnerActive    bool    `json:"-"`
	CustomerActive bool    `json:"-"`
}

type ResellerCreditInput struct {
	OperationID string  `json:"operation_id" binding:"required"`
	Kind        string  `json:"kind" binding:"required,oneof=purchase gift deduct"`
	Amount      float64 `json:"amount" binding:"required,gt=0"`
	Notes       string  `json:"notes"`
}

type ResellerCreditEntry struct {
	ID           int64     `json:"id"`
	CustomerID   int64     `json:"customer_id"`
	OwnerID      int64     `json:"owner_id"`
	OperationID  string    `json:"operation_id"`
	Kind         string    `json:"kind"`
	Amount       float64   `json:"amount"`
	FrozenAmount float64   `json:"frozen_amount"`
	BalanceAfter float64   `json:"balance_after"`
	FrozenAfter  float64   `json:"frozen_after"`
	PlatformCost float64   `json:"platform_cost"`
	Model        string    `json:"model"`
	Notes        string    `json:"notes"`
	CreatedAt    time.Time `json:"created_at"`
}

type ResellerCustomerInput struct {
	Email         string  `json:"email" binding:"required,email"`
	Username      string  `json:"username"`
	Password      string  `json:"password"`
	Status        string  `json:"status" binding:"required,oneof=active disabled"`
	Concurrency   int     `json:"concurrency" binding:"required,gt=0"`
	RPMLimit      int     `json:"rpm_limit" binding:"gte=0"`
	AllowedGroups []int64 `json:"allowed_groups"`
	Notes         string  `json:"notes"`
}

func (s *ResellerService) RequireCustomer(ctx context.Context, ownerID, customerID int64) error {
	if _, err := s.RequireEnabled(ctx, ownerID); err != nil {
		return err
	}
	owned, err := s.Repo.CustomerOwned(ctx, ownerID, customerID)
	if err != nil {
		return err
	}
	if !owned {
		return ErrResellerCustomerNotFound
	}
	return nil
}

func (s *ResellerService) Customer(ctx context.Context, ownerID, customerID int64) (*User, error) {
	if err := s.RequireCustomer(ctx, ownerID, customerID); err != nil {
		return nil, err
	}
	user, err := s.keys.userRepo.GetByID(ctx, customerID)
	if err != nil {
		return nil, err
	}
	account, err := s.Repo.CustomerAccount(ctx, customerID)
	if err != nil {
		return nil, err
	}
	applyResellerCustomerAccount(user, account)
	return user, nil
}

func (s *ResellerService) SaveCustomer(ctx context.Context, ownerID, customerID int64, input ResellerCustomerInput) (*User, error) {
	if _, err := s.RequireEnabled(ctx, ownerID); err != nil {
		return nil, err
	}
	owner, err := s.keys.userRepo.GetByID(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	if input.Concurrency <= 0 || input.Concurrency > owner.Concurrency || input.RPMLimit < 0 || (owner.RPMLimit > 0 && (input.RPMLimit == 0 || input.RPMLimit > owner.RPMLimit)) {
		return nil, infraerrors.BadRequest("INVALID_CUSTOMER_LIMITS", "客户并发与请求限额须在站长可用范围内")
	}
	groups, err := s.keys.GetAvailableGroups(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	allowed := make(map[int64]bool, len(groups))
	for _, group := range groups {
		allowed[group.ID] = true
	}
	for _, id := range input.AllowedGroups {
		if !allowed[id] {
			return nil, ErrGroupNotAllowed
		}
	}
	var user *User
	err = s.Repo.CustomerTransaction(ctx, ownerID, customerID, func(txctx context.Context) error {
		if customerID == 0 {
			if strings.TrimSpace(input.Password) == "" {
				return infraerrors.BadRequest("CUSTOMER_PASSWORD_REQUIRED", "创建客户时必须设置密码")
			}
			user = &User{Role: RoleUser, Balance: 0, SignupSource: "admin"}
		} else {
			var e error
			user, e = s.keys.userRepo.GetByID(txctx, customerID)
			if e != nil {
				return e
			}
		}
		user.Email = strings.TrimSpace(input.Email)
		user.Username = strings.TrimSpace(input.Username)
		user.Status = input.Status
		user.Concurrency = input.Concurrency
		user.RPMLimit = input.RPMLimit
		user.AllowedGroups = input.AllowedGroups
		user.RestrictPublicGroups = true
		if input.Password != "" {
			if e := user.SetPassword(input.Password); e != nil {
				return e
			}
		}
		if customerID == 0 {
			if e := s.keys.userRepo.Create(txctx, user); e != nil {
				return e
			}
			if e := s.Repo.BindCustomer(txctx, user.ID, ownerID); e != nil {
				return e
			}
		} else {
			fields := UserUpdateFields{Email: true, Username: true, Status: true, Concurrency: true, RPMLimit: true, AllowedGroups: true, RestrictPublicGroups: true, PasswordHash: input.Password != ""}
			if e := s.keys.userRepo.Update(txctx, user, fields); e != nil {
				return e
			}
		}
		return s.Repo.UpdateNotes(txctx, ownerID, user.ID, input.Notes)
	})
	if err != nil {
		return nil, err
	}
	s.keys.InvalidateAuthCacheByUserID(ctx, user.ID)
	return s.Customer(ctx, ownerID, user.ID)
}

func (s *ResellerService) ChangeCredit(ctx context.Context, ownerID, customerID int64, input ResellerCreditInput) (*ResellerCreditEntry, error) {
	if err := s.RequireCustomer(ctx, ownerID, customerID); err != nil {
		return nil, err
	}
	input.OperationID = strings.TrimSpace(input.OperationID)
	input.Amount = QuantizeUsageBillingAmount(input.Amount)
	if input.OperationID == "" || math.IsNaN(input.Amount) || math.IsInf(input.Amount, 0) || input.Amount <= 0 {
		return nil, infraerrors.BadRequest("INVALID_CUSTOMER_CREDIT", "请填写有效的额度")
	}
	if input.Kind != "purchase" && input.Kind != "gift" && input.Kind != "deduct" {
		return nil, infraerrors.BadRequest("INVALID_CUSTOMER_CREDIT_KIND", "请选择购买、赠送或扣减")
	}
	return s.Repo.ChangeCustomerCredit(ctx, ownerID, customerID, input)
}

func applyResellerCustomerAccount(user *User, account *ResellerCustomerAccount) {
	user.ResellerCustomer = account
	if account != nil {
		user.Balance = account.CreditBalance
		user.FrozenBalance = account.FrozenCredit
	}
}

func checkResellerCustomerFunds(account *ResellerCustomerAccount) error {
	if !account.CustomerActive {
		return infraerrors.Forbidden("RESELLER_CUSTOMER_INACTIVE", "客户账户已停用，请联系站长")
	}
	if !account.OwnerActive {
		return ErrResellerOwnerInactive
	}
	if account.CreditBalance <= 0 {
		return ErrResellerCreditInsufficient
	}
	if account.OwnerBalance <= 0 {
		return ErrResellerBalanceInsufficient
	}
	return nil
}

func (s *ResellerService) CustomerKeys(ctx context.Context, ownerID, customerID int64, p pagination.PaginationParams) ([]APIKey, int64, error) {
	if err := s.RequireCustomer(ctx, ownerID, customerID); err != nil {
		return nil, 0, err
	}
	keys, result, err := s.keys.List(ctx, customerID, p, APIKeyListFilters{})
	if err != nil {
		return nil, 0, err
	}
	return keys, result.Total, nil
}
func (s *ResellerService) CreateCustomerKey(ctx context.Context, ownerID, customerID int64, input CreateAPIKeyRequest) (*APIKey, error) {
	if err := s.RequireCustomer(ctx, ownerID, customerID); err != nil {
		return nil, err
	}
	input.Scope = "personal"
	return s.keys.Create(ctx, customerID, input)
}
func (s *ResellerService) UpdateCustomerKey(ctx context.Context, ownerID, customerID, keyID int64, input UpdateAPIKeyRequest) (*APIKey, error) {
	if err := s.RequireCustomer(ctx, ownerID, customerID); err != nil {
		return nil, err
	}
	return s.keys.Update(ctx, keyID, customerID, input)
}
func (s *ResellerService) DeleteCustomerKey(ctx context.Context, ownerID, customerID, keyID int64) error {
	if err := s.RequireCustomer(ctx, ownerID, customerID); err != nil {
		return err
	}
	return s.keys.Delete(ctx, keyID, customerID)
}

type ResellerOwnerLimits struct {
	Balance     float64 `json:"balance"`
	Concurrency int     `json:"concurrency"`
	RPMLimit    int     `json:"rpm_limit"`
}

func (s *ResellerService) OwnerLimits(ctx context.Context, ownerID int64) (*ResellerOwnerLimits, error) {
	owner, err := s.keys.userRepo.GetByID(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	return &ResellerOwnerLimits{Balance: owner.Balance, Concurrency: owner.Concurrency, RPMLimit: owner.RPMLimit}, nil
}

func (s *APIKeyService) ValidateResellerFunds(key *APIKey) error {
	if key.User.ResellerCustomer == nil {
		return nil
	}
	return checkResellerCustomerFunds(key.User.ResellerCustomer)
}
