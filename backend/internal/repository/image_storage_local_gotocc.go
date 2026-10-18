package repository

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/config"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// routedImageStorage keeps both readers available independently of the write
// destination. The namespace stored in PostgreSQL and Redis chooses the reader.
type routedImageStorage struct {
	s3         *S3ImageStorage
	routing    config.ImageStorageRoutingConfig
	signingKey []byte
}

const localImageURLIssuer = "sub2api-local-image"

func newRoutedImageStorage(s3Store *S3ImageStorage, cfg *config.ImageStorageConfig) (service.ImageStorage, error) {
	if err := os.MkdirAll(cfg.Routing.LocalDirectory, 0750); err != nil {
		return nil, fmt.Errorf("create local image directory: %w", err)
	}
	return &routedImageStorage{s3: s3Store, routing: *cfg.Routing, signingKey: []byte(cfg.LocalSigningKey)}, nil
}

func (s *routedImageStorage) Save(ctx context.Context, key, contentType string, data []byte) (string, error) {
	return s.SaveReader(ctx, key, contentType, bytes.NewReader(data), int64(len(data)))
}

func (s *routedImageStorage) SaveReader(ctx context.Context, key, contentType string, body io.Reader, size int64) (string, error) {
	if !strings.HasPrefix(key, service.LocalImageStorageKeyPrefix) {
		return s.s3.SaveReader(ctx, key, contentType, body, size)
	}
	root, err := os.OpenRoot(s.routing.LocalDirectory)
	if err != nil {
		return "", err
	}
	defer root.Close()
	relative := filepath.FromSlash(strings.TrimPrefix(key, service.LocalImageStorageKeyPrefix))
	if err := root.MkdirAll(filepath.Dir(relative), 0750); err != nil {
		return "", err
	}
	temporary := relative + ".tmp"
	file, err := root.OpenFile(temporary, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0640)
	if err != nil {
		return "", err
	}
	defer root.Remove(temporary)
	written, copyErr := io.Copy(file, body)
	closeErr := file.Close()
	if copyErr != nil {
		return "", copyErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	if written != size {
		return "", fmt.Errorf("local image write: received %d bytes, expected %d", written, size)
	}
	if err := root.Rename(temporary, relative); err != nil {
		return "", err
	}
	imageURL, _, err := s.SignURL(ctx, key)
	return imageURL, err
}

func (s *routedImageStorage) Open(ctx context.Context, key string) (*service.ImageStorageObject, error) {
	if !strings.HasPrefix(key, service.LocalImageStorageKeyPrefix) {
		return s.s3.Open(ctx, key)
	}
	relative := filepath.FromSlash(strings.TrimPrefix(key, service.LocalImageStorageKeyPrefix))
	file, err := os.OpenInRoot(s.routing.LocalDirectory, relative)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	return &service.ImageStorageObject{Body: file, ContentType: mime.TypeByExtension(filepath.Ext(relative)), Size: info.Size()}, nil
}

func (s *routedImageStorage) SignURL(ctx context.Context, key string) (string, int64, error) {
	if !strings.HasPrefix(key, service.LocalImageStorageKeyPrefix) {
		return s.s3.SignURL(ctx, key)
	}
	expiresAt := time.Now().UTC().Add(time.Duration(s.routing.URLExpiryHours) * time.Hour)
	claims := jwt.RegisteredClaims{
		Issuer: localImageURLIssuer, Subject: key, ExpiresAt: jwt.NewNumericDate(expiresAt),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.signingKey)
	if err != nil {
		return "", 0, err
	}
	return s.routing.PublicBaseURL + service.LocalImageURLPath + "?token=" + url.QueryEscape(token), expiresAt.Unix(), nil
}

func (s *routedImageStorage) OpenSigned(ctx context.Context, rawToken string) (*service.ImageStorageObject, error) {
	claims := &jwt.RegisteredClaims{}
	_, err := jwt.ParseWithClaims(rawToken, claims, func(*jwt.Token) (any, error) {
		return s.signingKey, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer(localImageURLIssuer), jwt.WithExpirationRequired())
	if err != nil || !strings.HasPrefix(claims.Subject, service.LocalImageStorageKeyPrefix) {
		return nil, service.ErrImageObjectNotFound
	}
	object, err := s.Open(ctx, claims.Subject)
	if os.IsNotExist(err) {
		return nil, service.ErrImageObjectNotFound
	}
	if err != nil {
		return nil, service.ErrImageObjectUnavailable.WithCause(err)
	}
	return object, nil
}

func (s *routedImageStorage) Check(ctx context.Context) error {
	if s.routing.WriteBackend == "s3" {
		return s.s3.Check(ctx)
	}
	key := service.LocalImageStorageKeyPrefix + ".healthcheck/" + uuid.NewString() + ".bin"
	data := []byte("local image storage write/read check")
	if _, err := s.Save(ctx, key, "application/octet-stream", data); err != nil {
		return err
	}
	root, err := os.OpenRoot(s.routing.LocalDirectory)
	if err != nil {
		return err
	}
	defer root.Close()
	defer root.Remove(strings.TrimPrefix(key, service.LocalImageStorageKeyPrefix))
	object, err := s.Open(ctx, key)
	if err != nil {
		return err
	}
	defer object.Body.Close()
	stored, err := io.ReadAll(object.Body)
	if err != nil {
		return err
	}
	if !bytes.Equal(stored, data) {
		return fmt.Errorf("local image storage read did not match written bytes")
	}
	return nil
}
