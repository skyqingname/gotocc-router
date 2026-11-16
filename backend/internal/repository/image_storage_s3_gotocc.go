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

// SignURL returns a fresh public or presigned URL for an existing object and,
// for presigned URLs, the instant the signature expires.
func (s *S3ImageStorage) SignURL(ctx context.Context, key string) (string, int64, error) {
	url, err := s.Sign(ctx, key)
	if err != nil || s.publicBaseURL != "" {
		return url, 0, err
	}
	return url, time.Now().UTC().Add(s.presignExpiry).Unix(), nil
}
