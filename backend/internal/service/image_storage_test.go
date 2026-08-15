package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// pngBytes is a minimal payload whose signature makes http.DetectContentType
// report image/png.
var pngBytes = []byte("\x89PNG\r\n\x1a\nfake-png-payload")

type savedImage struct {
	key         string
	contentType string
	data        []byte
}

type fakeImageStorage struct {
	saved     []savedImage
	url       string
	expiresAt int64
	err       error
}

type fakeStreamingImageStorage struct {
	fakeImageStorage
	streamCalls int
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func (f *fakeImageStorage) Save(_ context.Context, key, contentType string, data []byte) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	f.saved = append(f.saved, savedImage{key: key, contentType: contentType, data: append([]byte(nil), data...)})
	if f.url != "" {
		return f.url, nil
	}
	return "https://cdn.test/" + key, nil
}

func (f *fakeImageStorage) SignURL(_ context.Context, key string) (string, int64, error) {
	if f.err != nil {
		return "", 0, f.err
	}
	if f.url != "" {
		return f.url, f.expiresAt, nil
	}
	return "https://cdn.test/" + key, f.expiresAt, nil
}

func (f *fakeStreamingImageStorage) SaveReader(_ context.Context, key, contentType string, body io.Reader, size int64) (string, error) {
	f.streamCalls++
	data, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}
	if int64(len(data)) != size {
		return "", fmt.Errorf("read %d bytes; expected %d", len(data), size)
	}
	f.saved = append(f.saved, savedImage{key: key, contentType: contentType, data: data})
	return "https://cdn.test/" + key, nil
}

func TestImageResultUploaderRewritesB64JSON(t *testing.T) {
	storage := &fakeImageStorage{}
	uploader := NewImageResultUploader(storage, "images/", 0, nil)

	b64 := base64.StdEncoding.EncodeToString(pngBytes)
	result := json.RawMessage(`{"created":1,"data":[{"b64_json":"` + b64 + `","revised_prompt":"a cat"}]}`)

	out, err := uploader.Rewrite(context.Background(), "imgtask_abc", result)
	require.NoError(t, err)

	require.Len(t, storage.saved, 1)
	require.Equal(t, "images/imgtask_abc-0.png", storage.saved[0].key)
	require.Equal(t, "image/png", storage.saved[0].contentType)
	require.Equal(t, pngBytes, storage.saved[0].data)

	var parsed struct {
		Data []map[string]json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(out, &parsed))
	require.Len(t, parsed.Data, 1)
	require.JSONEq(t, `"https://cdn.test/images/imgtask_abc-0.png"`, string(parsed.Data[0]["url"]))
	require.NotEmpty(t, parsed.Data[0]["object_id"])
	require.JSONEq(t, `"images/imgtask_abc-0.png"`, string(parsed.Data[0]["storage_key"]))
	require.JSONEq(t, `"image/png"`, string(parsed.Data[0]["content_type"]))
	require.JSONEq(t, `24`, string(parsed.Data[0]["bytes"]))
	_, hasB64 := parsed.Data[0]["b64_json"]
	require.False(t, hasB64, "b64_json must be stripped after offload")
	require.JSONEq(t, `"a cat"`, string(parsed.Data[0]["revised_prompt"]), "unrelated fields preserved")
}

func TestImageResultUploaderStreamsB64JSONWhenStorageSupportsReader(t *testing.T) {
	storage := &fakeStreamingImageStorage{}
	uploader := NewImageResultUploader(storage, "images/", 0, nil)
	b64 := base64.StdEncoding.EncodeToString(pngBytes)
	result := json.RawMessage(`{"created":1,"data":[{"b64_json":"` + b64 + `"}]}`)

	out, err := uploader.Rewrite(context.Background(), "imgtask_stream", result)
	require.NoError(t, err)
	require.Equal(t, 1, storage.streamCalls)
	require.Len(t, storage.saved, 1)
	require.Equal(t, pngBytes, storage.saved[0].data)
	require.NotContains(t, string(out), "b64_json")
	require.Contains(t, string(out), `"bytes":24`)
}

func TestImageResultUploaderStreamingRejectsCorruptB64JSON(t *testing.T) {
	storage := &fakeStreamingImageStorage{}
	uploader := NewImageResultUploader(storage, "images/", 0, nil)
	payload := strings.Repeat("A", 684) + "%%%%"
	result := json.RawMessage(`{"data":[{"b64_json":"` + payload + `"}]}`)

	_, err := uploader.Rewrite(context.Background(), "imgtask_corrupt_stream", result)
	require.ErrorContains(t, err, "illegal base64")
	require.Empty(t, storage.saved)
}

