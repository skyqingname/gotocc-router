package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

var plazaConcretePlatforms = []string{
	PlatformAnthropic,
	PlatformGemini,
	PlatformOpenAI,
	PlatformAntigravity,
	PlatformGrok,
}

// PlazaModelSource exposes the same group model inventory used by the live
// gateway. Pricing enriches that inventory but never determines membership.
type PlazaModelSource interface {
	GetAvailableModels(ctx context.Context, groupID *int64, platform string) []string
	GetSchedulablePlatforms(ctx context.Context, groupID *int64) map[string]struct{}
}

// PlazaOfficialPricing 模型广场展示用的官方参考价（USD per token），与计费同源：
// LiteLLM → 内置兜底价卡 → 模型策略。字段为 nil 表示该项缺失（0 视为未配置）。
type PlazaOfficialPricing struct {
	InputPrice        *float64
	OutputPrice       *float64
	CacheWritePrice   *float64 // 5m 缓存写入（= LiteLLM cache_creation）
	CacheWrite1hPrice *float64 // 1h 缓存写入，仅计费会区分 5m/1h 时给出
	CacheReadPrice    *float64
	// Intervals 官方长上下文阶梯（多档时给出），不受分组开关影响。
	Intervals []PricingInterval
}

// PlazaModel 模型广场中单个模型条目：按实收口径合成的展示定价 + 官方参考价。
type PlazaModel struct {
	Name            string
	Platform        string
	Pricing         *ChannelModelPricing
	OfficialPricing *PlazaOfficialPricing
	// LongContextBasis 多档时的计价基准（整单 / 仅超出部分），单档为空。
	LongContextBasis ContextPricingBasis
	// TimePricing 计费会生效的分时倍率时段；无分时为 nil。
	TimePricing *TimePricingSchedule
}

// PlazaGroup 模型广场中以分组为顶层的条目。
//
// 与 AvailableGroupRef 相比多了 Description 与 Models；Models 来自该分组关联渠道的
// 支持模型（普通分组按分组平台隔离，Composite 分组展开关联渠道已配置的
// 具体平台），与「可用渠道」页口径一致。
type PlazaGroup struct {
	ID                 int64
	Name               string
	Description        string
	Platform           string
	SubscriptionType   string
	RateMultiplier     float64
	PeakRateEnabled    bool
	PeakStart          string
	PeakEnd            string
	PeakRateMultiplier float64
	IsExclusive        bool
	// 图片按次实付倍率：ImageRateIndependent 为 true 时，图片计费模型的实付
	// = 档位价 × ImageRateMultiplier，不乘分组/用户专属倍率（与计费口径一致）。
	ImageRateIndependent bool
	ImageRateMultiplier  float64
	// LongContextPricingEnabled 分组是否按上下文长度应用阶梯价；关闭时模型展示的是最低档。
	LongContextPricingEnabled bool
	Models                    []PlazaModel
}

// ModelPlazaService 聚合模型广场数据。
//
// 模型枚举来自渠道配置；token 模型的展示单价与阶梯由 BillingService 的阶梯表
// 查询给出（与扣费走同一条解析链与计费函数），图片/按次模型沿用渠道/分组档位价。
type ModelPlazaService struct {
	channelRepo    ChannelRepository
	groupRepo      GroupRepository
	pricingService *PricingService
	billingService *BillingService
	resolver       *ModelPricingResolver
	channelService *ChannelService
	modelSource    PlazaModelSource
}

// NewModelPlazaService 创建模型广场服务。
func NewModelPlazaService(
	channelRepo ChannelRepository,
	groupRepo GroupRepository,
	pricingService *PricingService,
	billingService *BillingService,
	resolver *ModelPricingResolver,
	gatewayService *GatewayService,
) *ModelPlazaService {
	var channelService *ChannelService
	if resolver != nil {
		channelService = resolver.channelService
	}
	var modelSource PlazaModelSource
	if gatewayService != nil {
		modelSource = gatewayService
	}
	return &ModelPlazaService{
		channelRepo:    channelRepo,
		groupRepo:      groupRepo,
		pricingService: pricingService,
		billingService: billingService,
		resolver:       resolver,
		channelService: channelService,
		modelSource:    modelSource,
	}
}

