//go:build unit

package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/config"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/timezone"
	"github.com/stretchr/testify/require"
)

func TestIsDeepSeekModel(t *testing.T) {
	deepseek := []string{
		"deepseek-flash", "deepseek-v4-flash", "deepseek-v4-pro", "deepseek-v4-flash-vision-exp",
		"deepseek-chat", "deepseek-reasoner", "deepseek-v3-2-251201",
		"deepseek-coder", "deepseek-foo", "deepseek-v4-pro-0813",
		"DEEPSEEK-V4-PRO", " deepseek-v4-flash ",
	}
	for _, m := range deepseek {
		require.True(t, isDeepSeekModel(m), "model %q should be deepseek", m)
	}

	nonDeepseek := []string{
		"gpt-5.4", "claude-sonnet-4", "deepseekcoder", // 无连字符不算 deepseek- 前缀
		"", " deepseek", // 无连字符后缀
	}
	for _, m := range nonDeepseek {
		require.False(t, isDeepSeekModel(m), "model %q should not be deepseek", m)
	}
}

func TestCalculateCostUnified_DeepseekPricingAtZeroFallsBackToNow(t *testing.T) {
	bs := newTestBillingService()
	resolver := NewModelPricingResolver(nil, bs)

	tokens := UsageTokens{InputTokens: 1000, OutputTokens: 500}
	base := CostInput{
		Ctx: context.Background(), Model: "deepseek-v4-flash", Tokens: tokens,
		RateMultiplier: 1.0, Resolver: resolver,
	}

	// PricingAt 零值 → 回退 timezone.Now()，与显式传入当前时刻结果一致。
	costZero, err := bs.CalculateCostUnified(base)
	require.NoError(t, err)

	costNow, err := bs.CalculateCostUnified(CostInput{
		Ctx: base.Ctx, Model: base.Model, Tokens: base.Tokens,
		RateMultiplier: base.RateMultiplier, Resolver: base.Resolver,
		PricingAt: timezone.Now(),
	})
	require.NoError(t, err)
	require.Equal(t, costZero.TotalCost, costNow.TotalCost)
}

// ---------------------------------------------------------------------------
// 官方价强制覆盖（远端旧价兜底）与未知 deepseek-* flash 兜底
// ---------------------------------------------------------------------------

