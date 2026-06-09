// Copyright 2026 Rahmad Afandi. MIT License.

// Package uploader stores uploaded multipart files to a local directory or an
// S3-compatible bucket behind a common Uploader interface.
package uploader

import (
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"

	"github.com/gabriel-vasile/mimetype"
)

// Uploader is an interface for uploading files.
type Uploader interface {
	Upload(file multipart.File, filename string) (string, error)
}

type config struct {
	maxSize     int64
	allowedMime []string
	baseURL     string // S3 only
	keyPrefix   string // S3 only
}

// Option configures an uploader.
type Option func(*config)

// WithMaxSize rejects uploads larger than the given number of bytes.
func WithMaxSize(bytes int64) Option { return func(c *config) { c.maxSize = bytes } }

// WithAllowedMime rejects uploads whose detected MIME type is not in the list.
func WithAllowedMime(mimes []string) Option { return func(c *config) { c.allowedMime = mimes } }

// WithBaseURL makes S3Uploader.Upload return baseURL+key instead of just the
// key. For S3Uploader only; LocalUploader ignores it.
func WithBaseURL(url string) Option { return func(c *config) { c.baseURL = url } }

// WithKeyPrefix namespaces S3 object keys (e.g. "avatars/"). For S3Uploader
// only; LocalUploader ignores it.
func WithKeyPrefix(prefix string) Option { return func(c *config) { c.keyPrefix = prefix } }

func newConfig(opts ...Option) config {
	var c config
	for _, o := range opts {
		o(&c)
	}
	return c
}

func sanitizeFilename(name string) (string, error) {
	clean := filepath.Base(filepath.Clean(name))
	if clean == "." || clean == ".." || clean == "" || clean == string(filepath.Separator) {
		return "", fmt.Errorf("invalid filename %q", name)
	}
	return clean, nil
}

func mimeAllowed(mt *mimetype.MIME, allowed []string) bool {
	for _, a := range allowed {
		if mt.Is(a) {
			return true
		}
	}
	return false
}

// validate sanitizes filename, enforces maxSize and allowedMime, and returns the
// sanitized base name and the detected MIME type. file is rewound to the start.
func (c config) validate(file multipart.File, filename string) (string, string, error) {
	safe, err := sanitizeFilename(filename)
	if err != nil {
		return "", "", err
	}

	if c.maxSize > 0 {
		size, err := file.Seek(0, io.SeekEnd)
		if err != nil {
			return "", "", err
		}
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return "", "", err
		}
		if size > c.maxSize {
			return "", "", fmt.Errorf("file size %d exceeds limit %d", size, c.maxSize)
		}
	}

	head := make([]byte, 512)
	// n is 0 on an empty file; mimetype.Detect handles a zero-length slice.
	n, _ := file.Read(head)
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", "", err
	}
	mt := mimetype.Detect(head[:n])
	if len(c.allowedMime) > 0 && !mimeAllowed(mt, c.allowedMime) {
		return "", "", fmt.Errorf("mime type %s not allowed", mt.String())
	}

	return safe, mt.String(), nil
}
