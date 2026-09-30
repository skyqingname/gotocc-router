package service

import (
	"context"
	"sort"
	"strings"

	"github.com/LuckyKuang/sub2api-plus/internal/config"
	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
)

// AutoRoutePriorityGroup exposes only the identity of an authorized candidate.
type AutoRoutePriorityGroup struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Platform string `json:"platform"`
}

type AutoRouteModelPriority struct {
	Model       string                   `json:"model"`
	MatchedRule string                   `json:"matched_rule"`
	Groups      []AutoRoutePriorityGroup `json:"groups"`
}

type AutoRoutePriorities struct {
	DefaultSource string                   `json:"default_source"`
	Groups        []AutoRoutePriorityGroup `json:"groups"`
	ModelRules    []AutoRouteModelPriority `json:"model_rules"`
}

// GetRoutingPriorities previews settings for creation as well as existing keys.
// It reuses scope authorization and routing compatibility, without authenticating
// a key, selecting an account, or checking/consuming transient limits.
func (s *AutoGroupResolver) GetRoutingPriorities(ctx context.Context, userID int64, scope string) (*AutoRoutePriorities, error) {
	scope = strings.ToLower(strings.TrimSpace(scope))
	if scope != "personal" && scope != "team" {
		return nil, infraerrors.BadRequest("INVALID_SCOPE", "scope must be personal or team")
	}
	if s == nil || s.keys == nil || s.keys.userRepo == nil || s.keys.groupRepo == nil || s.keys.userSubRepo == nil || s.catalog == nil || s.channels == nil {
		return nil, ErrAutoRouteUnavailable
	}
	if s.cfg != nil && s.cfg.RunMode == config.RunModeSimple {
		return nil, infraerrors.Forbidden("AUTO_ROUTING_UNSUPPORTED_RUN_MODE", "automatic routing requires standard run mode")
	}
	user, err := s.keys.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, ErrAutoRouteUnavailable.WithCause(err)
	}
	if user == nil || user.ID != userID || !user.IsActive() {
		return nil, ErrAutoRouteNoAccess
	}
	groups, err := s.keys.GetAvailableGroupsForScope(ctx, userID, scope)
	if err != nil {
		return nil, err
	}
	active := make([]Group, 0, len(groups))
	for _, group := range groups {
		if group.ID > 0 && group.IsActive() {
			active = append(active, group)
		}
	}
	sort.Slice(active, func(i, j int) bool {
		if active[i].SortOrder != active[j].SortOrder {
			return active[i].SortOrder < active[j].SortOrder
		}
		return active[i].ID < active[j].ID
	})
	policy, err := s.routingPolicy(ctx, false)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(active))
	for _, group := range active {
		ids = append(ids, group.ID)
	}
	catalog, err := s.catalog.LoadAutoRouteCatalog(ctx, ids)
	if err != nil {
		return nil, ErrAutoRouteUnavailable.WithCause(err)
	}
	if catalog == nil {
		return nil, ErrAutoRouteUnavailable
	}
	// This key is only a container for matchGroup's metadata result. Authorized
	// candidates above already reflect the personal user or team payer's scope.
	state := &autoRouteCatalogState{key: &APIKey{User: user}, groups: active, catalog: catalog, policy: policy}
	return s.routingPrioritiesFromState(ctx, state)
}

func (s *AutoGroupResolver) routingPrioritiesFromState(ctx context.Context, state *autoRouteCatalogState) (*AutoRoutePriorities, error) {
	competing := make(map[int64]bool)
	modelGroups := make(map[string]map[int64]bool)
	endpoints := []string{
		CompositeRouteEndpointResponses, CompositeRouteEndpointMessages,
		CompositeRouteEndpointChatCompletions, CompositeRouteEndpointGemini,
		CompositeRouteEndpointImages, CompositeRouteEndpointEmbeddings,
		CompositeRouteEndpointCountTokens, AutoRouteEndpointResponsesWS,
		AutoRouteEndpointBatchImages, AutoRouteEndpointVideos, AutoRouteEndpointLive,
		AutoRouteEndpointAlphaSearch, AutoRouteEndpointWebSearch, AutoRouteEndpointVoice,
	}
	for _, endpoint := range endpoints {
		input := AutoRouteRequest{Endpoint: endpoint, ClaudeCodeClient: true}
		candidates, err := s.catalogCandidateIDs(ctx, state, input)
		if err != nil {
			return nil, err
		}
		// An exact administrator rule gives us a finite public model name even
		// when account mappings only contain wildcards. Compatibility below still
		// has to prove it is shared by this user's authorized groups.
		seen := make(map[string]bool, len(candidates))
		for _, model := range candidates {
			seen[model] = true
		}
		for _, rule := range state.policy.ModelRules {
			if !strings.ContainsAny(rule.Model, "*?") && !seen[rule.Model] {
				candidates = append(candidates, rule.Model)
				seen[rule.Model] = true
			}
		}
		for _, model := range candidates {
			if err := ctx.Err(); err != nil {
				return nil, ErrAutoRouteUnavailable.WithCause(err)
			}
			input.Model = model
			matching := make([]int64, 0, len(state.groups))
			for i := range state.groups {
				decision, err := s.matchGroup(ctx, state.key, &state.groups[i], state.catalog, input)
				if err != nil {
					return nil, err
				}
				if decision != nil {
					matching = append(matching, state.groups[i].ID)
				}
			}
			if len(matching) < 2 {
				continue
			}
			if modelGroups[model] == nil {
				modelGroups[model] = make(map[int64]bool)
			}
			for _, id := range matching {
				competing[id] = true
				modelGroups[model][id] = true
			}
		}
	}
	defaults := &AutoGroupRoutingPolicy{DefaultGroupOrder: state.policy.DefaultGroupOrder}
	result := &AutoRoutePriorities{
		DefaultSource: "group_sort",
		Groups:        priorityGroups(defaults.OrderGroups(state.groups, ""), competing),
		ModelRules:    []AutoRouteModelPriority{},
	}
	for _, id := range defaults.DefaultGroupOrder {
		if competing[id] {
			result.DefaultSource = "administrator"
			break
		}
	}
	for model, members := range modelGroups {
		rule := state.policy.modelRule(model)
		if rule == nil || len(rule.GroupIDs) == 0 {
			continue
		}
		// Avoid revealing rules that only name groups outside this user's scope.
		relevant := false
		for _, id := range rule.GroupIDs {
			relevant = relevant || members[id]
		}
		if !relevant {
			continue
		}
		result.ModelRules = append(result.ModelRules, AutoRouteModelPriority{
			Model: model, MatchedRule: rule.Model,
			Groups: priorityGroups(state.policy.OrderGroups(state.groups, model), members),
		})
	}
	sort.Slice(result.ModelRules, func(i, j int) bool { return result.ModelRules[i].Model < result.ModelRules[j].Model })
	return result, nil
}

func priorityGroups(groups []Group, included map[int64]bool) []AutoRoutePriorityGroup {
	result := make([]AutoRoutePriorityGroup, 0, len(included))
	for _, group := range groups {
		if included[group.ID] {
			result = append(result, AutoRoutePriorityGroup{ID: group.ID, Name: group.Name, Platform: group.Platform})
		}
	}
	return result
}
