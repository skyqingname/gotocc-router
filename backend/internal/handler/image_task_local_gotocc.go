package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ReadLocalImage consumes a short-lived URL minted after the existing object
// owner check. It has the same bearer-URL semantics as an S3 presigned download.
func (h *AsyncImageHandler) ReadLocalImage(c *gin.Context) {
	object, err := h.tasks.OpenSignedLocalImage(c.Request.Context(), c.Query("token"))
	if err != nil {
		imageTaskError(c, err)
		return
	}
	defer object.Body.Close()
	c.DataFromReader(http.StatusOK, object.Size, object.ContentType, object.Body, map[string]string{
		"Cache-Control": "private, no-store",
	})
}
