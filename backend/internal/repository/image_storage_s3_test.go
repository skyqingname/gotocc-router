package repository

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/LuckyKuang/sub2api-plus/internal/config"
	"github.com/stretchr/testify/require"
)

func TestS3ImageStorageSaveReaderSendsBoundedStream(t *testing.T) {
	payload := "streamed-image-payload"
	var received string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPut, r.Method)
		require.Equal(t, int64(len(payload)), r.ContentLength)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		received = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	storage, err := NewS3ImageStorage(context.Background(), &config.ImageStorageConfig{
		Endpoint:        server.URL,
		Region:          "auto",
		Bucket:          "test-bucket",
		AccessKeyID:     "test-access-key",
		SecretAccessKey: "test-secret-key",
		ForcePathStyle:  true,
		PublicBaseURL:   "https://cdn.test",
	})
	require.NoError(t, err)

	reader := io.LimitReader(strings.NewReader(payload), int64(len(payload)))
	url, err := storage.SaveReader(context.Background(), "images/test.png", "image/png", reader, int64(len(payload)))
	require.NoError(t, err)
	require.Equal(t, payload, received)
	require.Equal(t, "https://cdn.test/images/test.png", url)
}

func TestS3ImageStorageCheckRequiresGetObjectPermission(t *testing.T) {
	var getCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getCalls.Add(1)
			http.Error(w, "read access denied", http.StatusForbidden)
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	storage, err := NewS3ImageStorage(context.Background(), &config.ImageStorageConfig{
		Endpoint:        server.URL,
		Region:          "us-east-1",
		Bucket:          "images",
		AccessKeyID:     "access-key",
		SecretAccessKey: "secret-key",
		Prefix:          "async-images",
		ForcePathStyle:  true,
	})
	require.NoError(t, err)

	err = storage.Check(context.Background())
	require.Error(t, err)
	require.ErrorContains(t, err, "S3 GetObject health check failed")
	require.Greater(t, getCalls.Load(), int32(0))
}
