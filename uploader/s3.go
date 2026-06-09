// Copyright 2026 Rahmad Afandi. MIT License.

package uploader

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"

	"github.com/minio/minio-go/v7"
)

// objectPutter is the minio client method S3Uploader needs. *minio.Client
// satisfies it; tests provide a fake.
type objectPutter interface {
	PutObject(ctx context.Context, bucket, key string, r io.Reader, size int64, opts minio.PutObjectOptions) (minio.UploadInfo, error)
}

// S3Uploader uploads files to an S3-compatible bucket (AWS S3, MinIO, R2, ...).
type S3Uploader struct {
	client objectPutter
	bucket string
	cfg    config
}

// NewS3Uploader creates an uploader backed by an S3-compatible bucket. Build the
// client with minio.New(endpoint, &minio.Options{Creds: ..., Secure: true}).
func NewS3Uploader(client *minio.Client, bucket string, opts ...Option) *S3Uploader {
	return newS3Uploader(client, bucket, opts...)
}

// newS3Uploader is the seam used by tests to inject a fake objectPutter.
func newS3Uploader(client objectPutter, bucket string, opts ...Option) *S3Uploader {
	return &S3Uploader{client: client, bucket: bucket, cfg: newConfig(opts...)}
}

// Upload validates the file, then puts it at <keyPrefix><sanitized-name> in the
// bucket. It returns the object key, or <baseURL>+key when WithBaseURL is set.
//
// The Uploader interface carries no context, so the put uses
// context.Background(); a per-call context would require an interface change.
func (u *S3Uploader) Upload(file multipart.File, filename string) (string, error) {
	safe, mimeType, err := u.cfg.validate(file, filename)
	if err != nil {
		return "", err
	}

	size, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		return "", err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	key := u.cfg.keyPrefix + safe
	if _, err := u.client.PutObject(context.Background(), u.bucket, key, file, size, minio.PutObjectOptions{ContentType: mimeType}); err != nil {
		return "", fmt.Errorf("uploader: put %s: %w", key, err)
	}

	if u.cfg.baseURL != "" {
		return u.cfg.baseURL + key, nil
	}
	return key, nil
}
