package admin

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/response"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/videoconfig"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
)

// VideoGroupConfigHandler is a management endpoint, not a paid generation route.
// It intentionally exposes no token, auth header, Base URL or raw account object.
func (h *GroupHandler) VideoGroupConfigHandler(settings *service.SettingService, preview bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || groupID <= 0 { c.AbortWithStatus(http.StatusBadRequest); return }
		if _, err := h.adminService.GetGroup(c.Request.Context(), groupID); err != nil { response.ErrorFrom(c, err); return }
		if c.Request.Method == http.MethodGet && !preview {
			view, err := settings.GetVideoGroupConfig(c.Request.Context(), groupID)
			if err != nil { response.ErrorFrom(c, err); return }
			response.Success(c, view)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 128*1024)
		decoder := json.NewDecoder(c.Request.Body)
		decoder.DisallowUnknownFields()
		badInput := func() { c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": "invalid_video_config", "message": "Invalid video configuration request"}) }
		if preview && c.Request.Method == http.MethodPost {
			var input struct {
				BindingID string `json:"binding_id"`
				Parameters map[string]json.RawMessage `json:"parameters"`
			}
			if decoder.Decode(&input) != nil || decoder.Decode(new(any)) != io.EOF { badInput(); return }
			quote, err := settings.PreviewVideoBinding(c.Request.Context(), groupID, input.BindingID, input.Parameters)
			if err != nil { response.ErrorFrom(c, err); return }
			response.Success(c, gin.H{"quote": quote, "charged": false, "submitted": false})
			return
		}
		if preview || c.Request.Method != http.MethodPut { c.AbortWithStatus(http.StatusMethodNotAllowed); return }
		var input struct {
			ExpectedVersion *int64 `json:"expected_version"`
			Config *videoconfig.Config `json:"config"`
		}
		if decoder.Decode(&input) != nil || decoder.Decode(new(any)) != io.EOF || input.ExpectedVersion == nil || input.Config == nil { badInput(); return }
		if _, err := videoconfig.Compile(*input.Config); err != nil { badInput(); return }
		// Existing credential ownership and group bindings remain authoritative.
		// A saved account ID never grants access by itself. Execution must recheck
		// eligibility; configuration edits do not move accounts between groups.
		for _, binding := range input.Config.Bindings {
			account, err := h.adminService.GetAccount(c.Request.Context(), binding.AccountID)
			if err != nil { response.ErrorFrom(c, err); return }
			bound := false
			if account != nil {
				for _, id := range account.GroupIDs { if id == groupID { bound = true; break } }
			}
			if !bound || account.Type != service.AccountTypeAPIKey ||
				(binding.Protocol == videoconfig.OpenAIJSON && account.Platform != service.PlatformOpenAI) ||
				(binding.Protocol == videoconfig.XAIVideo && account.Platform != service.PlatformGrok) {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"code": "video_account_mismatch", "message": "Each binding must reference an API-key account in this group with the matching provider protocol"})
				return
			}
		}
		view, err := settings.SaveVideoGroupConfig(c.Request.Context(), groupID, *input.ExpectedVersion, *input.Config)
		if err != nil { response.ErrorFrom(c, err); return }
		response.Success(c, view)
	}
}
