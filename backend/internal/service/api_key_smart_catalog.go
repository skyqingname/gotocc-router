package service

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/antigravity"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/claude"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/ctxkey"
	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/geminicli"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/openai"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/xai"
)

// AutoRouteModel is one public model that can be resolved for a particular
// endpoint. selected and selectedGroupID are deliberately internal: a catalog
// proves a model is usable without disclosing the group that will bill it.
type AutoRouteModel struct {
	ID       string `json:"id"`
	Platform string `json:"-"`

	selected        *AutoRouteDecision
	selectedGroupID int64
}

// AutoRouteCapabilities describes endpoints with at least one currently
// authorized, statically configured model. It is a display hint only; every
// request still resolves and admits its selected group independently.
type AutoRouteCapabilities struct {
	RoutingMode      string   `json:"routing_mode"`
	Protocols        []string `json:"protocols"`
	AsyncImageSubmit bool     `json:"async_image_submit"`
	BatchImageSubmit bool     `json:"batch_image_submit"`
}

type autoRouteCatalogState struct {
	policy  *AutoGroupRoutingPolicy
	key     *APIKey
	groups  []Group
	catalog *AutoRouteCatalog
}

type autoRouteCatalogPattern struct {
	value  string
	prefix bool
}

type autoRouteCatalogSources struct {
	exact    map[string]struct{}
	patterns []autoRouteCatalogPattern
}

func newAutoRouteCatalogSources() *autoRouteCatalogSources {
	return &autoRouteCatalogSources{exact: make(map[string]struct{})}
}

func (s *autoRouteCatalogSources) addExact(model string) {
	model = strings.TrimSpace(model)
	if model == "" {
		return
	}
	s.exact[model] = struct{}{}
	s.patterns = append(s.patterns, autoRouteCatalogPattern{value: model})
}

func (s *autoRouteCatalogSources) addWildcard(pattern string) {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return
	}
	s.patterns = append(s.patterns, autoRouteCatalogPattern{value: pattern})
}

func (s *autoRouteCatalogSources) addPrefix(prefix string) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return
	}
	s.patterns = append(s.patterns, autoRouteCatalogPattern{value: prefix, prefix: true})
}

func (s *autoRouteCatalogSources) matches(model string) bool {
	model = strings.TrimSpace(model)
	if model == "" {
		return false
	}
	for _, pattern := range s.patterns {
		if pattern.prefix {
			if strings.HasPrefix(model, pattern.value) {
				return true
			}
			continue
		}
		if autoRouteCatalogPatternMatches(pattern.value, model) {
			return true
		}
	}
	return false
}

func autoRouteCatalogPatternMatches(pattern, model string) bool {
	pattern = strings.TrimSpace(pattern)
	model = strings.TrimSpace(model)
	if pattern == "" || model == "" {
		return false
	}
	if pattern == model {
		return true
	}
	return strings.HasSuffix(pattern, "*") && strings.HasPrefix(model, strings.TrimSuffix(pattern, "*"))
}

// ListModels returns the strict authorized union for one concrete protocol
// endpoint. Callers must supply that endpoint rather than treating a generic
// model list as proof that every protocol can route the model.
func (s *AutoGroupResolver) ListModels(ctx context.Context, authenticated *APIKey, input AutoRouteRequest) ([]AutoRouteModel, error) {
	state, err := s.loadCatalogState(ctx, authenticated, input)
	if err != nil {
		return nil, err
	}
	return s.listModelsFromState(ctx, state, input)
}

