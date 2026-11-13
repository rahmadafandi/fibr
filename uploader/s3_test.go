// Copyright 2026 Rahmad Afandi. MIT License.

package uploader

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ctx is the default context for uploads in these tests.
var ctx = context.Background()

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
	u := newS3Uploader(fp, "my-bucket", WithKeyPrefix("avatars/"), WithOriginalNames())
	got, err := u.Upload(ctx, newFile([]byte("hello world")), "photo.png")
	require.NoError(t, err)
	assert.Equal(t, "avatars/photo.png", got)
	assert.Equal(t, 1, fp.calls)
	assert.Equal(t, "my-bucket", fp.bucket)
	assert.Equal(t, "avatars/photo.png", fp.key)
	assert.NotEmpty(t, fp.opts.ContentType)
}

func TestS3UploadBaseURL(t *testing.T) {
	fp := &fakePutter{}
	u := newS3Uploader(fp, "b", WithBaseURL("https://cdn.example.com/"), WithOriginalNames())
	got, err := u.Upload(ctx, newFile([]byte("hello world")), "photo.png")
	require.NoError(t, err)
	assert.Equal(t, "https://cdn.example.com/photo.png", got)
}

func TestS3UploadPutError(t *testing.T) {
	fp := &fakePutter{err: errors.New("boom")}
	u := newS3Uploader(fp, "b")
	_, err := u.Upload(ctx, newFile([]byte("hello world")), "photo.png")
	require.Error(t, err)
}

func TestS3UploadValidationBlocksPut(t *testing.T) {
	fp := &fakePutter{}
	u := newS3Uploader(fp, "b", WithMaxSize(2))
	_, err := u.Upload(ctx, newFile([]byte("too big")), "photo.png")
	require.Error(t, err)
	assert.Equal(t, 0, fp.calls)
}

func TestS3UploadDisallowedMimeBlocksPut(t *testing.T) {
	fp := &fakePutter{}
	u := newS3Uploader(fp, "b", WithAllowedMime([]string{"image/png"}))
	_, err := u.Upload(ctx, newFile([]byte("plain text, not png")), "note.txt")
	require.Error(t, err)
	assert.Equal(t, 0, fp.calls)
}

func TestS3ImplementsUploader(t *testing.T) {
	var _ Uploader = (*S3Uploader)(nil)
}

func TestUniqueNamesDoNotCollide(t *testing.T) {
	dir := t.TempDir()
	u := NewLocalUploader(dir)

	first, err := u.Upload(ctx, newFile([]byte("one")), "avatar.png")
	require.NoError(t, err)
	second, err := u.Upload(ctx, newFile([]byte("two")), "avatar.png")
	require.NoError(t, err)

	assert.NotEqual(t, first, second)
	assert.Regexp(t, `[0-9a-f]{16}-avatar\.png$`, first)
	got, err := os.ReadFile(first)
	require.NoError(t, err)
	assert.Equal(t, "one", string(got), "second upload must not overwrite the first")
}

func TestS3UniqueNamesKeepPrefix(t *testing.T) {
	fp := &fakePutter{}
	u := newS3Uploader(fp, "b", WithKeyPrefix("avatars/"))
	got, err := u.Upload(ctx, newFile([]byte("hello")), "photo.png")
	require.NoError(t, err)
	assert.Regexp(t, `^avatars/[0-9a-f]{16}-photo\.png$`, got)
}

func TestUploadContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := NewLocalUploader(t.TempDir()).Upload(ctx, newFile([]byte("x")), "a.txt")
	assert.ErrorIs(t, err, context.Canceled)

	fp := &fakePutter{}
	var up Uploader = newS3Uploader(fp, "b")
	_, err = up.Upload(context.Background(), newFile([]byte("x")), "a.txt")
	require.NoError(t, err)
	assert.Equal(t, 1, fp.calls)
}

func TestOriginalNamesOverwrite(t *testing.T) {
	dir := t.TempDir()
	u := NewLocalUploader(dir, WithOriginalNames())

	first, err := u.Upload(ctx, newFile([]byte("one")), "logo.png")
	require.NoError(t, err)
	second, err := u.Upload(ctx, newFile([]byte("two")), "logo.png")
	require.NoError(t, err)

	assert.Equal(t, first, second)
	got, err := os.ReadFile(second)
	require.NoError(t, err)
	assert.Equal(t, "two", string(got))
}
