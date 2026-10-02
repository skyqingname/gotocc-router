package repository

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/servertiming"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// SaveReader uploads a stream with an exact length. S3-compatible stores use
// Content-Length for bounded uploads and never need a second decoded buffer.
func (s *S3ImageStorage) SaveReader(ctx context.Context, key, contentType string, body io.Reader, size int64) (string, error) {
	if s == nil || s.client == nil || strings.TrimSpace(s.bucket) == "" {
		return "", fmt.Errorf("S3 image storage is not configured")
	}
	if body == nil || size < 0 {
		return "", fmt.Errorf("invalid S3 image upload")
	}
	finish := servertiming.ObserveDependency(ctx, "s3")
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        &s.bucket,
		Key:           &key,
		Body:          body,
		ContentLength: &size,
		ContentType:   &contentType,
	})
	finish()
	if err != nil {
		return "", fmt.Errorf("S3 PutObject: %w", err)
	}
	url, _, err := s.SignURL(ctx, key)
	return url, err
}

// SignURL returns a fresh public or presigned URL for an existing object.
func (s *S3ImageStorage) SignURL(ctx context.Context, key string) (string, int64, error) {
	if s == nil || s.client == nil || strings.TrimSpace(s.bucket) == "" {
		return "", 0, fmt.Errorf("S3 image storage is not configured")
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return "", 0, fmt.Errorf("S3 image storage key is empty")
	}

	if s.publicBaseURL != "" {
		return s.publicBaseURL + "/" + strings.TrimLeft(key, "/"), 0, nil
	}

	presignClient := s3.NewPresignClient(s.client)
	result, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	}, s3.WithPresignExpires(s.presignExpiry))
	if err != nil {
		return "", 0, fmt.Errorf("presign url: %w", err)
	}
	return result.URL, time.Now().UTC().Add(s.presignExpiry).Unix(), nil
}