func TestImageResultUploaderRejectsTrailingJSON(t *testing.T) {
	uploader := NewImageResultUploader(&fakeImageStorage{}, "images/", 0, nil)
	result := json.RawMessage(`{"data":[]} {"unexpected":true}`)

	_, err := uploader.Rewrite(context.Background(), "imgtask_trailing", result)
	require.ErrorContains(t, err, "trailing JSON content")
}

func TestImageResultUploaderRewritesURL(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(pngBytes)
	}))
	defer upstream.Close()

	storage := &fakeImageStorage{}
	uploader := NewImageResultUploader(storage, "images/", 0, nil)

	result := json.RawMessage(`{"created":1,"data":[{"url":"` + upstream.URL + `/pic.png"}]}`)
	out, err := uploader.Rewrite(context.Background(), "imgtask_xyz", result)
	require.NoError(t, err)

	require.Len(t, storage.saved, 1)
	require.Equal(t, pngBytes, storage.saved[0].data)
	require.Equal(t, "image/png", storage.saved[0].contentType)

	var parsed struct {
		Data []map[string]json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(out, &parsed))
	require.JSONEq(t, `"https://cdn.test/images/imgtask_xyz-0.png"`, string(parsed.Data[0]["url"]))
}

func TestImageResultUploaderRewritesImageDataURLWithoutHTTP(t *testing.T) {
	httpCalls := 0
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		httpCalls++
		return nil, errors.New("HTTP must not be called for data URLs")
	})}
	storage := &fakeImageStorage{}
	uploader := NewImageResultUploader(storage, "images/", 0, client)
	b64 := base64.StdEncoding.EncodeToString(pngBytes)
	result := json.RawMessage(`{"data":[{"url":"DATA:image/jpeg;name=photo.jpg;BaSe64,` + b64 + `","revised_prompt":"kept"}]}`)

	out, err := uploader.Rewrite(context.Background(), "imgtask_data", result)
	require.NoError(t, err)
	require.Zero(t, httpCalls)
	require.Len(t, storage.saved, 1)
	require.Equal(t, pngBytes, storage.saved[0].data)
	require.Equal(t, "image/png", storage.saved[0].contentType, "detected bytes take precedence over a conflicting declaration")
	require.Equal(t, "images/imgtask_data-0.png", storage.saved[0].key)

	var parsed struct {
		Data []map[string]json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(out, &parsed))
	require.JSONEq(t, `"https://cdn.test/images/imgtask_data-0.png"`, string(parsed.Data[0]["url"]))
	require.JSONEq(t, `"kept"`, string(parsed.Data[0]["revised_prompt"]))
}

func TestImageResultUploaderDataURLValidation(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr string
	}{
		{name: "missing comma", url: "data:image/png;base64", wantErr: "missing comma"},
		{name: "non image", url: "data:text/plain;base64,aGVsbG8=", wantErr: "is not an image"},
		{name: "non base64", url: "data:image/png,raw", wantErr: "not base64"},
		{name: "invalid base64", url: "data:image/png;base64,%%%", wantErr: "base64 payload"},
		{name: "invalid media type", url: "data:image/png;bad parameter;base64,aGVsbG8=", wantErr: "invalid media type"},
		{name: "parameter after base64", url: "data:image/png;base64;name=photo.png,aGVsbG8=", wantErr: "base64 marker must be the final header token"},
		{name: "duplicate base64 marker", url: "data:image/png;base64;base64,aGVsbG8=", wantErr: "duplicate base64 marker"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpCalls := 0
			client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				httpCalls++
				return nil, errors.New("HTTP must not be called for data URLs")
			})}
			uploader := NewImageResultUploader(&fakeImageStorage{}, "images/", 0, client)
			result, err := json.Marshal(map[string]any{"data": []map[string]string{{"url": tt.url}}})
			require.NoError(t, err)

			_, err = uploader.Rewrite(context.Background(), "imgtask_bad", result)
			require.ErrorContains(t, err, tt.wantErr)
			require.Zero(t, httpCalls)
		})
	}
}

func TestImageResultUploaderRejectsOversizedImageDataURL(t *testing.T) {
	storage := &fakeImageStorage{}
	uploader := NewImageResultUploader(storage, "images/", 3, nil)
	payload := base64.StdEncoding.EncodeToString([]byte("four"))
	result := json.RawMessage(`{"data":[{"url":"data:image/png;base64,` + payload + `"}]}`)

	_, err := uploader.Rewrite(context.Background(), "imgtask_large", result)
	require.ErrorContains(t, err, "decoded image data URL exceeds 3 bytes")
	require.Empty(t, storage.saved)
}