// BuildCodexModelsManifest builds a local Responses catalog from the same
// strict snapshot as ListModels. It never loads a provider manifest and never
// falls back to a global/default account list when a catalog read fails.
func (s *AutoGroupResolver) BuildCodexModelsManifest(ctx context.Context, authenticated *APIKey) ([]byte, error) {
	input := AutoRouteRequest{Endpoint: CompositeRouteEndpointResponses}
	input.ForcePlatform, _ = ctx.Value(ctxkey.ForcePlatform).(string)
	state, err := s.loadCatalogState(ctx, authenticated, input)
	if err != nil {
		return nil, err
	}
	models, err := s.listModelsFromState(ctx, state, input)
	if err != nil {
		return nil, err
	}

	type codexEnvelope struct {
		Models []json.RawMessage `json:"models"`
	}
	manifest := codexEnvelope{Models: make([]json.RawMessage, 0, len(models))}
	groups := make(map[int64]*Group, len(state.groups))
	for i := range state.groups {
		groups[state.groups[i].ID] = &state.groups[i]
	}
	for _, model := range models {
		if model.selected == nil || model.selected.Key == nil || model.selected.Key.GroupID == nil {
			return nil, ErrAutoRouteUnavailable
		}
		group := groups[model.selectedGroupID]
		if group == nil {
			return nil, ErrAutoRouteUnavailable
		}
		effectivePlatform := model.Platform
		if group.Platform == PlatformComposite {
			effectivePlatform = PlatformComposite
		}
		body, err := buildCodexModelsManifestForAccounts(
			effectivePlatform,
			[]string{model.ID},
			state.catalog.Accounts[group.ID],
			state.catalog.Routes[group.ID],
			true,
		)
		if err != nil {
			return nil, ErrAutoRouteUnavailable.WithCause(err)
		}
		var entry codexEnvelope
		if err := json.Unmarshal(body, &entry); err != nil {
			return nil, ErrAutoRouteUnavailable.WithCause(err)
		}
		manifest.Models = append(manifest.Models, entry.Models...)
	}
	return json.Marshal(manifest)
}

// GetRoutingCapabilities reports only protocol and submit paths that have an
// authorized configured model in the current snapshot. It does not reserve an
// account or check transient quota/capacity.
func (s *AutoGroupResolver) GetRoutingCapabilities(ctx context.Context, authenticated *APIKey) (*AutoRouteCapabilities, error) {
	state, err := s.loadCatalogState(ctx, authenticated, AutoRouteRequest{Endpoint: CompositeRouteEndpointResponses})
	if err != nil {
		return nil, err
	}

	capabilities := &AutoRouteCapabilities{RoutingMode: string(APIKeyRoutingAuto)}
	checks := []struct {
		protocol string
		input    AutoRouteRequest
	}{
		{protocol: "openai", input: AutoRouteRequest{Endpoint: CompositeRouteEndpointResponses}},
		{protocol: "anthropic", input: AutoRouteRequest{Endpoint: CompositeRouteEndpointMessages}},
		{protocol: "gemini", input: AutoRouteRequest{Endpoint: CompositeRouteEndpointGemini}},
	}
	for _, check := range checks {
		models, err := s.listModelsFromState(ctx, state, check.input)
		if err != nil {
			return nil, err
		}
		if len(models) > 0 {
			capabilities.Protocols = append(capabilities.Protocols, check.protocol)
		}
	}

	images, err := s.listModelsFromState(ctx, state, AutoRouteRequest{
		Endpoint: CompositeRouteEndpointImages, ImageGeneration: true,
	})
	if err != nil {
		return nil, err
	}
	capabilities.AsyncImageSubmit = len(images) > 0

	batch, err := s.listModelsFromState(ctx, state, AutoRouteRequest{
		Endpoint: AutoRouteEndpointBatchImages,
	})
	if err != nil {
		return nil, err
	}
	capabilities.BatchImageSubmit = len(batch) > 0
	sort.Strings(capabilities.Protocols)
	return capabilities, nil
}

