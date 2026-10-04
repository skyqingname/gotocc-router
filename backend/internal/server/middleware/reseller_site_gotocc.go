package middleware

import (
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/resellersite"
	"github.com/gin-gonic/gin"
)

func ResellerSite() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request = c.Request.WithContext(resellersite.WithHost(c.Request.Context(), c.Request.Host))
		c.Next()
	}
}
