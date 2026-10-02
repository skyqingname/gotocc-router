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
		ApplyCurrentResellerGroupPricing(ctx, g)
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
			VideoRateIndependent:      g.VideoRateIndependent,
			VideoRateMultiplier:       g.VideoRateMultiplier,
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