func (s *AutoGroupResolver) loadCatalogState(ctx context.Context, authenticated *APIKey, input AutoRouteRequest) (*autoRouteCatalogState, error) {
	if strings.TrimSpace(input.Endpoint) == "" {
		return nil, infraerrors.BadRequest("MODEL_CATALOG_ENDPOINT_REQUIRED", "endpoint is required for automatic routing model catalog")
	}
	key, err := s.FreshKey(ctx, authenticated)
	if err != nil {
		return nil, err
	}
	key, groups, err := s.eligibleGroups(ctx, key)
	if err != nil {
		return nil, err
	}
	if input.RequiredGroupID != nil {
		filtered := make([]Group, 0, 1)
		for _, group := range groups {
			if group.ID == *input.RequiredGroupID {
				filtered = append(filtered, group)
			}
		}
		if len(filtered) == 0 {
			return nil, ErrAutoRouteNoAccess
		}
		groups = filtered
	}
	if s.catalog == nil || s.channels == nil {
		return nil, ErrAutoRouteUnavailable
	}
	groupIDs := make([]int64, 0, len(groups))
	for _, group := range groups {
		groupIDs = append(groupIDs, group.ID)
	}
	catalog, err := s.catalog.LoadAutoRouteCatalog(ctx, groupIDs)
	if err != nil {
		return nil, ErrAutoRouteUnavailable.WithCause(err)
	}
	if catalog == nil {
		return nil, ErrAutoRouteUnavailable
	}
	policy, err := s.routingPolicy(ctx, input.RequiredGroupID != nil)
	if err != nil {
		return nil, err
	}
	return &autoRouteCatalogState{key: key, groups: groups, catalog: catalog, policy: policy}, nil
}