func TestGetModelPricing_DeepseekForcesOfficialRatesOverJSON(t *testing.T) {
	// JSON 给任意价（模拟远端旧价/占位价），deepseek-* 必须被强制覆盖为官方高峰价。
	pricingSvc := &PricingService{pricingData: map[string]*LiteLLMModelPricing{
		"deepseek-flash":               {InputCostPerToken: 1e-6, OutputCostPerToken: 2e-6, CacheReadInputTokenCost: 1e-8},
		"deepseek-v4-flash":            {InputCostPerToken: 1e-6, OutputCostPerToken: 2e-6, CacheReadInputTokenCost: 1e-8},
		"deepseek-v4-pro":              {InputCostPerToken: 1e-6, OutputCostPerToken: 2e-6, CacheReadInputTokenCost: 1e-8},
		"deepseek-v4-flash-vision-exp": {InputCostPerToken: 1e-6, OutputCostPerToken: 2e-6, CacheReadInputTokenCost: 1e-8},
		"deepseek-chat":                {InputCostPerToken: 1e-6, OutputCostPerToken: 2e-6, CacheReadInputTokenCost: 1e-8},
		"deepseek-reasoner":            {InputCostPerToken: 1e-6, OutputCostPerToken: 2e-6, CacheReadInputTokenCost: 1e-8},
	}}
	bs := NewBillingService(&config.Config{}, pricingSvc)

	tests := []struct {
		model                    string
		input, output, cacheRead float64
	}{
		// 2026-09-10 官方降价后：deepseek-flash（V4.1-Flash 新名）与旧名
		// deepseek-v4-flash 同按 Flash 新价。
		{"deepseek-flash", 3e-7, 1.2e-6, 6e-9},
		{"deepseek-v4-flash", 3e-7, 1.2e-6, 6e-9},
		{"deepseek-v4-flash-vision-exp", 3e-7, 1.2e-6, 6e-9},
		// 已停服的 chat/reasoner：即使 JSON 有旧条目也按 flash 价兜底。
		{"deepseek-chat", 3e-7, 1.2e-6, 6e-9},
		{"deepseek-reasoner", 3e-7, 1.2e-6, 6e-9},
	}
	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			// Flash 默认价格不随日期切换。
			pricing, err := bs.GetModelPricing(tt.model)
			require.NoError(t, err)
			require.InDelta(t, tt.input, pricing.InputPricePerToken, 1e-15)
			require.InDelta(t, tt.output, pricing.OutputPricePerToken, 1e-15)
			require.InDelta(t, tt.cacheRead, pricing.CacheReadPricePerToken, 1e-15)
			// 固定时点 2026-10-01 复核：仍走 Flash 新价。
			atPricing, err := bs.getModelPricingAt(tt.model, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC))
			require.NoError(t, err)
			require.InDelta(t, tt.input, atPricing.InputPricePerToken, 1e-15)
			require.InDelta(t, tt.output, atPricing.OutputPricePerToken, 1e-15)
			require.InDelta(t, tt.cacheRead, atPricing.CacheReadPricePerToken, 1e-15)
			require.True(t, bs.HasIdentifiedTokenPricing(tt.model))
		})
	}

	// pro 档（含版本化名称）：断言经固定时点的 getModelPricingAt，不依赖墙上时钟。
	// 官方确认 2026-09-14 后继续提供 Pro，计费方式保持不变。
	for _, model := range []string{"deepseek-v4-pro", "deepseek-v4-pro-0813"} {
		t.Run(model+"/after-september-14", func(t *testing.T) {
			pricing, err := bs.getModelPricingAt(model, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC))
			require.NoError(t, err)
			require.InDelta(t, 1.32e-6, pricing.InputPricePerToken, 1e-15)
			require.InDelta(t, 3.96e-6, pricing.OutputPricePerToken, 1e-15)
			require.InDelta(t, 4.4e-8, pricing.CacheReadPricePerToken, 1e-15)
		})
	}

	// 原计划切换时点前后均保持 Pro 价。
	proBefore, err := bs.getModelPricingAt("deepseek-v4-pro", time.Date(2026, 9, 14, 3, 59, 59, 0, time.UTC))
	require.NoError(t, err)
	require.InDelta(t, 1.32e-6, proBefore.InputPricePerToken, 1e-15)
	require.InDelta(t, 3.96e-6, proBefore.OutputPricePerToken, 1e-15)
	require.InDelta(t, 4.4e-8, proBefore.CacheReadPricePerToken, 1e-15)
	proAtCutoff, err := bs.getModelPricingAt("deepseek-v4-pro", time.Date(2026, 9, 14, 4, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.InDelta(t, 6.6e-7, proAtCutoff.InputPricePerToken, 1e-15)
	require.InDelta(t, 1.98e-6, proAtCutoff.OutputPricePerToken, 1e-15)
	require.InDelta(t, 2.2e-8, proAtCutoff.CacheReadPricePerToken, 1e-15)

	// 版本化名称（不在 JSON / fallbackPrices 精确表中）：按子串归档计价。
	// flash-0731 归 flash 档，GetModelPricing 断言稳定。
	versioned := []struct {
		model                    string
		input, output, cacheRead float64
	}{
		{"deepseek-v4-flash-0731", 3e-7, 1.2e-6, 6e-9},
	}
	for _, tt := range versioned {
		t.Run(tt.model, func(t *testing.T) {
			pricing, err := bs.GetModelPricing(tt.model)
			require.NoError(t, err)
			require.InDelta(t, tt.input, pricing.InputPricePerToken, 1e-15)
			require.InDelta(t, tt.output, pricing.OutputPricePerToken, 1e-15)
			require.InDelta(t, tt.cacheRead, pricing.CacheReadPricePerToken, 1e-15)
		})
	}
}

func TestGetModelPricing_UnknownDeepseekMapsToFlash(t *testing.T) {
	// JSON 含 $0 占位条目（如旧 deepseek-v3-2-251201）：未知 deepseek-* 不再
	// fail-closed，统一按 flash 价兜底（3e-7/1.2e-6/6e-9），不得按 $0 计费。
	pricingSvc := &PricingService{pricingData: map[string]*LiteLLMModelPricing{
		"deepseek-v3-2-251201": {InputCostPerToken: 0, OutputCostPerToken: 0},
	}}
	bs := NewBillingService(&config.Config{}, pricingSvc)

	for _, m := range []string{"deepseek-v3-2-251201", "deepseek-chat", "deepseek-reasoner", "deepseek-foo"} {
		t.Run(m, func(t *testing.T) {
			pricing, err := bs.GetModelPricing(m)
			require.NoError(t, err)
			require.InDelta(t, 3e-7, pricing.InputPricePerToken, 1e-15)
			require.InDelta(t, 1.2e-6, pricing.OutputPricePerToken, 1e-15)
			require.InDelta(t, 6e-9, pricing.CacheReadPricePerToken, 1e-15)
		})
	}
}

// ---------------------------------------------------------------------------
// 本地兜底 JSON：无 $0 占位条目，官方模型价格为官方低谷价
// ---------------------------------------------------------------------------

func TestDeepseekPricingFileMatchesOfficialRates(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "resources", "model-pricing", "model_prices_and_context_window.json"))
	require.NoError(t, err)

	pricingSvc := &PricingService{}
	pricingData, err := pricingSvc.parsePricingData(data)
	require.NoError(t, err)

	_, ok := pricingData["deepseek-v3-2-251201"]
	require.False(t, ok, "deepseek-v3-2-251201（$0 占位条目）必须从价格表中移除")
	for _, discontinued := range []string{"deepseek-chat", "deepseek-reasoner"} {
		_, ok := pricingData[discontinued]
		require.False(t, ok, "%s 已停止服务，必须从价格表中移除", discontinued)
	}

	tests := []struct {
		model                    string
		input, output, cacheRead float64
	}{
		{"deepseek-flash", 1.5e-7, 6e-7, 3e-9},
		{"deepseek-v4-flash", 1.5e-7, 6e-7, 3e-9},
		{"deepseek-v4-flash-vision-exp", 1.5e-7, 6e-7, 3e-9},
		{"deepseek-v4-pro", 6.6e-7, 1.98e-6, 2.2e-8},
	}
	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			entry, ok := pricingData[tt.model]
			require.True(t, ok, "model %s must exist in pricing file", tt.model)
			require.InDelta(t, tt.input, entry.InputCostPerToken, 1e-15)
			require.InDelta(t, tt.output, entry.OutputCostPerToken, 1e-15)
			require.InDelta(t, tt.cacheRead, entry.CacheReadInputTokenCost, 1e-15)
		})
	}
}

