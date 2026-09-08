package service

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/LuckyKuang/sub2api-plus/internal/config"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/ctxkey"
	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/ip"
)

const (
	AutoRouteEndpointBatchImages = "batch_images"
	AutoRouteEndpointVideos      = "videos"
	AutoRouteEndpointLive        = "live"
	AutoRouteEndpointAlphaSearch = "alpha_search"
	AutoRouteEndpointWebSearch   = "web_search"
	AutoRouteEndpointVoice       = "voice"
	AutoRouteEndpointResponsesWS = "responses_ws"
)

var (
	ErrAutoRouteNoAccess      = infraerrors.Forbidden("AUTO_ROUTE_NO_ACCESS", "no accessible group is available")
	ErrAutoRouteModelNotFound = infraerrors.NotFound("model_not_found", "the requested model is not available for this key and endpoint")
	ErrAutoRouteUnavailable   = infraerrors.ServiceUnavailable("AUTO_ROUTE_UNAVAILABLE", "automatic routing is temporarily unavailable")
	ErrAutoRouteContext       = infraerrors.Conflict("AUTO_ROUTE_CONTEXT_REQUIRED", "the original routing context is required; start a new request or connection")
)

type AutoRouteCatalog struct {
	Accounts map[int64][]Account
	Routes   map[int64][]CompositeModelRoute
}

type AutoRouteCatalogRepository interface {
	LoadAutoRouteCatalog(context.Context, []int64) (*AutoRouteCatalog, error)
}

type AutoRouteRequest struct {
	Model             string
	Endpoint          string
	ForcePlatform     string
	Provider          string
	ImageGeneration   bool
	ClaudeCodeClient  bool
	RequiredGroupID   *int64
	RequiredAccountID int64
}

type AutoRouteDecision struct {
	Endpoint          string
	Key               *APIKey
	Platform          string
	PublicModel       string
	UpstreamModel     string
	Composite         *CompositeRouteDecision
	ImageGeneration   bool
	RequiredAccountID int64
}

type AutoGroupResolver struct {
	policy         *AutoGroupRoutingPolicyService
	keys           *APIKeyService
	catalog        AutoRouteCatalogRepository
	channels       *ChannelService
	subscriptions  *SubscriptionService
	batchProviders *BatchImageProviderRegistry
	cfg            *config.Config
}

func NewAutoGroupResolver(keys *APIKeyService, catalog AutoRouteCatalogRepository, channels *ChannelService, subscriptions *SubscriptionService, policy *AutoGroupRoutingPolicyService, cfg *config.Config) *AutoGroupResolver {
	return &AutoGroupResolver{
		policy: policy,
		keys:   keys, catalog: catalog, channels: channels, subscriptions: subscriptions,
		batchProviders: NewBatchImageProviderRegistryFromConfig(cfg), cfg: cfg,
	}
}

// Resolve reads authorization and configured capabilities. It never selects an
// account, consumes a limit, maintains a subscription window, or contacts upstream.
func (s *AutoGroupResolver) Resolve(ctx context.Context, key *APIKey, input AutoRouteRequest) (*AutoRouteDecision, error) {
	input.Model = strings.TrimSpace(input.Model)
	if input.Model == "" {
		return nil, infraerrors.BadRequest("MODEL_REQUIRED", "model is required for automatic routing")
	}
	key, err := s.FreshKey(ctx, key)
	if err != nil {
		return nil, err
	}
	key, groups, err := s.eligibleGroups(ctx, key)
	if err != nil {
		return nil, err
	}
	policy, err := s.routingPolicy(ctx, input.RequiredGroupID != nil)
	if err != nil {
		return nil, err
	}
	groups = policy.OrderGroups(groups, input.Model)
	if input.RequiredGroupID != nil {
		selected := groups[:0]
		for _, group := range groups {
			if group.ID == *input.RequiredGroupID {
				selected = append(selected, group)
			}
		}
		groups = selected
		if len(groups) == 0 {
			return nil, ErrAutoRouteNoAccess
		}
	}
	groupIDs := make([]int64, 0, len(groups))
	for _, group := range groups {
		groupIDs = append(groupIDs, group.ID)
	}
	if s.catalog == nil || s.channels == nil {
		return nil, ErrAutoRouteUnavailable
	}
	catalog, err := s.catalog.LoadAutoRouteCatalog(ctx, groupIDs)
	if err != nil {
		return nil, ErrAutoRouteUnavailable.WithCause(err)
	}
	if catalog == nil {
		return nil, ErrAutoRouteUnavailable
	}
	for i := range groups {
		if err := ctx.Err(); err != nil {
			return nil, ErrAutoRouteUnavailable.WithCause(err)
		}
		decision, err := s.matchGroup(ctx, key, &groups[i], catalog, input)
		if err != nil {
			return nil, err
		}
		if decision != nil {
			return decision, nil
		}
	}
	return nil, ErrAutoRouteModelNotFound
}

