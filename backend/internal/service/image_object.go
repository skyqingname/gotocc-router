package service

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
)

var (
	ErrImageObjectNotFound    = infraerrors.New(http.StatusNotFound, "IMAGE_OBJECT_NOT_FOUND", "image object not found")
	ErrImageObjectUnavailable = infraerrors.New(http.StatusServiceUnavailable, "IMAGE_OBJECT_UNAVAILABLE", "image object storage is unavailable")
)

type ImageObjectRecord struct {
	ObjectID    string
	UserID      int64
	APIKeyID    int64
	TaskID      string
	StorageKey  string
	ContentType string
	Bytes       int64
	CreatedAt   time.Time
}

type ImageObjectRepository interface {
	CreateMany(ctx context.Context, objects []ImageObjectRecord) error
	GetOwned(ctx context.Context, objectID string, userID int64) (*ImageObjectRecord, error)
}

type ImageObjectURL struct {
	ObjectID     string `json:"id"`
	Object       string `json:"object"`
	StorageKey   string `json:"storage_key"`
	URL          string `json:"url"`
	URLExpiresAt int64  `json:"url_expires_at,omitempty"`
}

func (s *ImageTaskService) SetImageObjectRepository(repo ImageObjectRepository) {
	if s != nil {
		s.objectRepo = repo
	}
}

func (s *ImageTaskService) RefreshObjectURL(ctx context.Context, userID int64, objectID string) (*ImageObjectURL, error) {
	if s == nil || s.objectRepo == nil || userID <= 0 {
		return nil, ErrImageObjectUnavailable
	}
	objectID = strings.TrimSpace(objectID)
	if objectID == "" {
		return nil, ErrImageObjectNotFound
	}
	object, err := s.objectRepo.GetOwned(ctx, objectID, userID)
	if err != nil {
		if errors.Is(err, ErrImageObjectNotFound) {
			return nil, ErrImageObjectNotFound
		}
		return nil, ErrImageObjectUnavailable.WithCause(err)
	}
	uploader, _ := s.current()
	if uploader == nil {
		return nil, ErrImageObjectUnavailable
	}
	url, expiresAt, err := uploader.SignURL(ctx, object.StorageKey)
	if err != nil {
		return nil, ErrImageObjectUnavailable.WithCause(err)
	}
	return &ImageObjectURL{
		ObjectID:     object.ObjectID,
		Object:       "image.object",
		StorageKey:   object.StorageKey,
		URL:          url,
		URLExpiresAt: expiresAt,
	}, nil
}
