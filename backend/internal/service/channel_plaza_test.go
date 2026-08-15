//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPlazaImageDisplayPricing_GroupTiersOverrideChannelPricing(t *testing.T) {
	channelDefault := 0.2
	channel4K := 0.3
	group1K := 0.02
	pricing := &ChannelModelPricing{
		BillingMode:     BillingModeImage,
		PerRequestPrice: &channelDefault,
		Intervals:       []PricingInterval{{TierLabel: "4K", PerRequestPrice: &channel4K}},
	}
	group := &Group{ImagePrice1K: &group1K}

	got := plazaImageDisplayPricing(pricing, group)

	require.NotSame(t, pricing, got)
	require.Len(t, got.Intervals, 3)
	tiers := make(map[string]float64, len(got.Intervals))
	for _, interval := range got.Intervals {
		require.NotNil(t, interval.PerRequestPrice)
		tiers[interval.TierLabel] = *interval.PerRequestPrice
	}
	require.InDelta(t, group1K, tiers["1K"], 1e-12)
	require.InDelta(t, channelDefault, tiers["2K"], 1e-12)
	require.InDelta(t, channel4K, tiers["4K"], 1e-12)
	require.Len(t, pricing.Intervals, 1, "display composition must not mutate cached channel pricing")
}

func TestPlazaImageDisplayPricing_IgnoresNonImagePricing(t *testing.T) {
	groupPrice := 0.02
	pricing := &ChannelModelPricing{BillingMode: BillingModeToken, InputPrice: testPtrFloat64(3e-6)}

	got := plazaImageDisplayPricing(pricing, &Group{ImagePrice1K: &groupPrice})

	require.Same(t, pricing, got)
	require.Empty(t, got.Intervals)
}

func TestLookupOfficialPricing(t *testing.T) {
	pricingService := newStubPricingServiceFromMap(map[string]*LiteLLMModelPricing{
		"claude-sonnet": {
			Mode:                                "chat",
			InputCostPerToken:                   3e-6,
			OutputCostPerToken:                  1.5e-5,
			CacheCreationInputTokenCost:         3.75e-6,
			CacheCreationInputTokenCostAbove1hr: 6e-6,
			CacheReadInputTokenCost:             3e-7,
		},
		"token-absent": {
			Mode:               "image_generation",
			TokenPricingAbsent: true,
			OutputCostPerImage: 0.04,
		},
	})
	svc := &ChannelService{pricingService: pricingService}
	memo := make(map[string]*PlazaOfficialPricing)

	official := svc.lookupOfficialPricing("claude-sonnet", memo)

	require.NotNil(t, official)
	require.InDelta(t, 3e-6, *official.InputPrice, 1e-12)
	require.InDelta(t, 6e-6, *official.CacheWrite1hPrice, 1e-12)
	require.InDelta(t, 3e-7, *official.CacheReadPrice, 1e-12)
	require.Nil(t, svc.lookupOfficialPricing("unknown-model", memo))
	require.Nil(t, svc.lookupOfficialPricing("token-absent", memo))
}
