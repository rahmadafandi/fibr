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
