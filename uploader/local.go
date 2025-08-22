package uploader

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

// LocalUploader is an uploader that saves files to a local directory.
type LocalUploader struct {
	Path string
}

// NewLocalUploader creates a new LocalUploader.
func NewLocalUploader(path string) *LocalUploader {
	return &LocalUploader{Path: path}
}

// Upload uploads a file to a local directory.
func (u *LocalUploader) Upload(file multipart.File, filename string) (string, error) {
	// Create the directory if it doesn't exist
	if err := os.MkdirAll(u.Path, os.ModePerm); err != nil {
		return "", err
	}

	// Create the file
	dst, err := os.Create(filepath.Join(u.Path, filename))
	if err != nil {
		return "", err
	}
	defer dst.Close()

	// Copy the file
	if _, err = io.Copy(dst, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s", u.Path, filename), nil
}