// ---------------------------------------------------------------------------
// 2026-09-10 官方降价：deepseek-flash（V4.1-Flash 新名）与旧名同价；
// 官方确认 2026-09-14 后 deepseek-v4-pro 继续按 Pro 价计费
// ---------------------------------------------------------------------------

func TestCalculateCostUnified_DeepseekFlashAndLegacyFlashShareNewRates(t *testing.T) {
	bs := newTestBillingService()
	resolver := NewModelPricingResolver(nil, bs)

	tokens := UsageTokens{InputTokens: 1000, OutputTokens: 500, CacheReadTokens: 1000}
	// 2026-09-10 官方高峰价：1000*3e-7 + 500*1.2e-6 + 1000*6e-9 = 9.06e-4
	peakTotal := 1000*3e-7 + 500*1.2e-6 + 1000*6e-9

	// deepseek-flash 与 deepseek-v4-flash 都取 Flash 新价。
	// 时点取 2026-09-14（周一）12:00 UTC 低谷，峰谷倍率不影响断言。
	for _, model := range []string{"deepseek-flash", "deepseek-v4-flash"} {
		cost, err := bs.CalculateCostUnified(CostInput{
			Ctx: context.Background(), Model: model, Tokens: tokens,
			RateMultiplier: 1.0, Resolver: resolver,
			PricingAt: time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC),
		})
		require.NoError(t, err)
		require.InDelta(t, peakTotal, cost.TotalCost, 1e-10, "model %s must use new flash rates", model)
	}
}

func TestCalculateCostUnified_DeepseekProRetainsRatesAfterSeptember14(t *testing.T) {
	bs := newTestBillingService()
	resolver := NewModelPricingResolver(nil, bs)

	tokens := UsageTokens{InputTokens: 1000, OutputTokens: 500, CacheReadTokens: 1000}
	proTotal := 1000*6.6e-7 + 500*1.98e-6 + 1000*2.2e-8

	// 原计划切换时点之前（2026-09-13 周日，北京周末全天低谷）：按 Pro 价。
	before, err := bs.CalculateCostUnified(CostInput{
		Ctx: context.Background(), Model: "deepseek-v4-pro", Tokens: tokens,
		RateMultiplier: 1.0, Resolver: resolver,
		PricingAt: time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	require.InDelta(t, proTotal, before.TotalCost, 1e-10,
		"deepseek-v4-pro must use Pro rates before September 14")

	// 到达原计划切换时点（2026-09-14 04:00 UTC 整，周一低谷）：仍按 Pro 价。
	after, err := bs.CalculateCostUnified(CostInput{
		Ctx: context.Background(), Model: "deepseek-v4-pro", Tokens: tokens,
		RateMultiplier: 1.0, Resolver: resolver,
		PricingAt: time.Date(2026, 9, 14, 4, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	require.InDelta(t, proTotal, after.TotalCost, 1e-10,
		"deepseek-v4-pro must retain Pro rates on September 14")

	// 版本化名称同口径：deepseek-v4-pro-0813 也保持 Pro 价。
	versioned, err := bs.CalculateCostUnified(CostInput{
		Ctx: context.Background(), Model: "deepseek-v4-pro-0813", Tokens: tokens,
		RateMultiplier: 1.0, Resolver: resolver,
		PricingAt: time.Date(2026, 9, 14, 4, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	require.InDelta(t, proTotal, versioned.TotalCost, 1e-10,
		"versioned pro names must retain Pro rates on September 14")
}
