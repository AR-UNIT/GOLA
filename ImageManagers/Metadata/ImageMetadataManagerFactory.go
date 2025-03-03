package Metadata

import (
	"errors"
)

// ImageMetadataManager defines the required methods for managing image metadata.
type ImageMetadataManager interface {
	Initialize() error
	GetImageMetadata(imageID string) (map[string]string, error) /* THIS DOES NOT NEED TO BE DONE BY KAFKA */
	SetImageMetadata(imageID string, metadata map[string]string) error
	DeleteImageMetadata(imageID string) error
}

// GetImageMetadataManager returns an instance of the requested image metadata manager.
func GetImageMetadataManager(storageType string) (ImageMetadataManager, error) {
	switch storageType {
	case "postgres":
		return &PostgresImageMetadataManager{}, nil
	default:
		return nil, errors.New("unsupported metadata storage type")
	}
}
