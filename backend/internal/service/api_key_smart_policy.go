package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"sort"
	"strings"

	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
)

const autoGroupRoutingPolicySetting = "api_key_auto_group_routing"

type AutoGroupRoutingRule struct {
	Model    string  `json:"model"`
	GroupIDs []int64 `json:"group_ids"`
}

type AutoGroupRoutingPolicy struct {
	DefaultGroupOrder []int64                `json:"default_group_order"`
	ModelRules        []AutoGroupRoutingRule `json:"model_rules"`
}

type AutoGroupRoutingPolicyService struct {
	settings SettingRepository
	groups   GroupRepository
}

func NewAutoGroupRoutingPolicyService(settings SettingRepository, groups GroupRepository) *AutoGroupRoutingPolicyService {
	return &AutoGroupRoutingPolicyService{settings: settings, groups: groups}
}

func emptyAutoGroupRoutingPolicy() *AutoGroupRoutingPolicy {
	return &AutoGroupRoutingPolicy{DefaultGroupOrder: []int64{}, ModelRules: []AutoGroupRoutingRule{}}
}

func (s *AutoGroupRoutingPolicyService) Get(ctx context.Context) (*AutoGroupRoutingPolicy, error) {
	if s == nil || s.settings == nil {
		return nil, ErrAutoRouteUnavailable
	}
	raw, err := s.settings.GetValue(ctx, autoGroupRoutingPolicySetting)
	if errors.Is(err, ErrSettingNotFound) {
		return emptyAutoGroupRoutingPolicy(), nil
	}
	if err != nil {
		return nil, ErrAutoRouteUnavailable.WithCause(err)
	}
	policy := emptyAutoGroupRoutingPolicy()
	if len(raw) > 1<<20 || !strings.HasPrefix(strings.TrimSpace(raw), "{") {
		return nil, ErrAutoRouteUnavailable
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(policy) != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return nil, ErrAutoRouteUnavailable
	}
	if err := policy.Validate(); err != nil {
		return nil, ErrAutoRouteUnavailable.WithCause(err)
	}
	return policy, nil
}

func (p *AutoGroupRoutingPolicy) Validate() error {
	invalid := func() error {
		return infraerrors.BadRequest("INVALID_AUTO_ROUTING_POLICY", "Invalid model routing order: use unique models and positive, unique group IDs")
	}
	if p == nil || len(p.ModelRules) > 256 || len(p.DefaultGroupOrder) > 256 {
		return invalid()
	}
	validateIDs := func(ids []int64) bool {
		if len(ids) > 256 {
			return false
		}
		seen := make(map[int64]bool, len(ids))
		for _, id := range ids {
			if id <= 0 || seen[id] {
				return false
			}
			seen[id] = true
		}
		return true
	}
	if !validateIDs(p.DefaultGroupOrder) {
		return invalid()
	}
	seen := make(map[string]bool, len(p.ModelRules))
	for i := range p.ModelRules {
		rule := &p.ModelRules[i]
		rule.Model = strings.TrimSpace(rule.Model)
		if rule.Model == "" || len(rule.Model) > 256 || strings.ContainsAny(rule.Model, "\r\n\t") || seen[rule.Model] || !validateIDs(rule.GroupIDs) {
			return invalid()
		}
		if count := strings.Count(rule.Model, "*"); count > 1 || (count == 1 && !strings.HasSuffix(rule.Model, "*")) {
			return invalid()
		}
		seen[rule.Model] = true
		if rule.GroupIDs == nil {
			rule.GroupIDs = []int64{}
		}
	}
	if p.DefaultGroupOrder == nil {
		p.DefaultGroupOrder = []int64{}
	}
	if p.ModelRules == nil {
		p.ModelRules = []AutoGroupRoutingRule{}
	}
	return nil
}

func (s *AutoGroupRoutingPolicyService) Update(ctx context.Context, policy AutoGroupRoutingPolicy) error {
	if err := policy.Validate(); err != nil {
		return err
	}
	if s == nil || s.settings == nil || s.groups == nil {
		return ErrAutoRouteUnavailable
	}
	ids := make(map[int64]struct{})
	for _, id := range policy.DefaultGroupOrder {
		ids[id] = struct{}{}
	}
	for _, rule := range policy.ModelRules {
		for _, id := range rule.GroupIDs {
			ids[id] = struct{}{}
		}
	}
	for id := range ids {
		group, err := s.groups.GetByIDLite(ctx, id)
		if errors.Is(err, ErrGroupNotFound) || (err == nil && group == nil) {
			return infraerrors.BadRequest("INVALID_AUTO_ROUTING_GROUP", "A configured routing group does not exist")
		}
		if err != nil {
			return ErrAutoRouteUnavailable.WithCause(err)
		}
	}
	data, err := json.Marshal(policy)
	if err != nil {
		return ErrAutoRouteUnavailable.WithCause(err)
	}
	if err := s.settings.Set(ctx, autoGroupRoutingPolicySetting, string(data)); err != nil {
		return ErrAutoRouteUnavailable.WithCause(err)
	}
	return nil
}

// OrderGroups only reorders the caller's authorized candidates. It never adds
// a configured group to the set, and leaves shared catalog snapshots unchanged.
func (p *AutoGroupRoutingPolicy) OrderGroups(groups []Group, model string) []Group {
	result := append([]Group(nil), groups...)
	if p == nil {
		return result
	}
	var preferred []int64
	longest := -1
	for _, rule := range p.ModelRules {
		if rule.Model == model {
			preferred = rule.GroupIDs
			break
		}
		if strings.HasSuffix(rule.Model, "*") {
			prefix := strings.TrimSuffix(rule.Model, "*")
			if len(prefix) > longest && strings.HasPrefix(model, prefix) {
				preferred = rule.GroupIDs
				longest = len(prefix)
			}
		}
	}
	ranks := make(map[int64]int)
	for _, list := range [][]int64{preferred, p.DefaultGroupOrder} {
		for _, id := range list {
			if _, ok := ranks[id]; !ok {
				ranks[id] = len(ranks)
			}
		}
	}
	rank := func(id int64) int {
		if value, ok := ranks[id]; ok {
			return value
		}
		return len(ranks)
	}
	sort.SliceStable(result, func(i, j int) bool { return rank(result[i].ID) < rank(result[j].ID) })
	return result
}

func (s *AutoGroupResolver) routingPolicy(ctx context.Context, locked bool) (*AutoGroupRoutingPolicy, error) {
	if locked || s.policy == nil {
		return emptyAutoGroupRoutingPolicy(), nil
	}
	return s.policy.Get(ctx)
}

func (s *AutoGroupResolver) GetRoutingPolicy(ctx context.Context) (*AutoGroupRoutingPolicy, error) {
	if s == nil || s.policy == nil {
		return nil, ErrAutoRouteUnavailable
	}
	return s.policy.Get(ctx)
}

func (s *AutoGroupResolver) UpdateRoutingPolicy(ctx context.Context, policy AutoGroupRoutingPolicy) error {
	if s == nil || s.policy == nil {
		return ErrAutoRouteUnavailable
	}
	return s.policy.Update(ctx, policy)
}
