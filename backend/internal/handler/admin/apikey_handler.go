package admin

import (
	"strconv"

	"github.com/LuckyKuang/sub2api-plus/internal/handler/dto"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/response"
	"github.com/LuckyKuang/sub2api-plus/internal/service"

	"github.com/gin-gonic/gin"
)

// AdminAPIKeyHandler handles admin API key management
type AdminAPIKeyHandler struct {
	adminService service.AdminService
}

// NewAdminAPIKeyHandler creates a new admin API key handler
func NewAdminAPIKeyHandler(adminService service.AdminService) *AdminAPIKeyHandler {
	return &AdminAPIKeyHandler{
		adminService: adminService,
	}
}

// AdminUpdateAPIKeyGroupRequest represents the request to update an API key.
type AdminUpdateAPIKeyGroupRequest struct {
	RoutingMode         dto.NullableStringField `json:"routing_mode"`
	GroupID             dto.NullableInt64Field  `json:"group_id"`
	ResetRateLimitUsage *bool                   `json:"reset_rate_limit_usage"` // true=重置 5h/1d/7d 限速用量
}

// UpdateGroup handles updating an API key's admin-managed fields.
// PUT /api/v1/admin/api-keys/:id
func (h *AdminAPIKeyHandler) UpdateGroup(c *gin.Context) {
	keyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid API key ID")
		return
	}

	var req AdminUpdateAPIKeyGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	validationGroupID := req.GroupID.Value
	if validationGroupID != nil && *validationGroupID == 0 &&
		(req.RoutingMode.Value == nil || *req.RoutingMode.Value == service.APIKeyRoutingFixed) {
		validationGroupID = nil
	}
	if err := service.ValidateAPIKeyRoutingInput(req.RoutingMode.Value, req.RoutingMode.Set, validationGroupID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	result, err := h.adminService.AdminUpdateAPIKeyRouting(c.Request.Context(), keyID, service.APIKeyRoutingUpdate{
		RoutingMode: req.RoutingMode.Value, GroupID: req.GroupID.Value, GroupIDSet: req.GroupID.Set,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if req.ResetRateLimitUsage != nil && *req.ResetRateLimitUsage {
		resetKey, err := h.adminService.AdminResetAPIKeyRateLimitUsage(c.Request.Context(), keyID)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		result.APIKey = resetKey
	}

	resp := struct {
		APIKey                 *dto.AdminAPIKeySummary `json:"api_key"`
		AutoGrantedGroupAccess bool                    `json:"auto_granted_group_access"`
		GrantedGroupID         *int64                  `json:"granted_group_id,omitempty"`
		GrantedGroupName       string                  `json:"granted_group_name,omitempty"`
	}{
		APIKey:                 dto.AdminAPIKeySummaryFromService(result.APIKey),
		AutoGrantedGroupAccess: result.AutoGrantedGroupAccess,
		GrantedGroupID:         result.GrantedGroupID,
		GrantedGroupName:       result.GrantedGroupName,
	}
	response.Success(c, resp)
}
