package dto

import (
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/service"
)

// RedeemCodeUsage is one user's redemption of a shared redeem code.
type RedeemCodeUsage struct {
	ID           int64     `json:"id"`
	RedeemCodeID int64     `json:"redeem_code_id"`
	UserID       int64     `json:"user_id"`
	UsedAt       time.Time `json:"used_at"`
	User         *User     `json:"user,omitempty"`
}

func RedeemCodeUsageFromService(u *service.RedeemCodeUsage) RedeemCodeUsage {
	return RedeemCodeUsage{
		ID:           u.ID,
		RedeemCodeID: u.RedeemCodeID,
		UserID:       u.UserID,
		UsedAt:       u.UsedAt,
		User:         UserFromServiceShallow(u.User),
	}
}