// ListGroups 返回模型广场数据：每个活跃分组附带其可用模型与定价。
//
// 模型枚举口径与 ListAvailable 一致（Active 渠道、SupportedModels ∪ 全局定价回落、
// 平台隔离），仅把顶层从渠道换成分组：
//   - 渠道按 lower(name) 排序后遍历，保证同名模型去重结果确定；
//   - 同分组同名模型「先见者胜」，仅当已存条目无定价而新条目有定价时升级替换；
//   - token 模型的单价与阶梯按实收口径合成（见 ResolveContextPricingSchedule），
//     图片计费模型的档位价按实收口径合成（见 plazaImageDisplayPricing）；
//   - 每个模型附带官方参考价（查不到为 nil）；
//   - 只返回 Models 非空的分组；分组按 RateMultiplier 升序（同倍率按名称），
//     组内模型按名称排序。
//
// 可见性过滤（专属分组）不在此层做，由 handler 按登录态裁剪。
func (s *ModelPlazaService) listGroupsFromChannelPricing(ctx context.Context) ([]PlazaGroup, error) {
	channels, err := s.channelRepo.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("list channels: %w", err)
	}
	groups, err := s.groupRepo.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active groups: %w", err)
	}

	sort.SliceStable(channels, func(i, j int) bool {
		return strings.ToLower(channels[i].Name) < strings.ToLower(channels[j].Name)
	})

	byGroup := make(map[int64]*PlazaGroup, len(groups))
	groupEnt := make(map[int64]*Group, len(groups))
	order := make([]int64, 0, len(groups))
	for i := range groups {
		g := &groups[i]
		byGroup[g.ID] = &PlazaGroup{
			ID:                        g.ID,
			Name:                      g.Name,
			Description:               g.Description,
			Platform:                  g.Platform,
			SubscriptionType:          g.SubscriptionType,
			RateMultiplier:            g.RateMultiplier,
			PeakRateEnabled:           g.PeakRateEnabled,
			PeakStart:                 g.PeakStart,
			PeakEnd:                   g.PeakEnd,
			PeakRateMultiplier:        g.PeakRateMultiplier,
			IsExclusive:               g.IsExclusive,
			ImageRateIndependent:      g.ImageRateIndependent,
			ImageRateMultiplier:       g.ImageRateMultiplier,
			LongContextPricingEnabled: g.LongContextPricingEnabled,
		}
		groupEnt[g.ID] = g
		order = append(order, g.ID)
	}

	type modelKey struct {
		platform string
		name     string
	}
	// modelIdx[groupID][platform+modelName] = index into byGroup[groupID].Models
	modelIdx := make(map[int64]map[modelKey]int, len(groups))
	for i := range channels {
		ch := &channels[i]
		if ch.Status != StatusActive {
			continue
		}
		ch.normalizeBillingModelSource()
		supported := ch.SupportedModels()
		fillGlobalPricingFallback(s.pricingService, supported)

		for _, gid := range ch.GroupIDs {
			pg, ok := byGroup[gid]
			if !ok {
				continue
			}
			idx := modelIdx[gid]
			if idx == nil {
				idx = make(map[modelKey]int, len(supported))
				modelIdx[gid] = idx
			}
			for j := range supported {
				m := supported[j]
				if pg.Platform == PlatformComposite {
					if !isConcreteRequestPlatform(m.Platform) {
						continue
					}
				} else if m.Platform != pg.Platform {
					continue
				}
				key := modelKey{platform: m.Platform, name: m.Name}
				if at, seen := idx[key]; seen {
					// 先见者胜；仅当已存条目无定价而新条目有定价时升级。
					if pg.Models[at].Pricing == nil && m.Pricing != nil {
						pg.Models[at].Pricing = m.Pricing
					}
					continue
				}
				idx[key] = len(pg.Models)
				pg.Models = append(pg.Models, PlazaModel{
					Name:     m.Name,
					Platform: m.Platform,
					Pricing:  m.Pricing,
				})
			}
		}
	}

	officialMemo := make(map[string]*PlazaOfficialPricing)
	out := make([]PlazaGroup, 0, len(order))
	for _, gid := range order {
		pg := byGroup[gid]
		if len(pg.Models) == 0 {
			continue
		}
		sort.SliceStable(pg.Models, func(i, j int) bool {
			if pg.Models[i].Name != pg.Models[j].Name {
				return pg.Models[i].Name < pg.Models[j].Name
			}
			return pg.Models[i].Platform < pg.Models[j].Platform
		})
		g := groupEnt[gid]
		for j := range pg.Models {
			s.fillDisplayPricing(ctx, &pg.Models[j], g)
			pg.Models[j].OfficialPricing = s.lookupOfficialPricing(ctx, pg.Models[j].Name, officialMemo)
		}
		out = append(out, *pg)
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].RateMultiplier != out[j].RateMultiplier {
			return out[i].RateMultiplier < out[j].RateMultiplier
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// ListGroups returns active groups backed by schedulable accounts. Membership
// follows the live gateway inventory; pricing only enriches those models and
// therefore cannot make an unschedulable model visible.
func (s *ModelPlazaService) ListGroups(ctx context.Context) ([]PlazaGroup, error) {
	if s != nil && s.modelSource == nil {
		return s.listGroupsFromChannelPricing(ctx)
	}
	if s == nil || s.groupRepo == nil {
		return nil, fmt.Errorf("model plaza group repository is unavailable")
	}

	groups, err := s.groupRepo.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active groups: %w", err)
	}

	officialMemo := make(map[string]*PlazaOfficialPricing)
	out := make([]PlazaGroup, 0, len(groups))
	for i := range groups {
		group := &groups[i]
		if group.ActiveAccountCount <= 0 {
			continue
		}

		availableModels := s.resolveModels(ctx, group)
		if len(availableModels) == 0 {
			continue
		}

		models := make([]PlazaModel, 0, len(availableModels))
		for _, available := range availableModels {
			model := PlazaModel{
				Name:     available.name,
				Platform: available.platform,
				Pricing:  s.resolvePricing(ctx, group, available),
			}
			s.fillDisplayPricing(ctx, &model, group)
			model.OfficialPricing = s.lookupOfficialPricing(ctx, available.name, officialMemo)
			models = append(models, model)
		}

		out = append(out, PlazaGroup{
			ID:                        group.ID,
			Name:                      group.Name,
			Description:               group.Description,
			Platform:                  group.Platform,
			SubscriptionType:          group.SubscriptionType,
			RateMultiplier:            group.RateMultiplier,
			PeakRateEnabled:           group.PeakRateEnabled,
			PeakStart:                 group.PeakStart,
			PeakEnd:                   group.PeakEnd,
			PeakRateMultiplier:        group.PeakRateMultiplier,
			IsExclusive:               group.IsExclusive,
			ImageRateIndependent:      group.ImageRateIndependent,
			ImageRateMultiplier:       group.ImageRateMultiplier,
			LongContextPricingEnabled: group.LongContextPricingEnabled,
			Models:                    models,
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].RateMultiplier != out[j].RateMultiplier {
			return out[i].RateMultiplier < out[j].RateMultiplier
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

// fillDisplayPricing 把模型的展示定价换成实收口径：
// token 模型取计费阶梯表（单价与档位均由真实计费函数得出），
// 图片/按次模型（或阶梯表不可用时）沿用渠道定价与分组图片档位价。
func (s *ModelPlazaService) fillDisplayPricing(ctx context.Context, m *PlazaModel, g *Group) {
	if s.billingService != nil && s.resolver != nil {
		sched, err := s.billingService.ResolveContextPricingSchedule(ctx, s.resolver, ContextPricingScheduleInput{
			Model:    m.Name,
			Group:    g,
			Platform: m.Platform,
		})
		if err == nil && sched != nil && len(sched.Tiers) > 0 {
			m.Pricing = plazaPricingFromSchedule(m.Pricing, sched)
			if len(sched.Tiers) > 1 {
				m.LongContextBasis = sched.Basis
			}
			m.TimePricing = sched.TimePricing
			return
		}
	}
	m.Pricing = plazaImageDisplayPricing(m.Pricing, g)
}

// plazaPricingFromSchedule 把阶梯表压成展示用的 ChannelModelPricing：
// 平价取首档单价，多档时 Intervals 逐档给出绝对单价；图片/按次字段沿用原始定价。
func plazaPricingFromSchedule(raw *ChannelModelPricing, sched *ContextPricingSchedule) *ChannelModelPricing {
	out := &ChannelModelPricing{BillingMode: BillingModeToken}
	if raw != nil {
		out.ImageInputPrice = raw.ImageInputPrice
		out.ImageOutputPrice = raw.ImageOutputPrice
		out.PerRequestPrice = raw.PerRequestPrice
	}
	first := sched.Tiers[0]
	out.InputPrice = first.Input
	out.OutputPrice = first.Output
	out.CacheWritePrice = first.CacheWrite
	out.CacheReadPrice = first.CacheRead
	if len(sched.Tiers) > 1 {
		out.Intervals = plazaIntervalsFromTiers(sched.Tiers)
	}
	return out
}

func plazaIntervalsFromTiers(tiers []ContextPricingTier) []PricingInterval {
	intervals := make([]PricingInterval, 0, len(tiers))
	for i, t := range tiers {
		intervals = append(intervals, PricingInterval{
			MinTokens:       t.MinTokens,
			MaxTokens:       t.MaxTokens,
			TierLabel:       t.Label,
			InputPrice:      t.Input,
			OutputPrice:     t.Output,
			CacheWritePrice: t.CacheWrite,
			CacheReadPrice:  t.CacheRead,
			SortOrder:       i,
		})
	}
	return intervals
}

// plazaImageDisplayPricing 为图片计费模型合成展示定价，使档位价与实收口径一致：
// 每档（1K/2K/4K）单价 = 分组图片价 > 渠道同档位价 > 渠道默认按次价，无价的档不展示。
// 分组未配任何图片价、或定价非图片模式时原样返回。返回克隆，不修改入参
// （渠道定价指针指向缓存共享数据）。
func plazaImageDisplayPricing(p *ChannelModelPricing, g *Group) *ChannelModelPricing {
	if p == nil || g == nil || p.BillingMode != BillingModeImage {
		return p
	}
	if g.ImagePrice1K == nil && g.ImagePrice2K == nil && g.ImagePrice4K == nil {
		return p
	}
	channelTierPrice := func(label string) *float64 {
		for i := range p.Intervals {
			if p.Intervals[i].TierLabel == label && p.Intervals[i].PerRequestPrice != nil {
				return p.Intervals[i].PerRequestPrice
			}
		}
		return p.PerRequestPrice
	}
	tiers := []struct {
		label      string
		groupPrice *float64
	}{
		{"1K", g.ImagePrice1K},
		{"2K", g.ImagePrice2K},
		{"4K", g.ImagePrice4K},
	}
	clone := *p
	clone.Intervals = make([]PricingInterval, 0, len(tiers))
	for i, t := range tiers {
		price := t.groupPrice
		if price == nil {
			price = channelTierPrice(t.label)
		}
		if price == nil {
			continue
		}
		v := *price
		clone.Intervals = append(clone.Intervals, PricingInterval{
			TierLabel:       t.label,
			PerRequestPrice: &v,
			SortOrder:       i,
		})
	}
	return &clone
}

// lookupOfficialPricing 查询模型的官方参考价（与计费同源：LiteLLM → 内置兜底 → 模型策略），
// 带 memo 避免同名模型重复解析。官方阶梯按无分组、无渠道的口径查阶梯表。
// billingService 为 nil（测试场景）或查不到时返回 nil。
func (s *ModelPlazaService) lookupOfficialPricing(ctx context.Context, modelName string, memo map[string]*PlazaOfficialPricing) *PlazaOfficialPricing {
	if s.billingService == nil {
		return nil
	}
	if cached, ok := memo[modelName]; ok {
		return cached
	}
	var result *PlazaOfficialPricing
	if mp, err := s.billingService.GetModelPricing(modelName); err == nil && mp != nil {
		result = &PlazaOfficialPricing{
			InputPrice:      nonZeroPtr(mp.InputPricePerToken),
			OutputPrice:     nonZeroPtr(mp.OutputPricePerToken),
			CacheWritePrice: nonZeroPtr(mp.CacheCreationPricePerToken),
			CacheReadPrice:  nonZeroPtr(mp.CacheReadPricePerToken),
		}
		// 计费只在支持 5m/1h 分档时使用 1h 价，其余情况 1h 价对用户无意义。
		if mp.SupportsCacheBreakdown {
			result.CacheWrite1hPrice = nonZeroPtr(mp.CacheCreation1hPrice)
		}
		if s.resolver != nil {
			sched, schedErr := s.billingService.ResolveContextPricingSchedule(ctx, s.resolver, ContextPricingScheduleInput{Model: modelName})
			if schedErr == nil && sched != nil && len(sched.Tiers) > 1 {
				result.Intervals = plazaIntervalsFromTiers(sched.Tiers)
			}
		}
		if result.InputPrice == nil && result.OutputPrice == nil && result.CacheWritePrice == nil &&
			result.CacheWrite1hPrice == nil && result.CacheReadPrice == nil && len(result.Intervals) == 0 {
			result = nil
		}
	}
	memo[modelName] = result
	return result
}

type plazaAvailableModel struct {
	name     string
	platform string
}

func (s *ModelPlazaService) resolveModels(ctx context.Context, group *Group) []plazaAvailableModel {
	if group == nil {
		return nil
	}
	if group.Platform != PlatformComposite {
		return s.resolvePlatformModels(ctx, group.ID, group.Platform, true, nil)
	}

	groupID := group.ID
	schedulable := map[string]struct{}{}
	if s.modelSource != nil {
		schedulable = s.modelSource.GetSchedulablePlatforms(ctx, &groupID)
	}
	seen := make(map[string]struct{})
	models := make([]plazaAvailableModel, 0)
	for _, platform := range plazaConcretePlatforms {
		if _, ok := schedulable[platform]; !ok {
			continue
		}
		models = append(models, s.resolvePlatformModels(ctx, group.ID, platform, true, seen)...)
	}
	sortPlazaAvailableModels(models)
	return models
}

func (s *ModelPlazaService) resolvePlatformModels(
	ctx context.Context,
	groupID int64,
	platform string,
	useDefaults bool,
	sharedSeen map[string]struct{},
) []plazaAvailableModel {
	configured := []string(nil)
	if s.modelSource != nil {
		configured = s.modelSource.GetAvailableModels(ctx, &groupID, platform)
	}
	defaults := defaultModelsListCandidateIDs(platform)
	if len(configured) == 0 && useDefaults {
		configured = defaults
	}

	seen := sharedSeen
	if seen == nil {
		seen = make(map[string]struct{}, len(configured))
	}
	models := make([]plazaAvailableModel, 0, len(configured))
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		key := strings.ToLower(name)
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		models = append(models, plazaAvailableModel{name: name, platform: platform})
	}

	for _, configuredModel := range configured {
		model := strings.TrimSpace(configuredModel)
		prefix, wildcard := splitWildcardSuffix(model)
		if !wildcard {
			add(model)
			continue
		}
		prefix = strings.ToLower(prefix)
		for _, candidate := range defaults {
			if strings.HasPrefix(strings.ToLower(candidate), prefix) {
				add(candidate)
			}
		}
	}

	if sharedSeen == nil {
		sortPlazaAvailableModels(models)
	}
	return models
}

func sortPlazaAvailableModels(models []plazaAvailableModel) {
	sort.SliceStable(models, func(i, j int) bool {
		left := strings.ToLower(models[i].name)
		right := strings.ToLower(models[j].name)
		if left != right {
			return left < right
		}
		return models[i].platform < models[j].platform
	})
}

func (s *ModelPlazaService) resolvePricing(
	ctx context.Context,
	group *Group,
	model plazaAvailableModel,
) *ChannelModelPricing {
	groupPricing := matchGroupModelPricing(group, model.name)
	if !pricingNeedsFallback(groupPricing) {
		return plazaImageDisplayPricing(groupPricing, group)
	}

	var channelPricing *ChannelModelPricing
	if s.channelService != nil {
		pricingCtx := ctx
		if group.Platform == PlatformComposite {
			pricingCtx = WithResolvedTargetPlatform(ctx, model.platform)
		}
		channelPricing = s.channelService.GetChannelModelPricing(pricingCtx, group.ID, model.name)
		if !pricingNeedsFallback(channelPricing) {
			return plazaImageDisplayPricing(channelPricing, group)
		}
	}

	if s.channelService == nil {
		return nil
	}
	fallbackShape := groupPricing
	if fallbackShape == nil {
		fallbackShape = channelPricing
	}
	models := []SupportedModel{{
		Name:     model.name,
		Platform: model.platform,
		Pricing:  fallbackShape,
	}}
	fillGlobalPricingFallback(s.pricingService, models)
	if pricingNeedsFallback(models[0].Pricing) {
		return nil
	}
	return plazaImageDisplayPricing(models[0].Pricing, group)
}
