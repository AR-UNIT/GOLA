package RawStore

import (
	"errors"
	"github.com/minio/minio-go/v7"
)

// ImageStoreManager defines the required methods for an image storage system.
type ImageStoreManager interface {
	Initialize() error
	UploadImage(imageID string, imageData []byte) error
	DeleteImage(imageID string) error
}

// GetImageStoreManager returns an instance of the requested image storage manager.
func GetImageStoreManager(storageType string, client *minio.Client, bucketName string) (ImageStoreManager, error) {
	switch storageType {
	case "minio":
		manager := &MinioRawImageStorageManager{Client: client, BucketName: bucketName}
		if err := manager.Initialize(); err != nil {
			return nil, err
		}
		return manager, nil
	default:
		return nil, errors.New("unsupported storage type")
	}
}
