package uploader

import "mime/multipart"

// Uploader is an interface for uploading files.
type Uploader interface {
	Upload(file multipart.File, filename string) (string, error)
}
