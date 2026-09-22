package service

import infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"

type APIKeyRoutingUpdate struct {
	RoutingMode *string
	GroupID     *int64
	GroupIDSet  bool
}

func ValidateAPIKeyRoutingInput(mode *string, modeSet bool, groupID *int64) error {
	if modeSet || mode != nil {
		if mode == nil || (*mode != APIKeyRoutingFixed && *mode != APIKeyRoutingAuto) {
			return infraerrors.BadRequest("INVALID_ROUTING_MODE", "routing_mode must be fixed or auto")
		}
		if *mode == APIKeyRoutingAuto && groupID != nil {
			return infraerrors.BadRequest("ROUTING_MODE_CONFLICT", "automatic routing cannot bind a fixed group")
		}
	}
	if groupID != nil && *groupID <= 0 {
		return infraerrors.BadRequest("INVALID_GROUP_ID", "group_id must be a positive integer")
	}
	return nil
}
