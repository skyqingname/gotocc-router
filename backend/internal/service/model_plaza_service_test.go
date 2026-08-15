//go:build unit

package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type stubPlazaModelSource struct {
	models    map[int64]map[string][]string
	platforms map[int64]map[string]struct{}
}

func (s *stubPlazaModelSource) GetAvailableModels(_ context.Context, groupID *int64, platform string) []string {
	if s == nil || groupID == nil {
		return nil
	}
	return append([]string(nil), s.models[*groupID][platform]...)
}

func (s *stubPlazaModelSource) GetSchedulablePlatforms(_ context.Context, groupID *int64) map[string]struct{} {
	if s == nil || groupID == nil {
		return nil
	}
	out := make(map[string]struct{}, len(s.platforms[*groupID]))
	for platform := range s.platforms[*groupID] {
		out[platform] = struct{}{}
	}
	return out
}

func newModelPlazaServiceForTest(
	groups []Group,
	channels []Channel,
	modelSource PlazaModelSource,
	pricing *PricingService,
) *ModelPlazaService {
	repo := &mockChannelRepository{
		listAllFn: func(context.Context) ([]Channel, error) { return channels, nil },
		getGroupPlatformsFn: func(_ context.Context, groupIDs []int64) (map[int64]string, error) {
			platforms := make(map[int64]string, len(groupIDs))
			for _, groupID := range groupIDs {
				for i := range groups {
					if groups[i].ID == groupID {
						platforms[groupID] = groups[i].Platform
					}
				}
			}
			return platforms, nil
		},
	}
	channelService := NewChannelService(repo, &stubGroupRepoForAvailable{activeGroups: groups}, nil, pricing)
	return &ModelPlazaService{
		groupRepo:      &stubGroupRepoForAvailable{activeGroups: groups},
		channelService: channelService,
		modelSource:    modelSource,
	}
}

func TestModelPlazaService_UsesSchedulableModelsWithoutChannelPricing(t *testing.T) {
	groups := []Group{
		{ID: 1, Name: "GPT", Platform: PlatformOpenAI, ActiveAccountCount: 2, RateMultiplier: 1},
		{ID: 2, Name: "Grok", Platform: PlatformGrok, ActiveAccountCount: 1, RateMultiplier: 0.5},
		{ID: 3, Name: "Stale", Platform: PlatformOpenAI, ActiveAccountCount: 0, RateMultiplier: 0.1},
	}
	source := &stubPlazaModelSource{models: map[int64]map[string][]string{
		1: {PlatformOpenAI: {"gpt-5.6-sol", "GPT-5.6-SOL", "gpt-5.5"}},
		2: {PlatformGrok: {"grok-4.5"}},
		3: {PlatformOpenAI: {"must-not-leak"}},
	}}
	svc := newModelPlazaServiceForTest(groups, nil, source, nil)

	out, err := svc.ListGroups(context.Background())

	require.NoError(t, err)
	require.Len(t, out, 2)
	require.Equal(t, "Grok", out[0].Name)
	require.Equal(t, []string{"grok-4.5"}, plazaModelNames(out[0].Models))
	require.Equal(t, "GPT", out[1].Name)
	require.Equal(t, []string{"gpt-5.5", "gpt-5.6-sol"}, plazaModelNames(out[1].Models))
	for _, model := range out[1].Models {
		require.Nil(t, model.Pricing, "missing pricing must not remove a live model")
	}
}

func TestModelPlazaService_ExpandsWildcardsAndUsesPlatformDefaults(t *testing.T) {
	groups := []Group{
		{ID: 1, Name: "Mapped", Platform: PlatformOpenAI, ActiveAccountCount: 1, RateMultiplier: 1},
		{ID: 2, Name: "Default", Platform: PlatformGrok, ActiveAccountCount: 1, RateMultiplier: 1},
	}
	source := &stubPlazaModelSource{models: map[int64]map[string][]string{
		1: {PlatformOpenAI: {"gpt-5.6-*", "gpt-5.5"}},
	}}
	svc := newModelPlazaServiceForTest(groups, nil, source, nil)

	out, err := svc.ListGroups(context.Background())

	require.NoError(t, err)
	require.Len(t, out, 2)
	byID := map[int64]PlazaGroup{out[0].ID: out[0], out[1].ID: out[1]}
	require.Contains(t, plazaModelNames(byID[1].Models), "gpt-5.6-sol")
	require.Contains(t, plazaModelNames(byID[1].Models), "gpt-5.5")
	require.NotEmpty(t, byID[2].Models)
	for _, group := range byID {
		for _, model := range group.Models {
			require.NotContains(t, model.Name, "*")
		}
	}
}

