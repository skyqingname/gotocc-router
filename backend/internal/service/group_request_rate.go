package service

import (
	"context"
	"sync"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/ctxkey"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/reseller"
)

type requestGroupRate struct {
	Base, Factor float64
	Reseller     *reseller.Snapshot
}

func (g *Group) requestRateAt(at time.Time) (requestGroupRate, bool) {
	if g == nil || g.requestRates == nil {
		return requestGroupRate{}, false
	}
	value, ok := g.requestRates.Load(at)
	if !ok {
		return requestGroupRate{}, false
	}
	return value.(requestGroupRate), true
}

// Authentication materializes a fresh Group for each request. The per-request
// map is never serialized or shared with the authentication cache. WS turns use
// distinct pricing instants so queued settlement keeps the matching snapshot.
func freezeRequestGroupRate(ctx context.Context, at time.Time, resolve func(context.Context, int64, int64, float64) float64) {
	group, _ := ctx.Value(ctxkey.Group).(*Group)
	userID, _ := ctx.Value(ctxkey.UserID).(int64)
	if group == nil || userID <= 0 {
		return
	}
	if group.requestRates == nil {
		group.requestRates = &sync.Map{}
	}
	if _, exists := group.requestRateAt(at); exists {
		return
	}
	group.requestRates.Store(at, requestGroupRate{
		Base:     resolve(ctx, userID, group.ID, group.RateMultiplier),
		Factor:   group.PeakMultiplierAt(at),
		Reseller: ResellerPriceFromContext(ctx, userID, group.ID),
	})
}

func (s *GatewayService) WithTokenRequestPricing(ctx context.Context) (context.Context, time.Time) {
	ctx, at := WithGatewayTokenRequestPricing(ctx)
	freezeRequestGroupRate(ctx, at, s.ResolveUserGroupRateMultiplier)
	return ctx, at
}
