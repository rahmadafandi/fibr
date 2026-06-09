// Copyright 2026 Rahmad Afandi. MIT License.

package uploader

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakePutter records the last PutObject call and returns a configurable result.
type fakePutter struct {
	calls  int
	bucket string
	key    string
	opts   minio.PutObjectOptions
	err    error
}

func (f *fakePutter) PutObject(ctx context.Context, bucket, key string, r io.Reader, size int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	f.calls++
	f.bucket, f.key, f.opts = bucket, key, opts
	if f.err != nil {
		return minio.UploadInfo{}, f.err
	}
	_, _ = io.Copy(io.Discard, r)
	return minio.UploadInfo{Bucket: bucket, Key: key}, nil
}

func TestS3UploadHappyPath(t *testing.T) {
	fp := &fakePutter{}
	u := newS3Uploader(fp, "my-bucket", WithKeyPrefix("avatars/"))
	got, err := u.Upload(newFile([]byte("hello world")), "photo.png")
	require.NoError(t, err)
	assert.Equal(t, "avatars/photo.png", got)
	assert.Equal(t, 1, fp.calls)
	assert.Equal(t, "my-bucket", fp.bucket)
	assert.Equal(t, "avatars/photo.png", fp.key)
	assert.NotEmpty(t, fp.opts.ContentType)
}

func TestS3UploadBaseURL(t *testing.T) {
	fp := &fakePutter{}
	u := newS3Uploader(fp, "b", WithBaseURL("https://cdn.example.com/"))
	got, err := u.Upload(newFile([]byte("hello world")), "photo.png")
	require.NoError(t, err)
	assert.Equal(t, "https://cdn.example.com/photo.png", got)
}

func TestS3UploadPutError(t *testing.T) {
	fp := &fakePutter{err: errors.New("boom")}
	u := newS3Uploader(fp, "b")
	_, err := u.Upload(newFile([]byte("hello world")), "photo.png")
	require.Error(t, err)
}

func TestS3UploadValidationBlocksPut(t *testing.T) {
	fp := &fakePutter{}
	u := newS3Uploader(fp, "b", WithMaxSize(2))
	_, err := u.Upload(newFile([]byte("too big")), "photo.png")
	require.Error(t, err)
	assert.Equal(t, 0, fp.calls)
}

func TestS3ImplementsUploader(t *testing.T) {
	var _ Uploader = (*S3Uploader)(nil)
}