func TestModelPlazaService_CompositeUsesOnlySchedulableConcretePlatforms(t *testing.T) {
	groups := []Group{{ID: 9, Name: "Composite", Platform: PlatformComposite, ActiveAccountCount: 2, RateMultiplier: 1}}
	source := &stubPlazaModelSource{
		models: map[int64]map[string][]string{
			9: {
				PlatformAnthropic: {"shared-model", "claude-opus-4-6"},
				PlatformOpenAI:    {"SHARED-MODEL", "gpt-5.6-sol"},
				PlatformGrok:      {"must-not-leak"},
			},
		},
		platforms: map[int64]map[string]struct{}{
			9: {PlatformAnthropic: {}, PlatformOpenAI: {}},
		},
	}
	svc := newModelPlazaServiceForTest(groups, nil, source, nil)

	out, err := svc.ListGroups(context.Background())

	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Equal(t, []string{"claude-opus-4-6", "gpt-5.6-sol", "shared-model"}, plazaModelNames(out[0].Models))
	byName := make(map[string]PlazaModel, len(out[0].Models))
	for _, model := range out[0].Models {
		byName[strings.ToLower(model.Name)] = model
	}
	require.Equal(t, PlatformAnthropic, byName["shared-model"].Platform, "composite dedupe follows gateway platform order")
	require.NotContains(t, byName, "must-not-leak")
}

func TestModelPlazaService_PricingPrecedenceAndUnpricedVisibility(t *testing.T) {
	groupPrice := 1.0
	channelPrice := 2.0
	groups := []Group{{
		ID: 19, Name: "Prices", Platform: PlatformOpenAI, ActiveAccountCount: 1, RateMultiplier: 0.6,
		ModelPricing: []ChannelModelPricing{
			{Platform: PlatformOpenAI, Models: []string{"group-model"}, BillingMode: BillingModePerRequest, PerRequestPrice: &groupPrice},
			{Platform: PlatformOpenAI, Models: []string{"channel-model"}, BillingMode: BillingModeToken},
		},
	}}
	channels := []Channel{{
		ID: 2, Name: "channel", Status: StatusActive, GroupIDs: []int64{19},
		ModelPricing: []ChannelModelPricing{
			{Platform: PlatformOpenAI, Models: []string{"group-model"}, InputPrice: &channelPrice},
			{Platform: PlatformOpenAI, Models: []string{"channel-model"}, InputPrice: &channelPrice},
		},
	}}
	pricing := newStubPricingServiceFromMap(map[string]*LiteLLMModelPricing{
		"official-model": {InputCostPerToken: 3.0, OutputCostPerToken: 4.0},
	})
	source := &stubPlazaModelSource{models: map[int64]map[string][]string{
		19: {PlatformOpenAI: {"group-model", "channel-model", "official-model", "unpriced-model"}},
	}}
	svc := newModelPlazaServiceForTest(groups, channels, source, pricing)

	out, err := svc.ListGroups(context.Background())

	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Len(t, out[0].Models, 4)
	byName := make(map[string]PlazaModel, len(out[0].Models))
	for _, model := range out[0].Models {
		byName[model.Name] = model
	}
	require.InDelta(t, groupPrice, *byName["group-model"].Pricing.PerRequestPrice, 1e-12)
	require.InDelta(t, channelPrice, *byName["channel-model"].Pricing.InputPrice, 1e-12)
	require.InDelta(t, 3.0, *byName["official-model"].Pricing.InputPrice, 1e-12)
	require.Nil(t, byName["unpriced-model"].Pricing)
}

