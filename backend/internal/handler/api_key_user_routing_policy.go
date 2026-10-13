package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/response"
	"github.com/LuckyKuang/sub2api-plus/internal/server/middleware"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *GatewayHandler) UserGetKeyRoutingPreference(c *gin.Context) {
	subject, ok := middleware.GetReadSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid key ID")
		return
	}
	state, err := h.autoGroupResolver.GetKeyRoutingPreference(c.Request.Context(), subject.UserID, id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	response.Success(c, state)
}

func (h *GatewayHandler) UserUpdateKeyRoutingPreference(c *gin.Context) {
	// Writes always use the authenticated actor, never support impersonation.
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid key ID")
		return
	}
	var preference *service.AutoGroupRoutingPreference
	decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&preference) != nil || decoder.Decode(&struct{}{}) != io.EOF {
		response.BadRequest(c, "Invalid routing preference")
		return
	}
	if err := h.autoGroupResolver.UpdateKeyRoutingPreference(c.Request.Context(), subject.UserID, id, preference); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"updated": true})
}
