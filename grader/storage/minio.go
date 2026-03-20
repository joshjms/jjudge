package storage

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/jjudge-oj/grader/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOClient wraps the MinIO SDK client and bucket name.
type MinIOClient struct {
	client *minio.Client
	bucket string
}

// NewMinIOClient constructs a MinIO client from config.
func NewMinIOClient(cfg *config.MinIOConfig) (*MinIOClient, error) {
	if strings.TrimSpace(cfg.Endpoint) == "" {
		return nil, errors.New("minio endpoint is required")
	}
	if strings.TrimSpace(cfg.AccessKey) == "" || strings.TrimSpace(cfg.SecretKey) == "" {
		return nil, errors.New("minio access key and secret key are required")
	}
	if strings.TrimSpace(cfg.Bucket) == "" {
		return nil, errors.New("minio bucket is required")
	}

	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, err
	}

	return &MinIOClient{
		client: client,
		bucket: cfg.Bucket,
	}, nil
}

// EnsureBucket ensures the configured bucket exists.
func (m *MinIOClient) EnsureBucket(ctx context.Context) error {
	exists, err := m.client.BucketExists(ctx, m.bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return m.client.MakeBucket(ctx, m.bucket, minio.MakeBucketOptions{})
}

// Put uploads an object to the configured bucket.
func (m *MinIOClient) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	_, err := m.client.PutObject(ctx, m.bucket, key, r, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

// Get opens a reader for an object in the configured bucket.
func (m *MinIOClient) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	return m.client.GetObject(ctx, m.bucket, key, minio.GetObjectOptions{})
}

// Delete removes an object from the configured bucket.
func (m *MinIOClient) Delete(ctx context.Context, key string) error {
	return m.client.RemoveObject(ctx, m.bucket, key, minio.RemoveObjectOptions{})
}

// Client exposes the underlying MinIO SDK client.
func (m *MinIOClient) Client() *minio.Client {
	return m.client
}

// Bucket returns the configured bucket name.
func (m *MinIOClient) Bucket() string {
	return m.bucket
}
