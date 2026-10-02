package service

import (
	"context"
	"errors"
	"fmt"

	dbent "github.com/LuckyKuang/sub2api-plus/ent"
	"github.com/LuckyKuang/sub2api-plus/internal/config"
	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/logger"
)

func (s *adminServiceImpl) AdminUpdateAPIKeyRouting(ctx context.Context, keyID int64, input APIKeyRoutingUpdate) (*AdminUpdateAPIKeyGroupIDResult, error) {
	groupID := input.GroupID
	validationGroupID := groupID
	if groupID != nil && *groupID == 0 && (input.RoutingMode == nil || *input.RoutingMode == APIKeyRoutingFixed) {
		validationGroupID = nil
	}
	if err := ValidateAPIKeyRoutingInput(input.RoutingMode, input.RoutingMode != nil, validationGroupID); err != nil {
		return nil, err
	}
	if input.RoutingMode != nil && *input.RoutingMode == APIKeyRoutingAuto &&
		s.settingService != nil && s.settingService.cfg != nil && s.settingService.cfg.RunMode == config.RunModeSimple {
		return nil, infraerrors.Forbidden("AUTO_ROUTING_UNSUPPORTED_RUN_MODE", "automatic routing requires standard run mode")
	}
	apiKey, err := s.apiKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return nil, err
	}

	if input.RoutingMode == nil && groupID == nil {
		// nil 表示不修改，直接返回
		return &AdminUpdateAPIKeyGroupIDResult{APIKey: apiKey}, nil
	}
	if apiKey.IsAutoRouting() && input.RoutingMode == nil && groupID != nil && *groupID > 0 {
		return nil, infraerrors.BadRequest("ROUTING_MODE_CONFLICT", "select fixed routing before assigning a group")
	}
	if input.RoutingMode != nil && *input.RoutingMode == APIKeyRoutingFixed && groupID == nil && !input.GroupIDSet {
		if apiKey.IsAutoRouting() {
			return nil, infraerrors.BadRequest("ROUTING_GROUP_REQUIRED", "specify a group or explicitly unassign this key")
		}
		return &AdminUpdateAPIKeyGroupIDResult{APIKey: apiKey}, nil
	}
	apiKey.RoutingMode = APIKeyRoutingFixed
	if input.RoutingMode != nil {
		apiKey.RoutingMode = *input.RoutingMode
	}
	if apiKey.IsAutoRouting() || groupID == nil {
		unassigned := int64(0)
		groupID = &unassigned
	}

	result := &AdminUpdateAPIKeyGroupIDResult{}

	if *groupID == 0 {
		// 0 表示解绑分组（不修改 user_allowed_groups，避免影响用户其他 Key）
		apiKey.GroupID = nil
		apiKey.Group = nil
	} else {
		// 验证目标分组存在且状态为 active
		group, err := s.groupRepo.GetByID(ctx, *groupID)
		if err != nil {
			return nil, err
		}
		if group.Status != StatusActive {
			return nil, infraerrors.BadRequest("GROUP_NOT_ACTIVE", "target group is not active")
		}
		// 订阅类型分组：用户须持有该分组的有效订阅才可绑定
		if group.IsSubscriptionType() {
			if s.userSubRepo == nil {
				return nil, infraerrors.InternalServer("SUBSCRIPTION_REPOSITORY_UNAVAILABLE", "subscription repository is not configured")
			}
			if _, err := s.userSubRepo.GetActiveByUserIDAndGroupID(ctx, apiKey.UserID, *groupID); err != nil {
				if errors.Is(err, ErrSubscriptionNotFound) {
					return nil, infraerrors.BadRequest("SUBSCRIPTION_REQUIRED", "user does not have an active subscription for this group")
				}
				return nil, err
			}
		}

		gid := *groupID
		apiKey.GroupID = &gid
		apiKey.Group = group

		// 专属标准分组：使用事务保证「添加分组权限」与「更新 API Key」的原子性
		if group.IsExclusive && !group.IsSubscriptionType() {
			opCtx := ctx
			var tx *dbent.Tx
			if s.entClient == nil {
				logger.LegacyPrintf("service.admin", "Warning: entClient is nil, skipping transaction protection for exclusive group binding")
			} else {
				var txErr error
				tx, txErr = s.entClient.Tx(ctx)
				if txErr != nil {
					return nil, fmt.Errorf("begin transaction: %w", txErr)
				}
				defer func() { _ = tx.Rollback() }()
				opCtx = dbent.NewTxContext(ctx, tx)
			}

			if addErr := s.userRepo.AddGroupToAllowedGroups(opCtx, apiKey.UserID, gid); addErr != nil {
				return nil, fmt.Errorf("add group to user allowed groups: %w", addErr)
			}
			if err := s.apiKeyRepo.Update(opCtx, apiKey, APIKeyUpdateFields{GroupID: true, RoutingMode: true}); err != nil {
				return nil, fmt.Errorf("update api key: %w", err)
			}
			if tx != nil {
				if err := tx.Commit(); err != nil {
					return nil, fmt.Errorf("commit transaction: %w", err)
				}
			}

			result.AutoGrantedGroupAccess = true
			result.GrantedGroupID = &gid
			result.GrantedGroupName = group.Name

			// 失效认证缓存（在事务提交后执行）
			if s.authCacheInvalidator != nil {
				s.authCacheInvalidator.InvalidateAuthCacheByKey(ctx, apiKey.Key)
			}

			result.APIKey = apiKey
			return result, nil
		}
	}

	// 非专属分组 / 解绑：无需事务，单步更新即可
	if err := s.apiKeyRepo.Update(ctx, apiKey, APIKeyUpdateFields{GroupID: true, RoutingMode: true}); err != nil {
		return nil, fmt.Errorf("update api key: %w", err)
	}

	// 失效认证缓存
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByKey(ctx, apiKey.Key)
	}

	result.APIKey = apiKey
	return result, nil
}
