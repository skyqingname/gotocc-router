package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/logger"
	"go.uber.org/zap"
)

// AutoResponseAffinityCache stores a single atomic record in shared storage.
// Missing data is nil; dependency errors must never become a cache miss.
type AutoResponseAffinityCache interface {
	StoreAutoResponseAffinity(context.Context, string, []byte, time.Duration) error
	LoadAutoResponseAffinity(context.Context, string) ([]byte, error)
}

type AutoResponseAffinity struct {
	Platform       string    `json:"platform"`
	SubscriptionID int64     `json:"subscription_id"`
	APIKeyID       int64     `json:"api_key_id"`
	ActorUserID    int64     `json:"actor_user_id"`
	BillingUserID  int64     `json:"billing_user_id"`
	TeamID         int64     `json:"team_id"`
	GroupID        int64     `json:"group_id"`
	AccountID      int64     `json:"account_id"`
	ExpiresAt      time.Time `json:"expires_at"`
}

func autoResponseAffinityKey(keyID int64, responseID string) (string, error) {
	responseID = strings.TrimSpace(responseID)
	if keyID <= 0 || responseID == "" || len(responseID) > 1024 {
		return "", ErrAutoRouteContext
	}
	hash := sha256.Sum256([]byte(responseID))
	return strconv.FormatInt(keyID, 10) + ":" + hex.EncodeToString(hash[:]), nil
}

func (s *OpenAIGatewayService) BindAutoResponseAffinity(ctx context.Context, key *APIKey, responseID string, accountID int64) error {
	if s == nil {
		return ErrAutoRouteUnavailable
	}
	return s.bindAutoResponseAffinity(ctx, key, responseID, accountID, s.openAIWSResponseStickyTTL())
}

func (s *OpenAIGatewayService) bindAutoResponseAffinity(ctx context.Context, key *APIKey, responseID string, accountID int64, ttl time.Duration) error {
	if key == nil || !key.IsAutoRouting() || key.User == nil || key.UserID <= 0 || key.GroupID == nil || *key.GroupID <= 0 || accountID <= 0 {
		return ErrAutoRouteContext
	}
	index, err := autoResponseAffinityKey(key.ID, responseID)
	if err != nil {
		return err
	}
	if s == nil {
		return ErrAutoRouteUnavailable
	}
	cache, ok := s.cache.(AutoResponseAffinityCache)
	if !ok {
		return ErrAutoRouteUnavailable
	}
	binding := AutoResponseAffinity{
		APIKeyID: key.ID, ActorUserID: key.UserID, BillingUserID: key.User.ID,
		GroupID: *key.GroupID, AccountID: accountID, ExpiresAt: time.Now().Add(ttl),
	}
	if key.Group != nil {
		binding.Platform = key.Group.Platform
	}
	if route, ok := AutoRouteDecisionFromContext(ctx); ok {
		binding.Platform = route.Platform
	}
	binding.SubscriptionID, _ = ctx.Value(autoRouteSubscriptionKey{}).(int64)
	if key.TeamID != nil {
		binding.TeamID = *key.TeamID
	}
	data, err := json.Marshal(binding)
	if err != nil {
		return ErrAutoRouteUnavailable.WithCause(err)
	}
	cacheCtx, cancel := withOpenAIWSStateStoreRedisTimeout(ctx)
	defer cancel()
	if err := cache.StoreAutoResponseAffinity(cacheCtx, index, data, ttl); err != nil {
		return ErrAutoRouteUnavailable.WithCause(err)
	}
	return nil
}

func (s *OpenAIGatewayService) LookupAutoResponseAffinity(ctx context.Context, key *APIKey, responseID string) (*AutoResponseAffinity, error) {
	if key == nil || !key.IsAutoRouting() || key.User == nil {
		return nil, ErrAutoRouteNoAccess
	}
	index, err := autoResponseAffinityKey(key.ID, responseID)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, ErrAutoRouteUnavailable
	}
	cache, ok := s.cache.(AutoResponseAffinityCache)
	if !ok {
		return nil, ErrAutoRouteUnavailable
	}
	cacheCtx, cancel := withOpenAIWSStateStoreRedisTimeout(ctx)
	defer cancel()
	data, err := cache.LoadAutoResponseAffinity(cacheCtx, index)
	if err != nil {
		return nil, ErrAutoRouteUnavailable.WithCause(err)
	}
	if len(data) == 0 {
		return nil, ErrAutoRouteContext
	}
	var binding AutoResponseAffinity
	if json.Unmarshal(data, &binding) != nil || binding.GroupID <= 0 || binding.AccountID <= 0 || binding.ExpiresAt.IsZero() {
		return nil, ErrAutoRouteUnavailable
	}
	if !time.Now().Before(binding.ExpiresAt) {
		return nil, ErrAutoRouteContext
	}
	teamID := int64(0)
	if key.TeamID != nil {
		teamID = *key.TeamID
	}
	if binding.APIKeyID != key.ID || binding.ActorUserID != key.UserID || binding.BillingUserID != key.User.ID || binding.TeamID != teamID {
		return nil, ErrAutoRouteNoAccess
	}
	return &binding, nil
}

func (s *OpenAIGatewayService) bindAutoResponseAffinityFromContext(ctx context.Context, accountID int64, responseID string) {
	route, ok := AutoRouteDecisionFromContext(ctx)
	if !ok {
		return
	}
	if err := s.BindAutoResponseAffinity(ctx, route.Key, responseID, accountID); err != nil {
		logger.FromContext(ctx).Error("automatic response affinity write failed",
			zap.String("stage", "response_affinity"), zap.String("error_code", "AUTO_ROUTE_UNAVAILABLE"),
			zap.Int64("api_key_id", route.Key.ID), zap.Int64("account_id", accountID))
	}
}
