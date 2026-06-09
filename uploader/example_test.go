// Copyright 2026 Rahmad Afandi. MIT License.

package uploader_test

import (
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rahmadafandi/fibr/uploader"
)

// Wire an S3-compatible uploader. Upload(file, "photo.png") then returns
// "https://cdn.example.com/avatars/photo.png".
func ExampleNewS3Uploader() {
	client, err := minio.New("s3.amazonaws.com", &minio.Options{
		Creds:  credentials.NewStaticV4("KEY", "SECRET", ""),
		Secure: true,
	})
	if err != nil {
		log.Fatal(err)
	}

	up := uploader.NewS3Uploader(client, "my-bucket",
		uploader.WithKeyPrefix("avatars/"),
		uploader.WithMaxSize(5<<20),
		uploader.WithAllowedMime([]string{"image/png", "image/jpeg"}),
		uploader.WithBaseURL("https://cdn.example.com/"),
	)
	_ = up
}
