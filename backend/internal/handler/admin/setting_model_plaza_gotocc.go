package admin

import (
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/response"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
)

// plazaPurposes 是模型广场支持的用途，与 models.dev 的输出模态对应（文本记为语言）。
var plazaPurposes = map[string]bool{"language": true, "image": true, "video": true, "audio": true}

// GetModelPlazaOverrides 返回后台按模型填写的展示信息，键为小写模型名。
// GET /api/v1/admin/model-plaza/overrides
func (h *SettingHandler) GetModelPlazaOverrides(c *gin.Context) {
	overrides, err := h.settingService.GetModelPlazaOverrides(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, overrides)
}

// UpdateModelPlazaOverrides 整体替换后台按模型填写的展示信息，返回保存后的内容。
// PUT /api/v1/admin/model-plaza/overrides
func (h *SettingHandler) UpdateModelPlazaOverrides(c *gin.Context) {
	var req map[string]service.PlazaModelOverride
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid model plaza overrides")
		return
	}
	for _, override := range req {
		for _, purpose := range override.Purposes {
			if !plazaPurposes[purpose] {
				response.BadRequest(c, "Unknown model purpose: "+purpose)
				return
			}
		}
	}
	if err := h.settingService.SetModelPlazaOverrides(c.Request.Context(), req); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.GetModelPlazaOverrides(c)
}
