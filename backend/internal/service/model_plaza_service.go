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
// gateway. GatewayService implements this narrow contract.
type PlazaModelSource interface {
	GetAvailableModels(ctx context.Context, groupID *int64, platform string) []string
	GetSchedulablePlatforms(ctx context.Context, groupID *int64) map[string]struct{}
}

// ModelPlazaService builds the public catalog from schedulable groups and
// models. Pricing enriches those models but never determines membership.
type ModelPlazaService struct {
	groupRepo      GroupRepository
	channelService *ChannelService
	modelSource    PlazaModelSource
}

func NewModelPlazaService(
	groupRepo GroupRepository,
	channelService *ChannelService,
	gatewayService *GatewayService,
) *ModelPlazaService {
	return &ModelPlazaService{
		groupRepo:      groupRepo,
		channelService: channelService,
		modelSource:    gatewayService,
	}
}

// ListGroups returns active groups backed by schedulable accounts. Model
// membership follows gateway scheduling configuration, with platform defaults
// used when schedulable accounts have no explicit model mapping.
func (s *ModelPlazaService) ListGroups(ctx context.Context) ([]PlazaGroup, error) {
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
			models = append(models, PlazaModel{
				Name:            available.name,
				Platform:        available.platform,
				Pricing:         s.resolvePricing(ctx, group, available),
				OfficialPricing: s.lookupOfficialPricing(available.name, officialMemo),
			})
		}

		out = append(out, PlazaGroup{
			ID:                   group.ID,
			Name:                 group.Name,
			Description:          group.Description,
			Platform:             group.Platform,
			SubscriptionType:     group.SubscriptionType,
			RateMultiplier:       group.RateMultiplier,
			PeakRateEnabled:      group.PeakRateEnabled,
			PeakStart:            group.PeakStart,
			PeakEnd:              group.PeakEnd,
			PeakRateMultiplier:   group.PeakRateMultiplier,
			IsExclusive:          group.IsExclusive,
			ImageRateIndependent: group.ImageRateIndependent,
			ImageRateMultiplier:  group.ImageRateMultiplier,
			Models:               models,
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
	s.channelService.fillGlobalPricingFallback(models)
	if pricingNeedsFallback(models[0].Pricing) {
		return nil
	}
	return plazaImageDisplayPricing(models[0].Pricing, group)
}

func (s *ModelPlazaService) lookupOfficialPricing(
	modelName string,
	memo map[string]*PlazaOfficialPricing,
) *PlazaOfficialPricing {
	if s.channelService == nil {
		return nil
	}
	return s.channelService.lookupOfficialPricing(modelName, memo)
}