func (s *AutoGroupResolver) FreshKey(ctx context.Context, authenticated *APIKey) (*APIKey, error) {
	if s == nil || s.keys == nil || s.keys.apiKeyRepo == nil {
		return nil, ErrAutoRouteUnavailable
	}
	if s.cfg != nil && s.cfg.RunMode == config.RunModeSimple {
		return nil, infraerrors.Forbidden("AUTO_ROUTING_UNSUPPORTED_RUN_MODE", "automatic routing requires standard run mode")
	}
	if authenticated == nil || authenticated.ID <= 0 || !authenticated.IsAutoRouting() {
		return nil, ErrAutoRouteNoAccess
	}
	key, err := s.keys.apiKeyRepo.GetByID(ctx, authenticated.ID)
	if errors.Is(err, ErrAPIKeyNotFound) {
		return nil, ErrAutoRouteNoAccess
	}
	if err != nil {
		return nil, ErrAutoRouteUnavailable.WithCause(err)
	}
	if key == nil || key.User == nil || key.Key != authenticated.Key || key.UserID != authenticated.UserID ||
		!key.IsAutoRouting() || key.GroupID != nil ||
		(key.TeamID == nil) != (authenticated.TeamID == nil) ||
		(key.TeamID != nil && *key.TeamID != *authenticated.TeamID) {
		return nil, ErrAutoRouteContext
	}
	if key.IsExpired() || key.Status == StatusAPIKeyExpired {
		return nil, ErrAPIKeyExpired
	}
	if key.IsQuotaExhausted() || key.Status == StatusAPIKeyQuotaExhausted {
		return nil, ErrAPIKeyQuotaExhausted
	}
	if !key.IsActive() {
		return nil, ErrAutoRouteNoAccess
	}
	copyKey := *key
	s.keys.compileAPIKeyIPRules(&copyKey)
	if clientIP, ok := ctx.Value(autoRouteClientIPKey{}).(string); ok {
		if allowed, _ := ip.CheckIPRestrictionWithCompiledRules(clientIP, copyKey.CompiledIPWhitelist, copyKey.CompiledIPBlacklist); !allowed {
			return nil, ErrAutoRouteNoAccess
		}
	}
	return &copyKey, nil
}

func (s *AutoGroupResolver) eligibleGroups(ctx context.Context, key *APIKey) (*APIKey, []Group, error) {
	if s == nil || s.keys == nil || s.keys.userRepo == nil || s.keys.groupRepo == nil || s.keys.userSubRepo == nil {
		return nil, nil, ErrAutoRouteUnavailable
	}
	if s.cfg != nil && s.cfg.RunMode == config.RunModeSimple {
		return nil, nil, infraerrors.Forbidden("AUTO_ROUTING_UNSUPPORTED_RUN_MODE", "automatic routing requires standard run mode")
	}
	if key == nil || key.User == nil || !key.IsAutoRouting() || !key.IsActive() {
		return nil, nil, ErrAutoRouteNoAccess
	}
	copyKey := *key
	if key.TeamID != nil {
		hydrated, err := s.keys.hydrateTeamAPIKey(ctx, &copyKey, nil)
		if err != nil {
			return nil, nil, err
		}
		copyKey = *hydrated
	} else if key.User.ID != key.UserID {
		return nil, nil, ErrAutoRouteNoAccess
	}
	payer, err := s.keys.userRepo.GetByID(ctx, copyKey.User.ID)
	if err != nil {
		return nil, nil, ErrAutoRouteUnavailable.WithCause(err)
	}
	if payer == nil || payer.ID != copyKey.User.ID || !payer.IsActive() {
		return nil, nil, ErrAutoRouteNoAccess
	}
	user := *payer
	user.UserGroupRPMOverride = nil
	copyKey.User = &user
	groups, err := s.keys.GetAvailableGroups(ctx, user.ID)
	if err != nil {
		return nil, nil, ErrAutoRouteUnavailable.WithCause(err)
	}
	active := make([]Group, 0, len(groups))
	for _, group := range groups {
		if group.ID > 0 && group.IsActive() {
			active = append(active, group)
		}
	}
	if len(active) == 0 {
		return nil, nil, ErrAutoRouteNoAccess
	}
	sort.Slice(active, func(i, j int) bool {
		if active[i].SortOrder != active[j].SortOrder {
			return active[i].SortOrder < active[j].SortOrder
		}
		return active[i].ID < active[j].ID
	})
	return &copyKey, active, nil
}

