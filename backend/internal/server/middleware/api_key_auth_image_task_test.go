package middleware

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsAsyncImageReadRequest(t *testing.T) {
	require.True(t, isAsyncImageReadRequest(http.MethodGet, "/v1/images/tasks/imgtask_123"))
	require.True(t, isAsyncImageReadRequest(http.MethodGet, "/images/tasks/imgtask_123"))
	require.True(t, isAsyncImageReadRequest(http.MethodGet, "/v1/images/objects/imgobj_123/url"))
	require.True(t, isAsyncImageReadRequest(http.MethodGet, "/images/objects/imgobj_123/url"))
	require.False(t, isAsyncImageReadRequest(http.MethodPost, "/v1/images/tasks/imgtask_123"))
	require.False(t, isAsyncImageReadRequest(http.MethodPost, "/v1/images/objects/imgobj_123/url"))
	require.False(t, isAsyncImageReadRequest(http.MethodGet, "/v1/images/objects/imgobj_123"))
	require.False(t, isAsyncImageReadRequest(http.MethodGet, "/v1/images/objects/nested/imgobj_123/url"))
	require.False(t, isAsyncImageReadRequest(http.MethodGet, "/v1/images/generations"))
}
