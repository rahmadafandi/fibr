// Copyright 2025 Rahmad Afandi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package uploader

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/gabriel-vasile/mimetype"
)

// LocalUploader saves files to a local directory.
type LocalUploader struct {
	Path        string
	maxSize     int64
	allowedMime []string
}

// Option configures a LocalUploader.
type Option func(*LocalUploader)

// WithMaxSize rejects uploads larger than the given number of bytes.
func WithMaxSize(bytes int64) Option {
	return func(u *LocalUploader) { u.maxSize = bytes }
}

// WithAllowedMime rejects uploads whose detected MIME type is not in the list.
func WithAllowedMime(mimes []string) Option {
	return func(u *LocalUploader) { u.allowedMime = mimes }
}

// NewLocalUploader creates a new LocalUploader.
func NewLocalUploader(path string, opts ...Option) *LocalUploader {
	u := &LocalUploader{Path: path}
	for _, o := range opts {
		o(u)
	}
	return u
}

func sanitizeFilename(name string) (string, error) {
	clean := filepath.Base(filepath.Clean(name))
	if clean == "." || clean == ".." || clean == "" || clean == string(filepath.Separator) {
		return "", fmt.Errorf("invalid filename %q", name)
	}
	return clean, nil
}

// Upload writes file into the uploader's directory and returns the saved path.
func (u *LocalUploader) Upload(file multipart.File, filename string) (string, error) {
	safe, err := sanitizeFilename(filename)
	if err != nil {
		return "", err
	}

	if u.maxSize > 0 {
		size, err := file.Seek(0, io.SeekEnd)
		if err != nil {
			return "", err
		}
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return "", err
		}
		if size > u.maxSize {
			return "", fmt.Errorf("file size %d exceeds limit %d", size, u.maxSize)
		}
	}

	if len(u.allowedMime) > 0 {
		head := make([]byte, 512)
		// n is 0 on an empty file; mimetype.Detect handles a zero-length slice.
		n, _ := file.Read(head)
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return "", err
		}
		mt := mimetype.Detect(head[:n])
		if !mimeAllowed(mt, u.allowedMime) {
			return "", fmt.Errorf("mime type %s not allowed", mt.String())
		}
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
		dst.Close()
		os.Remove(dstPath)
		return "", err
	}

	if err := dst.Close(); err != nil {
		os.Remove(dstPath)
		return "", err
	}

	return dstPath, nil
}

func mimeAllowed(mt *mimetype.MIME, allowed []string) bool {
	for _, a := range allowed {
		if mt.Is(a) {
			return true
		}
	}
	return false
}
