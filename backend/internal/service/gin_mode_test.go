package service

import (
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

// Gin mode is process-global. Set it before parallel tests and their HTTP
// callbacks start; parallel tests must not write it while others are reading it.
func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}