func (s *AutoGroupResolver) matchGroup(ctx context.Context, key *APIKey, group *Group, catalog *AutoRouteCatalog, input AutoRouteRequest) (*AutoRouteDecision, error) {
	if group.ClaudeCodeOnly && !input.ClaudeCodeClient {
		return nil, nil
	}
	platform, model := group.Platform, input.Model
	var composite *CompositeRouteDecision
	if platform == PlatformComposite {
		decision, err := resolveAutoCompositeRoute(ctx, group.ID, input.Model, autoCompositeEndpoint(input.Endpoint), catalog)
		if err != nil {
			return nil, ErrAutoRouteUnavailable.WithCause(err)
		}
		if !decision.Matched {
			return nil, nil
		}
		composite = &decision
		platform, model = decision.TargetPlatform, decision.UpstreamModel
		ctx = WithCompositeRouteDecision(ctx, decision)
	} else if platform == PlatformOpenAI && input.Endpoint == CompositeRouteEndpointMessages {
		model = group.ResolveMessagesDispatchModel(input.Model)
	}
	if input.ForcePlatform != "" && platform != input.ForcePlatform {
		return nil, nil
	}
	if !autoRouteEndpointSupportsGroup(group, platform, input.Endpoint) {
		return nil, nil
	}
	imageGeneration := input.ImageGeneration || IsGPTImageGenerationModel(model)
	if imageGeneration && !group.AllowImageGeneration {
		return nil, nil
	}
	lookup, err := s.channels.lookupGroupChannel(ctx, group.ID)
	if err != nil {
		return nil, ErrAutoRouteUnavailable.WithCause(err)
	}
	mapping := ChannelMappingResult{MappedModel: model}
	if lookup != nil {
		mapping = resolveMapping(lookup, group.ID, model)
		billingModel := billingModelForRestriction(mapping.BillingModelSource, model, mapping.MappedModel)
		if billingModel != "" && checkRestricted(lookup, group.ID, billingModel) {
			return nil, nil
		}
	}
	for i := range catalog.Accounts[group.ID] {
		accountCopy := catalog.Accounts[group.ID][i]
		account := &accountCopy
		if input.RequiredAccountID > 0 && account.ID != input.RequiredAccountID {
			continue
		}
		mixed := (platform == PlatformAnthropic || platform == PlatformGemini) && account.Platform == PlatformAntigravity && account.IsMixedSchedulingEnabled()
		if account.Platform != platform && !mixed {
			continue
		}
		if account.Status != StatusActive || !account.Schedulable ||
			(group.RequireOAuthOnly && account.Type == AccountTypeAPIKey) ||
			(group.RequirePrivacySet && !account.IsPrivacySet()) {
			continue
		}
		accountModel := mapping.MappedModel
		if input.Endpoint == CompositeRouteEndpointImages && account.Platform == PlatformOpenAI {
			accountModel = model
		}
		explicitRoute := mapping.Mapped || (composite != nil && composite.Source == CompositeRouteSourceExplicit)
		if !autoRouteAccountClaimsModel(account, accountModel, explicitRoute) {
			continue
		}
		modelSupported := gatewayAccountSupportsModel(ctx, account, accountModel)
		if input.Endpoint == CompositeRouteEndpointImages && account.Platform == PlatformOpenAI {
			modelSupported = account.IsModelDirectlySupported(accountModel)
		}
		if !s.accountSupportsEndpoint(account, input) || !modelSupported {
			continue
		}
		if imageGeneration && (input.Endpoint == CompositeRouteEndpointResponses || input.Endpoint == AutoRouteEndpointResponsesWS) &&
			!account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityResponses) {
			continue
		}
		if lookup != nil && mapping.BillingModelSource == BillingModelSourceUpstream {
			upstreamModel := resolveAccountUpstreamModel(account, mapping.MappedModel)
			if account.IsOpenAICompatible() {
				upstreamModel = resolveOpenAIAccountUpstreamModelForRequest(account, mapping.MappedModel, false)
			}
			if upstreamModel != "" && checkRestricted(lookup, group.ID, upstreamModel) {
				continue
			}
		}
		copyKey, copyUser, copyGroup := *key, *key.User, *group
		groupID := group.ID
		copyKey.GroupID, copyKey.Group, copyKey.User = &groupID, &copyGroup, &copyUser
		copyUser.UserGroupRPMOverride = nil
		return &AutoRouteDecision{
			Endpoint: input.Endpoint,
			Key:      &copyKey, Platform: platform, PublicModel: input.Model,
			UpstreamModel: model, Composite: composite,
			ImageGeneration:   imageGeneration,
			RequiredAccountID: input.RequiredAccountID,
		}, nil
	}
	return nil, nil
}

