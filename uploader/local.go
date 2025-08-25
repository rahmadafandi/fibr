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