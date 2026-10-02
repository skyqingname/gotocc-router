package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	appTimezone "github.com/LuckyKuang/sub2api-plus/internal/pkg/timezone"
	"github.com/google/uuid"
)

// ImageStorageStreamWriter lets object stores consume a decoded image stream
// with an exact content length. The S3/R2 implementation uses this path so a
// large b64_json result does not require another full decoded byte slice.
type ImageStorageStreamWriter interface {
	SaveReader(ctx context.Context, key, contentType string, body io.Reader, size int64) (url string, err error)
}

// ImageStorageURLSigner mints a fresh URL for an existing object. Credentials
// remain inside the storage implementation; callers provide only an owned key.
type ImageStorageURLSigner interface {
	SignURL(ctx context.Context, key string) (url string, expiresAt int64, err error)
}

type StoredImageObject struct {
	ObjectID     string
	TaskID       string
	StorageKey   string
	ContentType  string
	Bytes        int64
	URL          string
	URLExpiresAt int64
}

// RewriteWithObjects returns both the compact task response and the durable
// ownership metadata that must be persisted before the task is completed.
func (u *ImageResultUploader) RewriteWithObjects(ctx context.Context, taskID string, result json.RawMessage) (json.RawMessage, []StoredImageObject, error) {
	if u == nil || u.storage == nil {
		return result, nil, nil
	}

	// Decode once so a large b64_json string is not retained simultaneously in
	// top-level RawMessage values and per-item RawMessage maps.
	var top map[string]any
	decoder := json.NewDecoder(bytes.NewReader(result))
	decoder.UseNumber()
	if err := decoder.Decode(&top); err != nil {
		return nil, nil, fmt.Errorf("parse image response: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, nil, errors.New("parse image response: trailing JSON content")
	}
	rawData, ok := top["data"]
	if !ok {
		// 没有 data 数组（结构不符合预期），保持原样返回，交由上层决定。
		return result, nil, nil
	}
	items, ok := rawData.([]any)
	if !ok {
		return nil, nil, errors.New("parse image response data: data is not an array")
	}
	if len(items) == 0 {
		return result, nil, nil
	}
	createdAt := appTimezone.Now()
	objects := make([]StoredImageObject, 0, len(items))
	for i, rawItem := range items {
		item, ok := rawItem.(map[string]any)
		if !ok {
			return nil, nil, fmt.Errorf("parse image response data: image %d is not an object", i)
		}
		key, contentType, imageBytes, objectURL, err := u.saveImageItem(ctx, taskID, i, item, createdAt)
		if err != nil {
			return nil, nil, fmt.Errorf("image %d: %w", i, err)
		}
		urlExpiresAt := int64(0)
		if signer, ok := u.storage.(ImageStorageURLSigner); ok {
			objectURL, urlExpiresAt, err = signer.SignURL(ctx, key)
			if err != nil {
				return nil, nil, fmt.Errorf("image %d: sign object URL: %w", i, err)
			}
		}
		object := StoredImageObject{
			ObjectID:     "imgobj_" + strings.ReplaceAll(uuid.NewString(), "-", ""),
			TaskID:       taskID,
			StorageKey:   key,
			ContentType:  contentType,
			Bytes:        imageBytes,
			URL:          objectURL,
			URLExpiresAt: urlExpiresAt,
		}
		item["url"] = object.URL
		item["object_id"] = object.ObjectID
		item["content_type"] = object.ContentType
		item["bytes"] = object.Bytes
		if object.URLExpiresAt > 0 {
			item["url_expires_at"] = object.URLExpiresAt
		}
		delete(item, "b64_json")
		items[i] = item
		objects = append(objects, object)
	}
	top["data"] = items
	out, err := json.Marshal(top)
	if err != nil {
		return nil, nil, fmt.Errorf("encode image response: %w", err)
	}
	return out, objects, nil
}

func (u *ImageResultUploader) SignURL(ctx context.Context, storageKey string) (string, int64, error) {
	if u == nil || u.storage == nil {
		return "", 0, errors.New("image object storage is unavailable")
	}
	signer, ok := u.storage.(ImageStorageURLSigner)
	if !ok {
		return "", 0, errors.New("image object storage does not support URL signing")
	}
	storageKey = strings.TrimSpace(storageKey)
	if storageKey == "" {
		return "", 0, errors.New("image object storage key is empty")
	}
	return signer.SignURL(ctx, storageKey)
}

func (u *ImageResultUploader) saveImageItem(ctx context.Context, taskID string, index int, item map[string]any, createdAt time.Time) (key string, contentType string, imageBytes int64, objectURL string, err error) {
	if storage, ok := u.storage.(ImageStorageStreamWriter); ok {
		if raw, exists := item["b64_json"]; exists {
			if payload, ok := raw.(string); ok {
				if payload = strings.TrimSpace(payload); payload != "" {
					expectedBytes, sizeErr := u.b64ImageDecodedSize(payload)
					if sizeErr != nil {
						return "", "", 0, "", sizeErr
					}
					decoded := base64.NewDecoder(base64.StdEncoding, strings.NewReader(payload))
					buffered := bufio.NewReaderSize(decoded, 512)
					header, peekErr := buffered.Peek(512)
					if peekErr != nil && !errors.Is(peekErr, io.EOF) {
						return "", "", 0, "", fmt.Errorf("decode b64_json: %w", peekErr)
					}
					if len(header) == 0 {
						return "", "", 0, "", errors.New("decode b64_json: empty image")
					}
					contentType = detectImageContentType(header)
					key = u.buildKeyAt(taskID, index, contentType, createdAt)
					counter := &countingReader{reader: buffered}
					objectURL, err = storage.SaveReader(ctx, key, contentType, counter, expectedBytes)
					if err != nil {
						return "", "", 0, "", fmt.Errorf("upload stream to object storage: %w", err)
					}
					if counter.bytesRead != expectedBytes {
						return "", "", 0, "", fmt.Errorf("upload stream consumed %d bytes; expected %d", counter.bytesRead, expectedBytes)
					}
					return key, contentType, expectedBytes, objectURL, nil
				}
			}
		}
	}

	data, contentType, err := u.fetchImageBytes(ctx, item)
	if err != nil {
		return "", "", 0, "", err
	}
	key = u.buildKeyAt(taskID, index, contentType, createdAt)
	objectURL, err = u.storage.Save(ctx, key, contentType, data)
	if err != nil {
		return "", "", 0, "", fmt.Errorf("upload to object storage: %w", err)
	}
	return key, contentType, int64(len(data)), objectURL, nil
}

type countingReader struct {
	reader    io.Reader
	bytesRead int64
}

func (r *countingReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.bytesRead += int64(n)
	return n, err
}

func (u *ImageResultUploader) decodeB64Image(payload string) ([]byte, string, error) {
	if _, err := u.b64ImageDecodedSize(payload); err != nil {
		return nil, "", err
	}
	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, "", fmt.Errorf("decode b64_json: %w", err)
	}
	return data, detectImageContentType(data), nil
}

func (u *ImageResultUploader) b64ImageDecodedSize(payload string) (int64, error) {
	limit := u.maxDownloadBytes
	if limit <= 0 {
		limit = defaultImageMaxDownloadBytes
	}
	if limit <= int64(^uint(0)>>1) && len(payload) > base64.StdEncoding.EncodedLen(int(limit)) {
		return 0, fmt.Errorf("decoded b64_json exceeds %d bytes", limit)
	}
	if len(payload)%4 != 0 {
		return 0, errors.New("decode b64_json: invalid base64 length")
	}
	padding := int64(0)
	if strings.HasSuffix(payload, "=") {
		padding++
	}
	if strings.HasSuffix(payload, "==") {
		padding++
	}
	decodedBytes := int64(len(payload)/4*3) - padding
	if decodedBytes > limit {
		return 0, fmt.Errorf("decoded b64_json exceeds %d bytes", limit)
	}
	return decodedBytes, nil
}
