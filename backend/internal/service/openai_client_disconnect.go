package service

import (
	"errors"

	"github.com/gin-gonic/gin"
)

// ErrOpenAIClientDisconnected identifies a downstream write failure separately
// from upstream transport and terminal response failures. Usage still settles.
var ErrOpenAIClientDisconnected = errors.New("stream usage incomplete: client disconnected")

const OpsClientDisconnectedKey = "ops_client_disconnected"

func markOpenAIClientDisconnected(c *gin.Context) error {
	c.Set(OpsClientDisconnectedKey, true)
	MarkOpsStreamError(c, "client_disconnect", "Downstream connection closed (client disconnected)", 499)
	return ErrOpenAIClientDisconnected
}
