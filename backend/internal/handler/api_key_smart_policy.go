package handler

import (
	"encoding/json"
	"io"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/response"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *GatewayHandler) AdminGetRoutingPolicy(c *gin.Context) {
	policy, err := h.autoGroupResolver.GetRoutingPolicy(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	response.Success(c, policy)
}

func (h *GatewayHandler) AdminUpdateRoutingPolicy(c *gin.Context) {
	var policy *service.AutoGroupRoutingPolicy
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&policy); err != nil {
		response.BadRequest(c, "Invalid routing policy")
		return
	}
	if policy == nil {
		response.BadRequest(c, "Invalid routing policy")
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		response.BadRequest(c, "Invalid routing policy")
		return
	}
	if err := h.autoGroupResolver.UpdateRoutingPolicy(c.Request.Context(), *policy); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"updated": true})
}
