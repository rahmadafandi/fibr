// Copyright 2026 Rahmad Afandi. MIT License.

package uploader

import (
	"context"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

// LocalUploader saves files to a local directory.
type LocalUploader struct {
	Path string
	cfg  config
}

// NewLocalUploader creates a new LocalUploader.
func NewLocalUploader(path string, opts ...Option) *LocalUploader {
	return &LocalUploader{Path: path, cfg: newConfig(opts...)}
}

// Upload writes file into the uploader's directory and returns the saved path.
// An existing file of the same name is replaced unless WithUniqueNames is set.
func (u *LocalUploader) Upload(file multipart.File, filename string) (string, error) {
	return u.UploadContext(context.Background(), file, filename)
}

// UploadContext is Upload, abandoned up front if ctx is already done.
func (u *LocalUploader) UploadContext(ctx context.Context, file multipart.File, filename string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	safe, _, err := u.cfg.validate(file, filename)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(u.Path, 0o750); err != nil {
		return "", err
	}

	dstPath := filepath.Join(u.Path, safe)
	dst, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o640)
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(dst, file); err != nil {
		_ = dst.Close()
		_ = os.Remove(dstPath)
		return "", err
	}

	if err := dst.Close(); err != nil {
		_ = os.Remove(dstPath)
		return "", err
	}

	return dstPath, nil
}
