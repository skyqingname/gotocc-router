package service

import (
	"context"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/pagination"
)

// RedeemActivity combines wallet entries and issued cards without duplicating
// the personal debit already recorded for a Canvas order.
type RedeemActivity struct {
	ID           int64                `json:"id"`
	Code         string               `json:"code"`
	Type         string               `json:"type"`
	Value        float64              `json:"value"`
	Status       string               `json:"status"`
	UsedAt       *time.Time           `json:"used_at"`
	CreatedAt    time.Time            `json:"created_at"`
	Notes        string               `json:"notes,omitempty"`
	GroupID      *int64               `json:"group_id,omitempty"`
	ValidityDays int                  `json:"validity_days"`
	Group        *RedeemActivityGroup `json:"group,omitempty"`
	CanvasCard   *CanvasCardOrder     `json:"canvas_card,omitempty"`
}

type RedeemActivityGroup struct {
	ID   int64     `json:"id"`
	Name string    `json:"name"`
	I18n GroupI18n `json:"-"`
}

func (s *CanvasCardService) Activity(ctx context.Context, userID int64, params pagination.PaginationParams) ([]RedeemActivity, int64, error) {
	return s.repo.Activity(ctx, userID, params)
}
