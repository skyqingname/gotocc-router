package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/response"
	"github.com/LuckyKuang/sub2api-plus/internal/server/middleware"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
)

type autoModelCatalog interface {
	ListModels(context.Context, *service.APIKey, service.AutoRouteRequest) ([]service.AutoRouteModel, error)
	BuildCodexModelsManifest(context.Context, *service.APIKey) ([]byte, error)
}

func serveAutoModels(c *gin.Context, catalog autoModelCatalog) {
	c.Abort()
	key, ok := middleware.GetAPIKeyFromContext(c)
	if !ok || key == nil || !key.IsAutoRouting() {
		middleware.WriteAutoRoutingError(c, service.ErrAutoRouteNoAccess)
		return
	}
	if catalog == nil {
		middleware.WriteAutoRoutingError(c, service.ErrAutoRouteUnavailable)
		return
	}
	c.Header("Cache-Control", "private, no-cache")
	if strings.HasSuffix(c.FullPath(), "/images/batches/models") {
		batchCatalog, ok := catalog.(interface {
			ListBatchModels(context.Context, *service.APIKey) ([]service.BatchImagePublicModel, error)
		})
		if !ok {
			middleware.WriteAutoRoutingError(c, service.ErrAutoRouteUnavailable)
			return
		}
		models, err := batchCatalog.ListBatchModels(c.Request.Context(), key)
		if err != nil {
			middleware.WriteAutoRoutingError(c, err)
			return
		}
		c.JSON(http.StatusOK, service.BatchImagePublicModelsResponse{Object: "list", Data: models})
		return
	}
	if c.Query("client_version") != "" || strings.HasPrefix(c.FullPath(), "/backend-api/codex/") {
		body, err := catalog.BuildCodexModelsManifest(c.Request.Context(), key)
		if err != nil {
			middleware.WriteAutoRoutingError(c, err)
			return
		}
		hash := sha256.Sum256(body)
		etag := `"` + hex.EncodeToString(hash[:]) + `"`
		c.Header("ETag", etag)
		if c.GetHeader("If-None-Match") == etag {
			c.Status(http.StatusNotModified)
			return
		}
		c.Data(http.StatusOK, "application/json", body)
		return
	}
	input := service.AutoRouteRequest{Endpoint: service.CompositeRouteEndpointResponses}
	SetClaudeCodeClientContext(c, nil, nil)
	input.ClaudeCodeClient = service.IsClaudeCodeClient(c.Request.Context())
	input.ForcePlatform, _ = middleware.GetForcePlatformFromContext(c)
	google := strings.Contains(c.FullPath(), "/v1beta/")
	if google {
		input.Endpoint = service.CompositeRouteEndpointGemini
	}
	models, err := catalog.ListModels(c.Request.Context(), key, input)
	if err != nil {
		middleware.WriteAutoRoutingError(c, err)
		return
	}
	entries := make([]gin.H, 0, len(models))
	for _, model := range models {
		if google {
			entry := gin.H{"name": "models/" + model.ID, "displayName": model.ID, "supportedGenerationMethods": []string{"generateContent", "streamGenerateContent", "countTokens"}}
			if requested := c.Param("model"); requested != "" {
				if requested == model.ID {
					c.JSON(http.StatusOK, entry)
					return
				}
				continue
			}
			entries = append(entries, entry)
		} else {
			entries = append(entries, gin.H{"id": model.ID, "object": "model", "created": int64(0), "owned_by": model.Platform})
		}
	}
	if google {
		if c.Param("model") != "" {
			middleware.WriteAutoRoutingError(c, service.ErrAutoRouteModelNotFound)
			return
		}
		c.JSON(http.StatusOK, gin.H{"models": entries})
		return
	}
	c.JSON(http.StatusOK, gin.H{"object": "list", "data": entries})
}

func (h *GatewayHandler) UserRoutingCapabilities(c *gin.Context) {
	h.routingCapabilities(c, false)
}

func (h *GatewayHandler) AdminRoutingCapabilities(c *gin.Context) {
	h.routingCapabilities(c, true)
}

func (h *GatewayHandler) routingCapabilities(c *gin.Context, admin bool) {
	subject, authenticated := middleware.GetAuthSubjectFromContext(c)
	if !admin && !authenticated {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid key ID")
		return
	}
	if h.apiKeyService == nil || h.autoGroupResolver == nil {
		response.ErrorFrom(c, service.ErrAutoRouteUnavailable)
		return
	}
	key, err := h.apiKeyService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if key == nil || (!admin && key.UserID != subject.UserID) {
		response.NotFound(c, "API key not found")
		return
	}
	if !key.IsAutoRouting() {
		response.BadRequest(c, "Routing capabilities require an automatic routing key")
		return
	}
	capabilities, err := h.autoGroupResolver.GetRoutingCapabilities(c.Request.Context(), key)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if h.cfg != nil {
		capabilities.BatchImageSubmit = capabilities.BatchImageSubmit && h.cfg.BatchImage.Enabled
		capabilities.AsyncImageSubmit = capabilities.AsyncImageSubmit && h.cfg.ImageStorage.Enabled
	}
	c.Header("Cache-Control", "no-store")
	response.Success(c, capabilities)
}
