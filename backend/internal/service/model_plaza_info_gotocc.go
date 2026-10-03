package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/modelcatalog"
)

// 模型广场的模型官方信息与近 24 小时状态（GoToCC）。官方信息来自随版打包的 models.dev 快照
// （internal/pkg/modelcatalog），后台按模型填写的简介、厂商和用途优先；不影响模型是否展示，也不影响计费。

// SettingKeyGotoCCModelPlazaOverrides 保存后台按模型填写的展示信息，值为 JSON 对象，键是小写模型名。
const SettingKeyGotoCCModelPlazaOverrides = "gotocc_model_plaza_overrides"

const (
	// plazaStatsWindow 是模型状态的统计窗口。
	plazaStatsWindow = 24 * time.Hour
	// plazaStatsTTL 是统计结果的缓存时长，公开页面频繁访问时不重复扫描用量表。
	plazaStatsTTL = time.Minute
)

// PlazaModelInfo 是模型广场展示的模型信息。Source 为 models.dev 时表示官方目录中有该模型。
type PlazaModelInfo struct {
	Vendor            string   `json:"vendor"`
	VendorName        string   `json:"vendor_name"`
	DisplayName       string   `json:"display_name"`
	Description       string   `json:"description"`
	CustomDescription bool     `json:"custom_description"`
	Purposes          []string `json:"purposes"`
	ContextWindow     int64    `json:"context_window"`
	MaxOutputTokens   int64    `json:"max_output_tokens"`
	InputModalities   []string `json:"input_modalities"`
	OutputModalities  []string `json:"output_modalities"`
	Reasoning         bool     `json:"reasoning"`
	ToolCall          bool     `json:"tool_call"`
	StructuredOutput  bool     `json:"structured_output"`
	Attachment        bool     `json:"attachment"`
	OpenWeights       bool     `json:"open_weights"`
	Knowledge         string   `json:"knowledge"`
	ReleaseDate       string   `json:"release_date"`
	Source            string   `json:"source"`
}

// PlazaModelStats 是全站近 24 小时的模型请求概况；SuccessRate 为百分比。
type PlazaModelStats struct {
	Requests        int64    `json:"requests"`
	Errors          int64    `json:"errors"`
	SuccessRate     *float64 `json:"success_rate"`
	AvgFirstTokenMs *float64 `json:"avg_first_token_ms"`
}

// PlazaModelOverride 是后台为单个模型填写的展示信息，非空字段覆盖官方目录的值。
type PlazaModelOverride struct {
	Description string   `json:"description"`
	Vendor      string   `json:"vendor"`
	Purposes    []string `json:"purposes"`
}

type plazaModelStatsFetcher interface {
	GetPlazaModelStats(ctx context.Context, since time.Time) (map[string]PlazaModelStats, error)
}

var plazaStatsCache struct {
	sync.Mutex
	at    time.Time
	stats map[string]PlazaModelStats
}

// GetModelPlazaOverrides 读取后台填写的模型展示信息；未设置时为空。
func (s *SettingService) GetModelPlazaOverrides(ctx context.Context) (map[string]PlazaModelOverride, error) {
	values, err := s.settingRepo.GetMultiple(ctx, []string{SettingKeyGotoCCModelPlazaOverrides})
	if err != nil {
		return nil, err
	}
	overrides := map[string]PlazaModelOverride{}
	raw := values[SettingKeyGotoCCModelPlazaOverrides]
	if raw == "" {
		return overrides, nil
	}
	if err := json.Unmarshal([]byte(raw), &overrides); err != nil {
		return nil, fmt.Errorf("decode model plaza overrides: %w", err)
	}
	return overrides, nil
}

// SetModelPlazaOverrides 整体保存模型展示信息，模型名统一小写，全部字段为空的条目不保存。
func (s *SettingService) SetModelPlazaOverrides(ctx context.Context, overrides map[string]PlazaModelOverride) error {
	cleaned := make(map[string]PlazaModelOverride, len(overrides))
	for name, override := range overrides {
		key := strings.ToLower(strings.TrimSpace(name))
		override.Description = strings.TrimSpace(override.Description)
		override.Vendor = strings.TrimSpace(override.Vendor)
		if key == "" || (override.Description == "" && override.Vendor == "" && len(override.Purposes) == 0) {
			continue
		}
		cleaned[key] = override
	}
	encoded, err := json.Marshal(cleaned)
	if err != nil {
		return err
	}
	return s.settingRepo.Set(ctx, SettingKeyGotoCCModelPlazaOverrides, string(encoded))
}

// EnrichModels 为广场分组中的每个模型补充官方信息、后台覆盖和近 24 小时状态；
// 官方目录有 token 单价的模型，官方参考价改用目录价（含上下文分档）。
func (s *ModelPlazaService) EnrichModels(ctx context.Context, groups []PlazaGroup, settings *SettingService) error {
	overrides, err := settings.GetModelPlazaOverrides(ctx)
	if err != nil {
		return err
	}
	stats, err := s.plazaModelStats(ctx)
	if err != nil {
		return err
	}
	for gi := range groups {
		for mi := range groups[gi].Models {
			model := &groups[gi].Models[mi]
			key := strings.ToLower(model.Name)
			entry, found := modelcatalog.Lookup(model.Name)
			info := plazaModelInfo(model, entry, found, overrides[key])
			model.Info = &info
			if found && entry.Model.Cost != nil && entry.Model.Cost.Input != nil {
				model.OfficialPricing = plazaOfficialFromCatalog(entry.Model.Cost)
			}
			if stat, ok := stats[key]; ok {
				model.Stats = &stat
			}
		}
	}
	return nil
}

