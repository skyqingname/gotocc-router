package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Old CC Switch imports append /v1/usage to a base URL already ending in /v1.
// Normalize only that published read-only alias before the usual authentication
// and billing middleware, so empty wallets can still query their remaining usage.
func canonicalCCSwitchUsageAlias(c *gin.Context) {
	if c.Request.Method == http.MethodGet && c.Request.URL.Path == "/v1/v1/usage" {
		c.Request.URL.Path = "/v1/usage"
		c.Request.URL.RawPath = ""
	}
	c.Next()
}