func TestImageResultUploaderRejectsOversizedB64JSONBeforeStorage(t *testing.T) {
	storage := &fakeImageStorage{}
	uploader := NewImageResultUploader(storage, "images/", 3, nil)
	payload := base64.StdEncoding.EncodeToString([]byte("four"))
	result := json.RawMessage(`{"data":[{"b64_json":"` + payload + `"}]}`)

	_, err := uploader.Rewrite(context.Background(), "imgtask_large_b64", result)
	require.ErrorContains(t, err, "decoded b64_json exceeds 3 bytes")
	require.Empty(t, storage.saved)
}

func TestImageResultUploaderB64JSONTakesPrecedenceOverDataURL(t *testing.T) {
	storage := &fakeImageStorage{}
	uploader := NewImageResultUploader(storage, "images/", 0, nil)
	b64 := base64.StdEncoding.EncodeToString(pngBytes)
	result := json.RawMessage(`{"data":[{"b64_json":"` + b64 + `","url":"data:text/plain,not-an-image"}]}`)

	_, err := uploader.Rewrite(context.Background(), "imgtask_precedence", result)
	require.NoError(t, err)
	require.Len(t, storage.saved, 1)
	require.Equal(t, pngBytes, storage.saved[0].data)
}

func TestImageResultUploaderPropagatesStorageError(t *testing.T) {
	storage := &fakeImageStorage{err: errors.New("bucket unreachable")}
	uploader := NewImageResultUploader(storage, "images/", 0, nil)

	b64 := base64.StdEncoding.EncodeToString(pngBytes)
	result := json.RawMessage(`{"data":[{"b64_json":"` + b64 + `"}]}`)

	_, err := uploader.Rewrite(context.Background(), "imgtask_err", result)
	require.Error(t, err)
	require.Contains(t, err.Error(), "bucket unreachable")
}

func TestImageResultUploaderNilStoragePassthrough(t *testing.T) {
	var uploader *ImageResultUploader
	result := json.RawMessage(`{"data":[{"url":"https://example.test/x.png"}]}`)
	out, err := uploader.Rewrite(context.Background(), "imgtask_nil", result)
	require.NoError(t, err)
	require.JSONEq(t, string(result), string(out))
}

func TestImageTaskServiceCompleteOffloadsToStorage(t *testing.T) {
	store := &imageTaskMemoryStore{}
	storage := &fakeImageStorage{}
	uploader := NewImageResultUploader(storage, "images/", 0, nil)
	svc := NewImageTaskServiceWithUploader(store, uploader, time.Hour, time.Minute)
	svc.SetImageObjectRepository(&imageObjectMemoryRepository{})
	require.True(t, svc.Enabled())

	owner := ImageTaskOwner{UserID: 1, APIKeyID: 2}
	created, err := svc.Create(context.Background(), owner)
	require.NoError(t, err)

	b64 := base64.StdEncoding.EncodeToString(pngBytes)
	result := json.RawMessage(`{"created":1,"data":[{"b64_json":"` + b64 + `"}]}`)
	require.NoError(t, svc.Complete(context.Background(), created.ID, http.StatusOK, result))

	got, err := svc.Get(context.Background(), owner, created.ID)
	require.NoError(t, err)
	require.Equal(t, ImageTaskStatusCompleted, got.Status)
	require.Equal(t, "https://cdn.test/images/"+created.ID+"-0.png", got.ImageURL)
	require.NotContains(t, string(got.Result), "b64_json", "large base64 must not be persisted to Redis")
	require.Len(t, storage.saved, 1)
}

type imageObjectMemoryRepository struct {
	objects map[string]ImageObjectRecord
	err     error
}

func (r *imageObjectMemoryRepository) CreateMany(_ context.Context, objects []ImageObjectRecord) error {
	if r.err != nil {
		return r.err
	}
	if r.objects == nil {
		r.objects = make(map[string]ImageObjectRecord)
	}
	for _, object := range objects {
		r.objects[object.ObjectID] = object
	}
	return nil
}

func (r *imageObjectMemoryRepository) GetOwned(_ context.Context, objectID string, userID int64) (*ImageObjectRecord, error) {
	if r.err != nil {
		return nil, r.err
	}
	object, ok := r.objects[objectID]
	if !ok || object.UserID != userID {
		return nil, ErrImageObjectNotFound
	}
	return &object, nil
}