// plazaModelStats 读取近 24 小时各模型的请求概况，结果缓存一分钟。
func (s *ModelPlazaService) plazaModelStats(ctx context.Context) (map[string]PlazaModelStats, error) {
	plazaStatsCache.Lock()
	defer plazaStatsCache.Unlock()
	if plazaStatsCache.stats != nil && time.Since(plazaStatsCache.at) < plazaStatsTTL {
		return plazaStatsCache.stats, nil
	}
	gateway, ok := s.modelSource.(*GatewayService)
	if !ok {
		return map[string]PlazaModelStats{}, nil
	}
	fetcher, ok := gateway.usageLogRepo.(plazaModelStatsFetcher)
	if !ok {
		return map[string]PlazaModelStats{}, nil
	}
	stats, err := fetcher.GetPlazaModelStats(ctx, time.Now().Add(-plazaStatsWindow))
	if err != nil {
		return nil, fmt.Errorf("plaza model stats: %w", err)
	}
	plazaStatsCache.at = time.Now()
	plazaStatsCache.stats = stats
	return stats, nil
}

func plazaModelInfo(model *PlazaModel, entry modelcatalog.Entry, found bool, override PlazaModelOverride) PlazaModelInfo {
	info := PlazaModelInfo{Vendor: model.Platform, Purposes: plazaPurposesFromPricing(model)}
	if found {
		m := entry.Model
		info.Vendor = entry.ProviderID
		info.VendorName = entry.ProviderName
		info.DisplayName = m.Name
		info.Description = m.Description
		info.Purposes = plazaPurposesFromModalities(m.Modalities.Output)
		info.ContextWindow = m.Limit.Context
		info.MaxOutputTokens = m.Limit.Output
		info.InputModalities = m.Modalities.Input
		info.OutputModalities = m.Modalities.Output
		info.Reasoning = m.Reasoning
		info.ToolCall = m.ToolCall
		info.StructuredOutput = m.StructuredOutput
		info.Attachment = m.Attachment
		info.OpenWeights = m.OpenWeights
		info.Knowledge = m.Knowledge
		info.ReleaseDate = m.ReleaseDate
		info.Source = "models.dev"
	}
	if override.Description != "" {
		info.Description = override.Description
		info.CustomDescription = true
	}
	if override.Vendor != "" && override.Vendor != info.Vendor {
		info.Vendor = override.Vendor
		info.VendorName = ""
	}
	if len(override.Purposes) > 0 {
		info.Purposes = override.Purposes
	}
	return info
}

// 官方目录没有该模型时，用途按计费方式和平台判断：图片计费为生图，视频计费或视频平台为视频，其余为语言。
func plazaPurposesFromPricing(model *PlazaModel) []string {
	if model.Pricing != nil && model.Pricing.BillingMode == BillingModeImage {
		return []string{"image"}
	}
	if model.Platform == "video" || (model.Pricing != nil && model.Pricing.BillingMode == BillingModeVideo) {
		return []string{"video"}
	}
	return []string{"language"}
}

// 用途按官方目录的输出模态：文本为语言，其余模态同名。
func plazaPurposesFromModalities(outputs []string) []string {
	purposes := make([]string, 0, len(outputs))
	for _, output := range outputs {
		purpose := output
		if output == "text" {
			purpose = "language"
		}
		purposes = append(purposes, purpose)
	}
	return purposes
}

// plazaOfficialFromCatalog 把目录的美元 / 百万 token 单价换成每 token 单价；有上下文分档时生成区间，
// 首档为基础价，之后每档从其阈值开始。
func plazaOfficialFromCatalog(cost *modelcatalog.Cost) *PlazaOfficialPricing {
	official := &PlazaOfficialPricing{
		InputPrice:      plazaPerToken(cost.Input),
		OutputPrice:     plazaPerToken(cost.Output),
		CacheWritePrice: plazaPerToken(cost.CacheWrite),
		CacheReadPrice:  plazaPerToken(cost.CacheRead),
	}
	if len(cost.Tiers) == 0 {
		return official
	}
	lower := 0
	input, output, write, read := cost.Input, cost.Output, cost.CacheWrite, cost.CacheRead
	for _, tier := range cost.Tiers {
		upper := int(tier.ContextOver)
		official.Intervals = append(official.Intervals, PricingInterval{
			MinTokens:       lower,
			MaxTokens:       &upper,
			TierLabel:       "≤" + plazaTokenLabel(upper),
			InputPrice:      plazaPerToken(input),
			OutputPrice:     plazaPerToken(output),
			CacheWritePrice: plazaPerToken(write),
			CacheReadPrice:  plazaPerToken(read),
		})
		lower = upper
		input, output, write, read = tier.Input, tier.Output, tier.CacheWrite, tier.CacheRead
	}
	official.Intervals = append(official.Intervals, PricingInterval{
		MinTokens:       lower,
		TierLabel:       ">" + plazaTokenLabel(lower),
		InputPrice:      plazaPerToken(input),
		OutputPrice:     plazaPerToken(output),
		CacheWritePrice: plazaPerToken(write),
		CacheReadPrice:  plazaPerToken(read),
	})
	return official
}

func plazaPerToken(perMillion *float64) *float64 {
	if perMillion == nil {
		return nil
	}
	value := *perMillion / 1_000_000
	return &value
}

func plazaTokenLabel(tokens int) string {
	if tokens%1_000_000 == 0 {
		return fmt.Sprintf("%dM", tokens/1_000_000)
	}
	return fmt.Sprintf("%dK", tokens/1_000)
}
