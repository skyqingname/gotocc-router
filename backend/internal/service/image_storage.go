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
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const defaultImageMaxDownloadBytes int64 = 32 << 20 // 32 MiB

// ImageStorage 把图片字节写入对象存储并返回可访问 URL。
//
// 这是对象存储的可插拔抽象：适配一个新的对象存储厂商，只需实现本接口
// （例如包一个厂商 SDK），无需改动任务/网关逻辑。仓库内自带一个 S3 兼容实现
// （repository.S3ImageStorage），适用于 AWS S3 / Cloudflare R2 / 阿里云 OSS / MinIO 等。
type ImageStorage interface {
	// Save 把 data 以 key 存入对象存储，返回可下载的 URL（公开直链或 presigned 临时链接）。
	// contentType 为图片 MIME 类型，如 "image/png"。
	Save(ctx context.Context, key, contentType string, data []byte) (url string, err error)
}

// ImageStorageReader is an optional companion to ImageStorage. It is used by
// the async-task ZIP endpoint to open only deterministic objects previously
// written for that task. Keeping it optional preserves compatibility with
// third-party storage adapters that only support result uploads.
type ImageStorageReader interface {
	Open(ctx context.Context, key string) (*ImageStorageObject, error)
}

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

type ImageStorageObject struct {
	Body        io.ReadCloser
	ContentType string
	Size        int64
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

// ImageStorageHealthChecker is implemented by storage adapters that can verify
// the configured bucket with real object operations before async tasks are enabled.
// It intentionally remains optional so third-party ImageStorage adapters do not
// need to change their Save contract.
type ImageStorageHealthChecker interface {
	Check(ctx context.Context) error
}

// ImageResultUploader 是 ImageStorage 的上层编排器（与具体厂商无关）：
// 把上游生图响应里的每张图片（b64_json 解码 / url 下载）转存到对象存储，
// 并把响应结果改写为只含短链接的紧凑 JSON，从而避免大 base64 落 Redis。
type ImageResultUploader struct {
	storage          ImageStorage
	httpClient       *http.Client
	prefix           string
	maxDownloadBytes int64
}

// NewImageResultUploader 构造一个 uploader；storage 为 nil 时 Rewrite 直接透传。
func NewImageResultUploader(storage ImageStorage, prefix string, maxDownloadBytes int64, httpClient *http.Client) *ImageResultUploader {
	if httpClient == nil {
		httpClient = defaultImageDownloadHTTPClient()
	}
	if maxDownloadBytes <= 0 {
		maxDownloadBytes = defaultImageMaxDownloadBytes
	}
	return &ImageResultUploader{
		storage:          storage,
		httpClient:       httpClient,
		prefix:           prefix,
		maxDownloadBytes: maxDownloadBytes,
	}
}

func defaultImageDownloadHTTPClient() *http.Client {
	return &http.Client{Timeout: 60 * time.Second}
}

// Rewrite 将 result（上游生图响应 JSON）里的每张图片转存到对象存储，
// 返回改写后的紧凑结果（data[i].url 指向对象存储，b64_json 被移除）。
// 任一图片转存失败即返回 error（调用方据此将任务标记为失败，绝不把大 blob 落 Redis）。
func (u *ImageResultUploader) Rewrite(ctx context.Context, taskID string, result json.RawMessage) (json.RawMessage, error) {
	rewritten, _, err := u.RewriteWithObjects(ctx, taskID, result)
	return rewritten, err
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

	objects := make([]StoredImageObject, 0, len(items))
	for i, rawItem := range items {
		item, ok := rawItem.(map[string]any)
		if !ok {
			return nil, nil, fmt.Errorf("parse image response data: image %d is not an object", i)
		}
		key, contentType, imageBytes, objectURL, err := u.saveImageItem(ctx, taskID, i, item)
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
		item["storage_key"] = object.StorageKey
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

func (u *ImageResultUploader) saveImageItem(ctx context.Context, taskID string, index int, item map[string]any) (key string, contentType string, imageBytes int64, objectURL string, err error) {
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
					key = u.buildKey(taskID, index, contentType)
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
	key = u.buildKey(taskID, index, contentType)
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

func (u *ImageResultUploader) fetchImageBytes(ctx context.Context, item map[string]any) ([]byte, string, error) {
	if raw, ok := item["b64_json"]; ok {
		if b64, ok := raw.(string); ok {
			if b64 = strings.TrimSpace(b64); b64 != "" {
				return u.decodeB64Image(b64)
			}
		}
	}
	if raw, ok := item["url"]; ok {
		if rawURL, ok := raw.(string); ok {
			if rawURL = strings.TrimSpace(rawURL); rawURL != "" {
				if len(rawURL) >= len("data:") && strings.EqualFold(rawURL[:len("data:")], "data:") {
					return u.decodeImageDataURL(rawURL)
				}
				return u.download(ctx, rawURL)
			}
		}
	}
	return nil, "", errors.New("image item has neither b64_json nor url")
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

func (u *ImageResultUploader) decodeImageDataURL(rawURL string) ([]byte, string, error) {
	header, payload, ok := strings.Cut(rawURL[len("data:"):], ",")
	if !ok {
		return nil, "", errors.New("decode image data URL: missing comma separator")
	}

	parts := strings.Split(header, ";")
	if strings.TrimSpace(parts[0]) == "" {
		return nil, "", errors.New("decode image data URL: missing media type")
	}
	base64Index := len(parts) - 1
	if base64Index < 1 || !strings.EqualFold(strings.TrimSpace(parts[base64Index]), "base64") {
		for i := 1; i < base64Index; i++ {
			if strings.EqualFold(strings.TrimSpace(parts[i]), "base64") {
				return nil, "", errors.New("decode image data URL: base64 marker must be the final header token")
			}
		}
		return nil, "", errors.New("decode image data URL: payload is not base64 encoded")
	}
	for i := 1; i < base64Index; i++ {
		if strings.EqualFold(strings.TrimSpace(parts[i]), "base64") {
			return nil, "", errors.New("decode image data URL: duplicate base64 marker")
		}
	}
	mediaTypeHeader := strings.Join(parts[:base64Index], ";")
	declaredType, _, err := mime.ParseMediaType(mediaTypeHeader)
	if err != nil {
		return nil, "", fmt.Errorf("decode image data URL: invalid media type: %w", err)
	}
	declaredType = strings.ToLower(declaredType)
	if !strings.HasPrefix(declaredType, "image/") {
		return nil, "", fmt.Errorf("decode image data URL: media type %q is not an image", declaredType)
	}

	limit := u.maxDownloadBytes
	if limit <= 0 {
		limit = defaultImageMaxDownloadBytes
	}
	decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(payload))
	data, err := io.ReadAll(io.LimitReader(decoder, limit+1))
	if err != nil {
		return nil, "", fmt.Errorf("decode image data URL base64 payload: %w", err)
	}
	if int64(len(data)) > limit {
		return nil, "", fmt.Errorf("decoded image data URL exceeds %d bytes", limit)
	}

	contentType := detectedImageContentType(data)
	if contentType == "" {
		contentType = declaredType
	}
	return data, contentType, nil
}

func (u *ImageResultUploader) download(ctx context.Context, rawURL string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("build download request: %w", err)
	}
	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("download image: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, "", fmt.Errorf("download image: unexpected status %d", resp.StatusCode)
	}
	limit := u.maxDownloadBytes
	if limit <= 0 {
		limit = defaultImageMaxDownloadBytes
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, "", fmt.Errorf("read image body: %w", err)
	}
	if int64(len(data)) > limit {
		return nil, "", fmt.Errorf("downloaded image exceeds %d bytes", limit)
	}
	contentType := strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0])
	if !strings.HasPrefix(contentType, "image/") {
		contentType = detectImageContentType(data)
	}
	return data, contentType, nil
}

func (u *ImageResultUploader) buildKey(taskID string, index int, contentType string) string {
	return u.prefix + taskID + "-" + strconv.Itoa(index) + extensionForContentType(contentType)
}

// OpenStoredTaskImage opens an exact object written by Rewrite. The extension
// is derived only from the task's stored image URL and must be one of the image
// formats this uploader can create; neither host nor path from that URL is ever
// fetched or otherwise trusted.
func (u *ImageResultUploader) OpenStoredTaskImage(ctx context.Context, taskID string, index int, storedURL string) (*ImageStorageObject, string, error) {
	if u == nil || u.storage == nil {
		return nil, "", errors.New("image storage is unavailable")
	}
	reader, ok := u.storage.(ImageStorageReader)
	if !ok {
		return nil, "", errors.New("image storage does not support reading stored images")
	}
	extension := imageExtensionFromStoredURL(storedURL)
	if extension == "" {
		return nil, "", errors.New("stored image URL has an unsupported extension")
	}
	object, err := reader.Open(ctx, u.prefix+taskID+"-"+strconv.Itoa(index)+extension)
	if err != nil {
		return nil, "", err
	}
	if object == nil || object.Body == nil {
		return nil, "", errors.New("stored image object is empty")
	}
	if strings.TrimSpace(object.ContentType) == "" {
		object.ContentType = imageContentTypeFromExtension(extension)
	}
	return object, extension, nil
}

func imageExtensionFromStoredURL(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return ""
	}
	path := strings.ToLower(parsed.Path)
	for _, extension := range []string{".png", ".jpg", ".jpeg", ".webp", ".gif"} {
		if strings.HasSuffix(path, extension) {
			if extension == ".jpeg" {
				return ".jpg"
			}
			return extension
		}
	}
	return ""
}

func imageContentTypeFromExtension(extension string) string {
	switch strings.ToLower(extension) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	default:
		return "image/png"
	}
}

func detectImageContentType(data []byte) string {
	if ct := detectedImageContentType(data); ct != "" {
		return ct
	}
	return "image/png"
}

func detectedImageContentType(data []byte) string {
	ct := strings.TrimSpace(strings.Split(http.DetectContentType(data), ";")[0])
	if strings.HasPrefix(ct, "image/") {
		return ct
	}
	return ""
}

func extensionForContentType(ct string) string {
	switch {
	case strings.Contains(ct, "png"):
		return ".png"
	case strings.Contains(ct, "jpeg"), strings.Contains(ct, "jpg"):
		return ".jpg"
	case strings.Contains(ct, "webp"):
		return ".webp"
	case strings.Contains(ct, "gif"):
		return ".gif"
	default:
		return ".png"
	}
}
