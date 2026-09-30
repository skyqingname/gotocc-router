package service

import (
	"context"
	"strings"
)

// Video creation must obtain an explicit quote before an upstream request.
// Resolution tiers match the request; durations are never clamped for pricing.
func (s *OpenAIGatewayService) calculateConfiguredOpenAIVideoCost(ctx context.Context, model string, apiKey *APIKey, request *OpenAIForwardResult, multiplier float64) *CostBreakdown {
	resolved := s.resolveOpenAIChannelPricing(ctx, model, apiKey)
	if resolved != nil && resolved.Source == PricingSourceGroup && isOpenAIVideoRequestBillingMode(resolved.Mode) {
		return configuredOpenAIVideoQuote(resolved, request, multiplier)
	}
	for _, key := range []*APIKey{apiKey, s.apiKeyWithFreshGroupMediaPricing(ctx, apiKey)} {
		if price := configuredOpenAIVideoGroupPrice(videoPriceConfigFromAPIKey(key), model, request.VideoResolution); price != nil {
			return configuredOpenAIVideoQuote(&ResolvedPricing{Mode: BillingModeVideo, DefaultPerRequestPrice: *price}, request, multiplier)
		}
	}
	if resolved != nil && isOpenAIVideoRequestBillingMode(resolved.Mode) {
		return configuredOpenAIVideoQuote(resolved, request, multiplier)
	}
	return nil
}

func configuredOpenAIVideoQuote(pricing *ResolvedPricing, request *OpenAIForwardResult, multiplier float64) *CostBreakdown {
	if len(pricing.RequestTiers) == 0 && pricing.channelPricing != nil && pricing.channelPricing.PerRequestPrice == nil {
		return nil
	}
	price := pricing.DefaultPerRequestPrice
	if len(pricing.RequestTiers) > 0 {
		found := false
		var generalPrice *float64
		for _, tier := range pricing.RequestTiers {
			if strings.EqualFold(strings.TrimSpace(tier.TierLabel), request.VideoResolution) && tier.PerRequestPrice != nil {
				price, found = *tier.PerRequestPrice, true
				break
			}
			if strings.TrimSpace(tier.TierLabel) == "" && tier.MinTokens == 0 && tier.PerRequestPrice != nil {
				generalPrice = tier.PerRequestPrice
			}
		}
		if !found {
			if generalPrice == nil {
				return nil
			}
			price = *generalPrice
		}
	}
	units, mode := openAIVideoBillingUnits(pricing.Mode, request.VideoCount, request.VideoDurationSeconds)
	total := price * units
	return &CostBreakdown{TotalCost: total, ActualCost: total * multiplier, BillingMode: string(mode)}
}

func configuredOpenAIVideoGroupPrice(prices *VideoPriceConfig, model, resolution string) *float64 {
	if prices == nil {
		return nil
	}
	family := CanonicalGrokImagineVideoPriceFamily(model)
	if family == "" {
		family = strings.ToLower(strings.TrimSpace(model))
	}
	if price, exists := prices.ModelPrices[family][resolution]; exists {
		return &price
	}
	switch resolution {
	case VideoBillingResolution480P:
		return prices.Price480P
	case VideoBillingResolution720P:
		return prices.Price720P
	case VideoBillingResolution1080P:
		return prices.Price1080P
	}
	return nil
}
