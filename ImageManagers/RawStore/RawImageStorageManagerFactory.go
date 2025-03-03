package RawStore

import (
	"errors"
	"os"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// ImageStoreManager defines the required methods for an image storage system.
type ImageStoreManager interface {
	Initialize() error
	UploadImage(imageID string, imageData []byte) error
	DeleteImage(imageID string) error
	FetchImage(imageID string) ([]byte, error) /* THIS DOES NOT NEED TO BE DONE BY KAFKA */
}

// GetImageStoreManager returns an instance of the requested image storage manager.
func GetImageStoreManager(storageType string) (ImageStoreManager, error) {
	switch storageType {
	case "minio":
		// Read configuration from environment variables.
		endpoint := os.Getenv("MINIO_ENDPOINT")        // e.g., "localhost:9000"
		accessKey := os.Getenv("MINIO_ACCESS_KEY")     // e.g., "minioadmin"
		secretKey := os.Getenv("MINIO_SECRET_KEY")     // e.g., "minioadmin"
		useSSL := os.Getenv("MINIO_USE_SSL") == "true" // e.g., "false"
		bucketName := os.Getenv("MINIO_BUCKET_NAME")   // e.g., "images"

		// Create a new MinIO client.
		client, err := minio.New(endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
			Secure: useSSL,
		})
		if err != nil {
			return nil, err
		}

		// Initialize the custom manager.
		manager := &MinioRawImageStorageManager{
			Client:     client,
			BucketName: bucketName,
		}
		if err := manager.Initialize(); err != nil {
			return nil, err
		}
		return manager, nil

	default:
		return nil, errors.New("unsupported storage type")
	}
}
