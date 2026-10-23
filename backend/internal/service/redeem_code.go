package service

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

type RedeemCode struct {
	CashbackAmount float64 // Response-only: the separate cashback credited by this redemption.
	ID             int64
	Code           string
	Type           string
	Value          float64
	Status         string
	UsedBy         *int64
	UsedAt         *time.Time
	Notes          string
	CreatedAt      time.Time
	ExpiresAt      *time.Time

	GenerateOrder bool
	OrderCurrency string
	OrderPlanID   *int64
	OrderAmount   float64
	OrderPlanName string

	GroupID      *int64
	ValidityDays int

	// MaxUses is the redemption limit: nil keeps the schema default of a
	// single-use code, 0 is unlimited, other values make a shared code.
	MaxUses   *int
	UsedCount int

	User  *User
	Group *Group
}

// IsShared reports whether several users may redeem this code, each once.
func (r *RedeemCode) IsShared() bool {
	return r.MaxUses != nil && *r.MaxUses != 1
}

func (r *RedeemCode) IsUsed() bool {
	return r.Status == StatusUsed
}

func (r *RedeemCode) IsExpired() bool {
	return r.IsExpiredAt(time.Now())
}

func (r *RedeemCode) IsExpiredAt(now time.Time) bool {
	if r == nil {
		return false
	}
	if r.Status == StatusExpired {
		return true
	}
	return r.Status == StatusUnused && r.ExpiresAt != nil && !r.ExpiresAt.After(now)
}

func (r *RedeemCode) CanUse() bool {
	return r.Status == StatusUnused && !r.IsExpired()
}

// RedeemCodeUsage is one user's redemption of a shared code.
type RedeemCodeUsage struct {
	ID           int64
	RedeemCodeID int64
	UserID       int64
	UsedAt       time.Time

	User *User
}

func GenerateRedeemCode() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
