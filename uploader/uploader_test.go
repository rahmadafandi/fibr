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
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// readSeekCloser adapts a *bytes.Reader to multipart.File for tests.
type readSeekCloser struct {
	*bytes.Reader
}

func (readSeekCloser) Close() error { return nil }
func (r readSeekCloser) ReadAt(p []byte, off int64) (int, error) {
	return r.Reader.ReadAt(p, off)
}

func newFile(content []byte) *readSeekCloser {
	return &readSeekCloser{bytes.NewReader(content)}
}

func TestLocalUploader(t *testing.T) {
	path := "./test_uploads"
	defer os.RemoveAll(path)

	t.Run("Basic", func(t *testing.T) {
		u := NewLocalUploader(path)
		filePath, err := u.Upload(newFile([]byte("test file")), "test.txt")
		assert.NoError(t, err)
		assert.Equal(t, filepath.Join(path, "test.txt"), filePath)
		_, err = os.Stat(filePath)
		assert.NoError(t, err)
	})

	t.Run("SanitizesTraversal", func(t *testing.T) {
		u := NewLocalUploader(path)
		_, err := u.Upload(newFile([]byte("x")), "../../etc/evil")
		assert.NoError(t, err) // filepath.Base strips the path...
		// ...and the file lands inside path, named "evil".
		_, statErr := os.Stat(filepath.Join(path, "evil"))
		assert.NoError(t, statErr)
		_, outsideErr := os.Stat("../../etc/evil")
		assert.Error(t, outsideErr)
	})

	t.Run("RejectsEmptyName", func(t *testing.T) {
		u := NewLocalUploader(path)
		_, err := u.Upload(newFile([]byte("x")), "")
		assert.Error(t, err)
	})

	t.Run("MaxSize", func(t *testing.T) {
		u := NewLocalUploader(path, WithMaxSize(4))
		_, err := u.Upload(newFile([]byte("too large")), "big.txt")
		assert.Error(t, err)
	})

	t.Run("AllowedMime", func(t *testing.T) {
		u := NewLocalUploader(path, WithAllowedMime([]string{"image/png"}))
		_, err := u.Upload(newFile([]byte("plain text not png")), "x.txt")
		assert.Error(t, err)
	})

	t.Run("FilePermissions", func(t *testing.T) {
		u := NewLocalUploader(path)
		fp, err := u.Upload(newFile([]byte("perm")), "perm.txt")
		assert.NoError(t, err)
		info, err := os.Stat(fp)
		assert.NoError(t, err)
		assert.Equal(t, os.FileMode(0o640), info.Mode().Perm())
	})

	t.Run("AllowsDotsInName", func(t *testing.T) {
		u := NewLocalUploader(path)
		fp, err := u.Upload(newFile([]byte("v2")), "version..2.tar.gz")
		assert.NoError(t, err)
		assert.Equal(t, filepath.Join(path, "version..2.tar.gz"), fp)
	})
}

var _ io.ReaderAt = readSeekCloser{}
