package service

import (
	"github.com/gin-gonic/gin"
)

// Gin mode is process-global. Set it before parallel tests and their HTTP
// callbacks start; parallel tests must not write it while others are reading it.
func init() {
	gin.SetMode(gin.TestMode)
}
