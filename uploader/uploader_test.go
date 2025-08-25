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
	"testing"

	"github.com/stretchr/testify/assert"
)

// a dummy file for testing
type dummyFile struct {
	io.Reader
	io.ReaderAt
	io.Seeker
	io.Closer
}

func (df *dummyFile) Close() error {
	return nil
}

func (df *dummyFile) ReadAt(p []byte, off int64) (n int, err error) {
	return 0, nil
}

func (df *dummyFile) Seek(offset int64, whence int) (int64, error) {
	return 0, nil
}

func TestUploader(t *testing.T) {
	t.Run("LocalUploader", func(t *testing.T) {
		path := "./test_uploads"
		uploader := NewLocalUploader(path)

		// Create a dummy file
		fileContent := []byte("test file")
		file := &dummyFile{bytes.NewReader(fileContent), nil, nil, nil}
		filename := "test.txt"

		// Upload the file
		filePath, err := uploader.Upload(file, filename)
		assert.NoError(t, err)
		assert.Equal(t, path+"/"+filename, filePath)

		// Check if the file exists
		_, err = os.Stat(filePath)
		assert.NoError(t, err)

		// Clean up
		os.RemoveAll(path)
	})
}