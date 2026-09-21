package service

import (
	"context"
	"math"
	"strings"
	"time"

	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/reseller"
)

type ResellerProfile struct {
	UserID            int64      `json:"user_id"`
	Enabled           bool       `json:"enabled"`
	InvitationCode    string     `json:"invitation_code"`
	DefaultMultiplier float64    `json:"default_multiplier"`
	RebateRates       []*float64 `json:"rebate_rates"`
	GlobalRebateRates []float64  `json:"global_rebate_rates,omitempty"`
}
type ResellerPrice struct {
	CustomerID *int64   `json:"customer_id"`
	GroupID    *int64   `json:"group_id"`
	Multiplier *float64 `json:"multiplier"`
}
type ResellerCustomer struct {
	UserID      int64      `json:"user_id"`
	Username    string     `json:"username"`
	Email       string     `json:"email"`
	Status      string     `json:"status"`
	Notes       string     `json:"notes"`
	CreatedAt   time.Time  `json:"created_at"`
	Charged     float64    `json:"charged"`
	Profit      float64    `json:"profit"`
	LastUsageAt *time.Time `json:"last_usage_at"`
}
type ResellerEarning struct {
	ID         int64     `json:"id"`
	CustomerID int64     `json:"customer_id"`
	Username   string    `json:"username"`
	GroupID    int64     `json:"group_id"`
	GroupName  string    `json:"group_name"`
	Model      string    `json:"model"`
	Charged    float64   `json:"charged"`
	Cost       float64   `json:"cost"`
	Profit     float64   `json:"profit"`
	Multiplier float64   `json:"multiplier"`
	CreatedAt  time.Time `json:"created_at"`
}
type ResellerSummary struct {
	CustomerCount int64   `json:"customer_count"`
	Charged       float64 `json:"charged"`
	Profit        float64 `json:"profit"`
	Rebate        float64 `json:"rebate"`
}
type ResellerRepository interface {
	Profile(context.Context, int64) (*ResellerProfile, error)
	SaveProfile(context.Context, int64, bool, []*float64) (*ResellerProfile, error)
	Invitation(context.Context, string) (*ResellerProfile, error)
	BindCustomer(context.Context, int64, int64) error
	CustomerOwned(context.Context, int64, int64) (bool, error)
	Customers(context.Context, int64, string, int, int) ([]ResellerCustomer, int64, error)
	UpdateNotes(context.Context, int64, int64, string) error
	Prices(context.Context, int64) ([]ResellerPrice, error)
	SetPrices(context.Context, int64, *int64, *float64, []ResellerPrice) error
	Summary(context.Context, int64) (*ResellerSummary, error)
	Earnings(context.Context, int64, int, int) ([]ResellerEarning, int64, error)
	Pricing(context.Context, int64) (map[int64]*reseller.Snapshot, error)
}
type ResellerService struct {
	Repo     ResellerRepository
	keys     *APIKeyService
	settings *SettingService
}