func TestModelPlazaService_EmptyGroupMetadataShapesOfficialFallback(t *testing.T) {
	groups := []Group{{
		ID: 20, Name: "Image", Platform: PlatformOpenAI, ActiveAccountCount: 1, RateMultiplier: 1,
		ModelPricing: []ChannelModelPricing{{
			Platform: PlatformOpenAI, Models: []string{"official-image"}, BillingMode: BillingModeImage,
		}},
	}}
	channels := []Channel{{
		ID: 2, Name: "channel", Status: StatusActive, GroupIDs: []int64{20},
		ModelPricing: []ChannelModelPricing{{
			Platform: PlatformOpenAI, Models: []string{"official-image"}, BillingMode: BillingModeToken,
		}},
	}}
	pricing := newStubPricingServiceFromMap(map[string]*LiteLLMModelPricing{
		"official-image": {Mode: "image_generation", OutputCostPerImage: 0.04},
	})
	source := &stubPlazaModelSource{models: map[int64]map[string][]string{
		20: {PlatformOpenAI: {"official-image"}},
	}}
	svc := newModelPlazaServiceForTest(groups, channels, source, pricing)

	out, err := svc.ListGroups(context.Background())

	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Len(t, out[0].Models, 1)
	require.Equal(t, BillingModeImage, out[0].Models[0].Pricing.BillingMode)
	require.InDelta(t, 0.04, *out[0].Models[0].Pricing.PerRequestPrice, 1e-12)
}

func TestModelPlazaService_CompositePricingUsesConcretePlatform(t *testing.T) {
	openAIPrice := 2.0
	groups := []Group{{ID: 10, Name: "Composite", Platform: PlatformComposite, ActiveAccountCount: 1, RateMultiplier: 1}}
	channels := []Channel{{
		ID: 1, Name: "multi", Status: StatusActive, GroupIDs: []int64{10},
		ModelPricing: []ChannelModelPricing{
			{Platform: PlatformAnthropic, Models: []string{"gpt-5.6-sol"}, InputPrice: testPtrFloat64(9)},
			{Platform: PlatformOpenAI, Models: []string{"gpt-5.6-sol"}, InputPrice: &openAIPrice},
		},
	}}
	source := &stubPlazaModelSource{
		models:    map[int64]map[string][]string{10: {PlatformOpenAI: {"gpt-5.6-sol"}}},
		platforms: map[int64]map[string]struct{}{10: {PlatformOpenAI: {}}},
	}
	svc := newModelPlazaServiceForTest(groups, channels, source, nil)

	out, err := svc.ListGroups(context.Background())

	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Len(t, out[0].Models, 1)
	require.Equal(t, PlatformOpenAI, out[0].Models[0].Platform)
	require.InDelta(t, openAIPrice, *out[0].Models[0].Pricing.InputPrice, 1e-12)
}

func TestModelPlazaService_PreservesGroupImageDisplayPricing(t *testing.T) {
	channelDefault := 0.2
	channel4K := 0.3
	group1K := 0.02
	groups := []Group{{
		ID: 10, Name: "Images", Platform: PlatformOpenAI, ActiveAccountCount: 1, RateMultiplier: 1,
		ImagePrice1K: &group1K, ImageRateIndependent: true, ImageRateMultiplier: 1,
	}}
	channels := []Channel{{
		ID: 1, Name: "images", Status: StatusActive, GroupIDs: []int64{10},
		ModelPricing: []ChannelModelPricing{{
			Platform: PlatformOpenAI, Models: []string{"gpt-image-2"}, BillingMode: BillingModeImage,
			PerRequestPrice: &channelDefault,
			Intervals:       []PricingInterval{{TierLabel: "4K", PerRequestPrice: &channel4K}},
		}},
	}}
	source := &stubPlazaModelSource{models: map[int64]map[string][]string{10: {PlatformOpenAI: {"gpt-image-2"}}}}
	svc := newModelPlazaServiceForTest(groups, channels, source, nil)

	out, err := svc.ListGroups(context.Background())

	require.NoError(t, err)
	require.True(t, out[0].ImageRateIndependent)
	require.Len(t, out[0].Models[0].Pricing.Intervals, 3)
	tiers := make(map[string]float64)
	for _, interval := range out[0].Models[0].Pricing.Intervals {
		tiers[interval.TierLabel] = *interval.PerRequestPrice
	}
	require.InDelta(t, group1K, tiers["1K"], 1e-12)
	require.InDelta(t, channelDefault, tiers["2K"], 1e-12)
	require.InDelta(t, channel4K, tiers["4K"], 1e-12)
}

func TestModelPlazaService_PropagatesGroupRepositoryError(t *testing.T) {
	sentinel := errors.New("boom")
	svc := &ModelPlazaService{groupRepo: &stubGroupRepoForAvailable{listActiveErr: sentinel}}

	out, err := svc.ListGroups(context.Background())

	require.Nil(t, out)
	require.ErrorIs(t, err, sentinel)
}

func plazaModelNames(models []PlazaModel) []string {
	out := make([]string, 0, len(models))
	for _, model := range models {
		out = append(out, strings.TrimSpace(model.Name))
	}
	return out
}