func autoRouteAccountClaimsModel(account *Account, model string, explicitRoute bool) bool {
	if explicitRoute || account.IsOpenAIPassthroughEnabled() || len(account.GetModelMapping()) > 0 || account.Platform == PlatformAntigravity {
		return true
	}
	platform, recognized := DetectModelPlatform(model)
	return recognized && platform == account.Platform
}

func autoRouteEndpointSupportsGroup(group *Group, platform, endpoint string) bool {
	switch endpoint {
	case CompositeRouteEndpointMessages, CompositeRouteEndpointCountTokens:
		return isConcreteRequestPlatform(platform) && (platform != PlatformOpenAI || group.AllowMessagesDispatch)
	case CompositeRouteEndpointResponses, CompositeRouteEndpointChatCompletions:
		return isConcreteRequestPlatform(platform)
	case AutoRouteEndpointResponsesWS:
		return platform == PlatformOpenAI || platform == PlatformGrok || IsCNProvider(platform)
	case CompositeRouteEndpointGemini:
		return platform == PlatformGemini || platform == PlatformAntigravity
	case CompositeRouteEndpointEmbeddings, AutoRouteEndpointAlphaSearch:
		return platform == PlatformOpenAI
	case CompositeRouteEndpointImages:
		return group.AllowImageGeneration && (platform == PlatformOpenAI || platform == PlatformGrok)
	case AutoRouteEndpointBatchImages:
		return group.AllowBatchImageGeneration && group.Platform == PlatformGemini && !group.IsSubscriptionType() && platform == PlatformGemini
	case AutoRouteEndpointVideos:
		return platform == PlatformOpenAI || platform == PlatformGrok
	case AutoRouteEndpointLive:
		return group.AllowLive && platform == PlatformOpenAI
	case AutoRouteEndpointWebSearch, AutoRouteEndpointVoice:
		return platform == PlatformGrok
	default:
		return false
	}
}

func (s *AutoGroupResolver) accountSupportsEndpoint(account *Account, input AutoRouteRequest) bool {
	switch input.Endpoint {
	case AutoRouteEndpointVideos:
		if account.Platform == PlatformOpenAI {
			return s.cfg != nil && s.cfg.VideoTask.Enabled
		}
		return account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityGrokMediaGeneration)
	case AutoRouteEndpointResponsesWS:
		return openAIAccountTransportCompatible(s.cfg, NewOpenAIWSProtocolResolver(s.cfg), account, OpenAIUpstreamTransportResponsesWebsocketV2Ingress)
	case CompositeRouteEndpointEmbeddings:
		return account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityEmbeddings)
	case AutoRouteEndpointAlphaSearch:
		return account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityAlphaSearch)
	case AutoRouteEndpointLive:
		return account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityLive)
	case CompositeRouteEndpointChatCompletions:
		return !account.IsOpenAICompatible() || account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityChatCompletions)
	case CompositeRouteEndpointImages:
		if account.Platform == PlatformGrok {
			return account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityGrokMediaGeneration)
		}
		return account.SupportsOpenAIImageCapability(OpenAIImagesCapabilityBasic)
	case AutoRouteEndpointBatchImages:
		for _, name := range batchImageProviderSelectionOrder(input.Provider) {
			if provider, ok := s.batchProviders.Get(name); ok && provider.SupportsAccount(account) {
				return true
			}
		}
		return false
	default:
		return true
	}
}

func autoCompositeEndpoint(endpoint string) string {
	switch endpoint {
	case AutoRouteEndpointResponsesWS, AutoRouteEndpointLive, AutoRouteEndpointAlphaSearch:
		return CompositeRouteEndpointResponses
	case AutoRouteEndpointBatchImages:
		return CompositeRouteEndpointImages
	case AutoRouteEndpointVideos, AutoRouteEndpointVoice, AutoRouteEndpointWebSearch:
		return CompositeRouteEndpointAny
	default:
		return endpoint
	}
}

