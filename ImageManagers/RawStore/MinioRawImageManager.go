package RawStore

import (
	"bytes"
	"context"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"io"
	"log"
)

// MinioRawImageStorageManager manages image storage using MinIO.
type MinioRawImageStorageManager struct {
	Client     *minio.Client
	BucketName string
}

// NewMinioClient initializes a new MinIO client.
func NewMinioClient(endpoint, accessKey, secretKey string, useSSL bool) (*minio.Client, error) {
	return minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
}

// Initialize ensures the bucket exists in MinIO.
func (m *MinioRawImageStorageManager) Initialize() error {
	exists, err := m.Client.BucketExists(context.Background(), m.BucketName)
	if err != nil {
		return err
	}
	if !exists {
		err = m.Client.MakeBucket(context.Background(), m.BucketName, minio.MakeBucketOptions{})
		if err != nil {
			return err
		}
		log.Printf("Bucket %s created", m.BucketName)
	}
	return nil
}

// UploadImage uploads an image to MinIO.
func (m *MinioRawImageStorageManager) UploadImage(imageID string, imageData []byte) error {
	reader := io.NopCloser(io.Reader(bytes.NewReader(imageData)))
	_, err := m.Client.PutObject(context.Background(), m.BucketName, imageID, reader, int64(len(imageData)), minio.PutObjectOptions{})
	return err
}

// DeleteImage removes an image from MinIO.
func (m *MinioRawImageStorageManager) DeleteImage(imageID string) error {
	return m.Client.RemoveObject(context.Background(), m.BucketName, imageID, minio.RemoveObjectOptions{})
}