func NewResellerService(repo ResellerRepository, keys *APIKeyService, settings *SettingService) *ResellerService {
	return &ResellerService{Repo: repo, keys: keys, settings: settings}
}
func (s *ResellerService) RequireEnabled(ctx context.Context, userID int64) (*ResellerProfile, error) {
	p, err := s.Repo.Profile(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !p.Enabled {
		return nil, infraerrors.Forbidden("RESELLER_DISABLED", "站长中心尚未向此用户开放")
	}
	return p, nil
}
func (s *ResellerService) AdminProfile(ctx context.Context, userID int64) (*ResellerProfile, error) {
	p, e := s.Repo.Profile(ctx, userID)
	if e != nil {
		return nil, e
	}
	rates, e := s.settings.GetAffiliateRebateRates(ctx)
	if e != nil {
		return nil, e
	}
	p.GlobalRebateRates = append([]float64(nil), rates[:]...)
	return p, nil
}
func (s *ResellerService) SaveProfile(ctx context.Context, userID int64, enabled bool, rates []*float64) (*ResellerProfile, error) {
	if len(rates) != AffiliateRebateGenerations {
		return nil, infraerrors.BadRequest("INVALID_RESELLER_REBATES", "请分别配置一、二、三代比例")
	}
	for _, v := range rates {
		if v != nil && (math.IsNaN(*v) || math.IsInf(*v, 0) || *v < 0 || *v > 100) {
			return nil, infraerrors.BadRequest("INVALID_RESELLER_REBATES", "返佣比例应在 0 到 100 之间")
		}
	}
	return s.Repo.SaveProfile(ctx, userID, enabled, rates)
}
func (s *ResellerService) Groups(ctx context.Context, userID int64) ([]Group, error) {
	groups, err := s.keys.GetAvailableGroups(ctx, userID)
	if err != nil {
		return nil, err
	}
	rates, err := s.keys.userGroupRateRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := range groups {
		if rate, ok := rates[groups[i].ID]; ok {
			groups[i].RateMultiplier = rate
		}
		ApplyResellerGroupPricing(ctx, userID, &groups[i])
	}
	return groups, nil
}
func (s *ResellerService) SetPrices(ctx context.Context, ownerID int64, customerID *int64, overall *float64, prices []ResellerPrice) error {
	if _, err := s.RequireEnabled(ctx, ownerID); err != nil {
		return err
	}
	if customerID != nil {
		owned, err := s.Repo.CustomerOwned(ctx, ownerID, *customerID)
		if err != nil {
			return err
		}
		if !owned {
			return infraerrors.NotFound("RESELLER_CUSTOMER_NOT_FOUND", "客户不属于当前站长")
		}
	}
	groups, err := s.Groups(ctx, ownerID)
	if err != nil {
		return err
	}
	available := map[int64]bool{}
	for _, g := range groups {
		available[g.ID] = true
	}
	valid := func(v *float64) bool {
		return v == nil || (!math.IsNaN(*v) && !math.IsInf(*v, 0) && *v >= reseller.MinimumMultiplier)
	}
	if !valid(overall) {
		return infraerrors.BadRequest("INVALID_RESELLER_MULTIPLIER", "客户倍率不得低于 1")
	}
	seen := map[int64]bool{}
	for i := range prices {
		p := &prices[i]
		if p.GroupID == nil || !available[*p.GroupID] || seen[*p.GroupID] || !valid(p.Multiplier) {
			return infraerrors.BadRequest("INVALID_RESELLER_PRICE", "请检查分组权限和倍率")
		}
		seen[*p.GroupID] = true
		p.CustomerID = customerID
	}
	return s.Repo.SetPrices(ctx, ownerID, customerID, overall, prices)
}
func (s *ResellerService) BindRegistration(ctx context.Context, userID, ownerID int64) error {
	return s.Repo.BindCustomer(ctx, userID, ownerID)
}
func IsResellerInvitation(code string) bool {
	return strings.HasPrefix(strings.ToUpper(strings.TrimSpace(code)), "RS-")
}

type resellerPricingContextKey struct{}
type resellerPricingContext struct {
	UserID int64
	Prices map[int64]*reseller.Snapshot
}

func WithResellerPrices(ctx context.Context, userID int64, prices map[int64]*reseller.Snapshot) context.Context {
	return context.WithValue(ctx, resellerPricingContextKey{}, resellerPricingContext{userID, prices})
}
func ResellerPricesFromContext(ctx context.Context, userID int64) map[int64]*reseller.Snapshot {
	p, _ := ctx.Value(resellerPricingContextKey{}).(resellerPricingContext)
	if p.UserID != userID {
		return nil
	}
	return p.Prices
}
func ResellerPriceFromContext(ctx context.Context, userID, groupID int64) *reseller.Snapshot {
	return ResellerPricesFromContext(ctx, userID)[groupID]
}
func (k *APIKey) ResellerPrice() *reseller.Snapshot {
	if k == nil || k.GroupID == nil {
		return nil
	}
	return k.ResellerPrices[*k.GroupID]
}

func ResellerPriceForOptionalGroup(ctx context.Context, userID int64, groupID *int64) *reseller.Snapshot {
	if groupID == nil {
		return nil
	}
	return ResellerPriceFromContext(ctx, userID, *groupID)
}
func ApplyResellerGroupPricing(ctx context.Context, userID int64, g *Group) {
	if q := ResellerPriceFromContext(ctx, userID, g.ID); q != nil {
		g.RateMultiplier = q.TextRate
		g.ImageRateIndependent = true
		g.ImageRateMultiplier = q.ImageRate
		g.VideoRateIndependent = true
		g.VideoRateMultiplier = q.VideoRate
	}
}

func (k *APIKey) ResellerPriceAt(at time.Time) *reseller.Snapshot {
	if k != nil && k.Group != nil {
		if snapshot, ok := k.Group.requestRateAt(at); ok {
			return snapshot.Reseller
		}
	}
	return k.ResellerPrice()
}

// Public model discovery may be anonymous; only the authenticated context has prices.
func ApplyCurrentResellerGroupPricing(ctx context.Context, g *Group) {
	p, _ := ctx.Value(resellerPricingContextKey{}).(resellerPricingContext)
	ApplyResellerGroupPricing(ctx, p.UserID, g)
}