func TestImageTaskServicePersistsOwnershipAndRefreshesURLAcrossAPIKeys(t *testing.T) {
	store := &imageTaskMemoryStore{}
	storage := &fakeImageStorage{expiresAt: 1893456000}
	uploader := NewImageResultUploader(storage, "images/", 0, nil)
	objects := &imageObjectMemoryRepository{}
	svc := NewImageTaskServiceWithUploader(store, uploader, time.Hour, time.Minute)
	svc.SetImageObjectRepository(objects)

	owner := ImageTaskOwner{UserID: 7, APIKeyID: 9}
	created, err := svc.Create(context.Background(), owner)
	require.NoError(t, err)
	b64 := base64.StdEncoding.EncodeToString(pngBytes)
	require.NoError(t, svc.Complete(context.Background(), created.ID, http.StatusOK, json.RawMessage(`{"data":[{"b64_json":"`+b64+`"}]}`)))

	completed, err := svc.Get(context.Background(), owner, created.ID)
	require.NoError(t, err)
	var payload struct {
		Data []struct {
			ObjectID     string `json:"object_id"`
			StorageKey   string `json:"storage_key"`
			URLExpiresAt int64  `json:"url_expires_at"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(completed.Result, &payload))
	require.Len(t, payload.Data, 1)
	require.NotEmpty(t, payload.Data[0].ObjectID)
	require.Equal(t, "images/"+created.ID+"-0.png", payload.Data[0].StorageKey)
	require.Equal(t, int64(1893456000), payload.Data[0].URLExpiresAt)
	record := objects.objects[payload.Data[0].ObjectID]
	require.Equal(t, int64(7), record.UserID)
	require.Equal(t, int64(9), record.APIKeyID)

	refreshed, err := svc.RefreshObjectURL(context.Background(), 7, payload.Data[0].ObjectID)
	require.NoError(t, err)
	require.Equal(t, payload.Data[0].StorageKey, refreshed.StorageKey)
	require.Equal(t, int64(1893456000), refreshed.URLExpiresAt)
	_, err = svc.RefreshObjectURL(context.Background(), 8, payload.Data[0].ObjectID)
	require.ErrorIs(t, err, ErrImageObjectNotFound)
}

func TestImageTaskServiceObjectRecordFailureDoesNotCompleteTask(t *testing.T) {
	store := &imageTaskMemoryStore{}
	uploader := NewImageResultUploader(&fakeImageStorage{}, "images/", 0, nil)
	svc := NewImageTaskServiceWithUploader(store, uploader, time.Hour, time.Minute)
	svc.SetImageObjectRepository(&imageObjectMemoryRepository{err: errors.New("database unavailable")})
	owner := ImageTaskOwner{UserID: 1, APIKeyID: 2}
	created, err := svc.Create(context.Background(), owner)
	require.NoError(t, err)
	b64 := base64.StdEncoding.EncodeToString(pngBytes)
	require.NoError(t, svc.Complete(context.Background(), created.ID, http.StatusOK, json.RawMessage(`{"data":[{"b64_json":"`+b64+`"}]}`)))
	got, err := svc.Get(context.Background(), owner, created.ID)
	require.NoError(t, err)
	require.Equal(t, ImageTaskStatusFailed, got.Status)
	require.Contains(t, string(got.Error), "persist generated image reference")
}

func TestImageTaskServiceCompleteOffloadFailureMarksFailed(t *testing.T) {
	store := &imageTaskMemoryStore{}
	storage := &fakeImageStorage{err: errors.New("bucket unreachable")}
	uploader := NewImageResultUploader(storage, "images/", 0, nil)
	svc := NewImageTaskServiceWithUploader(store, uploader, time.Hour, time.Minute)

	owner := ImageTaskOwner{UserID: 1, APIKeyID: 2}
	created, err := svc.Create(context.Background(), owner)
	require.NoError(t, err)

	b64 := base64.StdEncoding.EncodeToString(pngBytes)
	result := json.RawMessage(`{"data":[{"b64_json":"` + b64 + `"}]}`)
	require.NoError(t, svc.Complete(context.Background(), created.ID, http.StatusOK, result))

	got, err := svc.Get(context.Background(), owner, created.ID)
	require.NoError(t, err)
	require.Equal(t, ImageTaskStatusFailed, got.Status)
	require.Equal(t, http.StatusBadGateway, got.HTTPStatus)
	require.Contains(t, string(got.Error), "object storage")
	require.NotContains(t, string(got.Result), "b64_json", "failed offload must not persist base64 to Redis")
}