func resolveAutoCompositeRoute(ctx context.Context, groupID int64, model, endpoint string, catalog *AutoRouteCatalog) (CompositeRouteDecision, error) {
	if route, ok := matchCompositeRoute(catalog.Routes[groupID], model, endpoint); ok {
		upstream := strings.TrimSpace(route.UpstreamModel)
		if upstream == "" {
			upstream = model
		}
		return CompositeRouteDecision{
			Matched: true, Source: CompositeRouteSourceExplicit, GroupID: groupID,
			PublicModel: model, TargetPlatform: route.TargetPlatform,
			UpstreamModel: upstream, Endpoint: endpoint, Route: &route,
		}, nil
	}
	resolver := NewCompositeRouteResolver(nil)
	resolver.SetModelOwnershipResolver(func(_ context.Context, _ int64, requested string) (CompositeModelOwnership, error) {
		platforms := make(map[string]struct{})
		for _, account := range catalog.Accounts[groupID] {
			if isConcreteRequestPlatform(account.Platform) && explicitModelMappingClaims(account, requested) {
				platforms[account.Platform] = struct{}{}
			}
		}
		if len(platforms) > 1 {
			return CompositeModelOwnership{Ambiguous: true}, nil
		}
		for platform := range platforms {
			return CompositeModelOwnership{Matched: true, TargetPlatform: platform}, nil
		}
		return CompositeModelOwnership{}, nil
	})
	return resolver.Resolve(ctx, groupID, model, endpoint)
}

type autoRouteContextKey struct{}
type autoRouteClientIPKey struct{}

func WithAutoRouteClientIP(ctx context.Context, clientIP string) context.Context {
	return context.WithValue(ctx, autoRouteClientIPKey{}, clientIP)
}

type autoRouteDecisionKey struct{}
type autoRouteDeferredKey struct{}
type AutoRouteDeferredKind string

const (
	AutoRouteDeferredModels   AutoRouteDeferredKind = "models"
	AutoRouteDeferredResource AutoRouteDeferredKind = "resource"
	AutoRouteDeferredWS       AutoRouteDeferredKind = "websocket"
)

type autoRouteLock struct {
	KeyID   int64
	GroupID int64
}

func WithAutoRouteLock(ctx context.Context, keyID, groupID int64) context.Context {
	return context.WithValue(ctx, autoRouteContextKey{}, autoRouteLock{KeyID: keyID, GroupID: groupID})
}

func AutoRouteGroupID(ctx context.Context) (int64, bool) {
	if ctx == nil {
		return 0, false
	}
	lock, ok := ctx.Value(autoRouteContextKey{}).(autoRouteLock)
	return lock.GroupID, ok && lock.KeyID > 0 && lock.GroupID > 0
}

func IsAutoRoutingRequest(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	_, ok := ctx.Value(autoRouteContextKey{}).(autoRouteLock)
	return ok
}

func WithAutoRouteDeferred(ctx context.Context, keyID int64, kind AutoRouteDeferredKind) context.Context {
	ctx = WithAutoRouteLock(ctx, keyID, 0)
	return context.WithValue(ctx, autoRouteDeferredKey{}, kind)
}

func CanDeferAutoRoute(ctx context.Context, keyID int64) bool {
	lock, ok := ctx.Value(autoRouteContextKey{}).(autoRouteLock)
	if !ok || lock.KeyID != keyID || lock.GroupID != 0 {
		return false
	}
	kind, _ := ctx.Value(autoRouteDeferredKey{}).(AutoRouteDeferredKind)
	return kind == AutoRouteDeferredModels || kind == AutoRouteDeferredResource || kind == AutoRouteDeferredWS
}

func (d *AutoRouteDecision) RequestContext(ctx context.Context) context.Context {
	ctx = WithAutoRouteLock(ctx, d.Key.ID, *d.Key.GroupID)
	ctx = context.WithValue(ctx, autoRouteDecisionKey{}, d)
	ctx = context.WithValue(ctx, ctxkey.Group, d.Key.Group)
	if d.Composite != nil {
		ctx = WithCompositeRouteDecision(ctx, *d.Composite)
	}
	if d.ImageGeneration && d.Platform == PlatformOpenAI {
		ctx = WithOpenAIImageGenerationIntent(ctx)
	}
	return ctx
}

func AutoRouteDecisionFromContext(ctx context.Context) (*AutoRouteDecision, bool) {
	if ctx == nil {
		return nil, false
	}
	decision, ok := ctx.Value(autoRouteDecisionKey{}).(*AutoRouteDecision)
	return decision, ok && decision != nil
}
