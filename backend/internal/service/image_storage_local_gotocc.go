package service

import "context"

// LocalImageStorageKeyPrefix is a durable storage namespace, not a directory.
// Unmarked keys retain the original S3 meaning; no file-existence fallback is used.
const LocalImageStorageKeyPrefix = "local:"

const LocalImageURLPath = "/v1/images/local"

type ImageStorageSignedReader interface {
	OpenSigned(context.Context, string) (*ImageStorageObject, error)
}

func (s *ImageTaskService) OpenSignedLocalImage(ctx context.Context, token string) (*ImageStorageObject, error) {
	uploader, _ := s.current()
	if uploader == nil {
		return nil, ErrImageObjectUnavailable
	}
	reader, ok := uploader.storage.(ImageStorageSignedReader)
	if !ok {
		return nil, ErrImageObjectNotFound
	}
	return reader.OpenSigned(ctx, token)
}

func (s *ImageStorageSettingService) WriteBackend() string {
	if s.fallback.Routing != nil {
		return s.fallback.Routing.WriteBackend
	}
	return "s3"
}