func (s *AutoGroupResolver) listModelsFromState(ctx context.Context, state *autoRouteCatalogState, input AutoRouteRequest) ([]AutoRouteModel, error) {
	if state == nil || state.key == nil || state.catalog == nil {
		return nil, ErrAutoRouteUnavailable
	}
	candidates, err := s.catalogCandidateIDs(ctx, state, input)
	if err != nil {
		return nil, err
	}
	models := make([]AutoRouteModel, 0, len(candidates))
	for _, candidate := range candidates {
		if err := ctx.Err(); err != nil {
			return nil, ErrAutoRouteUnavailable.WithCause(err)
		}
		decision, err := s.resolveCatalogModel(ctx, state, candidate, input)
		if errors.Is(err, ErrAutoRouteModelNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if decision == nil || decision.Key == nil || decision.Key.GroupID == nil {
			return nil, ErrAutoRouteUnavailable
		}
		models = append(models, AutoRouteModel{
			ID:              candidate,
			Platform:        decision.Platform,
			selected:        decision,
			selectedGroupID: *decision.Key.GroupID,
		})
	}
	sort.Slice(models, func(i, j int) bool { return models[i].ID < models[j].ID })
	return models, nil
}

func (s *AutoGroupResolver) resolveCatalogModel(ctx context.Context, state *autoRouteCatalogState, model string, input AutoRouteRequest) (*AutoRouteDecision, error) {
	input.Model = model
	groups := state.policy.OrderGroups(state.groups, model)
	for i := range groups {
		decision, err := s.matchGroup(ctx, state.key, &groups[i], state.catalog, input)
		if err != nil {
			return nil, err
		}
		if decision != nil {
			return decision, nil
		}
	}
	return nil, ErrAutoRouteModelNotFound
}

func (s *AutoGroupResolver) catalogCandidateIDs(ctx context.Context, state *autoRouteCatalogState, input AutoRouteRequest) ([]string, error) {
	seen := make(map[string]struct{})
	for i := range state.groups {
		sources, err := s.catalogSourcesForGroup(ctx, &state.groups[i], state.catalog, input)
		if err != nil {
			return nil, err
		}
		for model := range sources.exact {
			if !autoRouteGroupCatalogAllowsModel(&state.groups[i], model) {
				continue
			}
			seen[model] = struct{}{}
		}
		if !state.groups[i].CustomModelsListEnabled() {
			continue
		}
		for _, model := range state.groups[i].ModelsListConfig.Models {
			model = strings.TrimSpace(model)
			if model == "" || strings.ContainsAny(model, "*?") || !sources.matches(model) {
				continue
			}
			seen[model] = struct{}{}
		}
	}
	models := make([]string, 0, len(seen))
	for model := range seen {
		models = append(models, model)
	}
	sort.Strings(models)
	return models, nil
}

func autoRouteGroupCatalogAllowsModel(group *Group, model string) bool {
	if group == nil || !group.CustomModelsListEnabled() {
		return true
	}
	for _, pattern := range group.ModelsListConfig.Models {
		if autoRouteCatalogPatternMatches(pattern, model) {
			return true
		}
	}
	return false
}

func (s *AutoGroupResolver) catalogSourcesForGroup(ctx context.Context, group *Group, catalog *AutoRouteCatalog, input AutoRouteRequest) (*autoRouteCatalogSources, error) {
	if group == nil || catalog == nil {
		return nil, ErrAutoRouteUnavailable
	}
	sources := newAutoRouteCatalogSources()
	accounts := catalog.Accounts[group.ID]
	platforms := make(map[string]struct{})
	for i := range accounts {
		account := accounts[i]
		if !autoRouteCatalogAccountUsable(group, &account) {
			continue
		}
		if group.Platform == PlatformComposite {
			platforms[account.Platform] = struct{}{}
		} else {
			platforms[group.Platform] = struct{}{}
		}
		for model := range account.GetModelMapping() {
			if strings.ContainsAny(model, "*?") {
				sources.addWildcard(model)
				continue
			}
			sources.addExact(model)
		}
	}
	for platform := range platforms {
		for _, model := range autoRouteDefaultCatalogModelIDs(platform) {
			sources.addExact(model)
		}
	}

	lookup, err := s.channels.lookupGroupChannel(ctx, group.ID)
	if err != nil {
		return nil, ErrAutoRouteUnavailable.WithCause(err)
	}
	if lookup != nil && lookup.channel != nil {
		for _, platform := range matchingPlatforms(group.Platform) {
			for model := range lookup.channel.ModelMapping[platform] {
				if strings.ContainsAny(model, "*?") {
					sources.addWildcard(model)
					continue
				}
				sources.addExact(model)
			}
		}
	}

	if group.Platform == PlatformComposite {
		endpoint := normalizeCompositeRouteEndpoint(autoCompositeEndpoint(input.Endpoint))
		for _, route := range catalog.Routes[group.ID] {
			if !route.Enabled {
				continue
			}
			routeEndpoint := normalizeCompositeRouteEndpoint(route.Endpoint)
			if routeEndpoint != endpoint && routeEndpoint != CompositeRouteEndpointAny {
				continue
			}
			if normalizeCompositeRouteMatchType(route.MatchType) == CompositeRouteMatchPrefix {
				sources.addPrefix(route.PublicModel)
				continue
			}
			sources.addExact(route.PublicModel)
		}
	}
	return sources, nil
}

func autoRouteCatalogAccountUsable(group *Group, account *Account) bool {
	if group == nil || account == nil || account.Status != StatusActive || !account.Schedulable ||
		(group.RequireOAuthOnly && account.Type == AccountTypeAPIKey) ||
		(group.RequirePrivacySet && !account.IsPrivacySet()) {
		return false
	}
	if group.Platform == PlatformComposite {
		return isConcreteRequestPlatform(account.Platform)
	}
	if account.Platform == group.Platform {
		return true
	}
	return (group.Platform == PlatformAnthropic || group.Platform == PlatformGemini) &&
		account.Platform == PlatformAntigravity && account.IsMixedSchedulingEnabled()
}

func autoRouteDefaultCatalogModelIDs(platform string) []string {
	switch platform {
	case PlatformOpenAI:
		return openai.DefaultModelIDs()
	case PlatformAnthropic:
		return claude.DefaultModelIDs()
	case PlatformGemini:
		models := make([]string, 0, len(geminicli.DefaultModels))
		for _, model := range geminicli.DefaultModels {
			models = append(models, model.ID)
		}
		return models
	case PlatformAntigravity:
		models := antigravity.DefaultModels()
		ids := make([]string, 0, len(models))
		for _, model := range models {
			ids = append(ids, model.ID)
		}
		return ids
	case PlatformGrok:
		return xai.DefaultModelIDs()
	default:
		// Domestic and unknown providers have no safe finite built-in catalog.
		return nil
	}
}
