package Metadata

import (
	dbCommons "GOLA/commons/db"
	"database/sql"
	"encoding/json"
	"errors"
	_ "github.com/lib/pq"
)

// PostgresImageMetadataManager manages image metadata using PostgreSQL.
type PostgresImageMetadataManager struct {
	DB *sql.DB
}

// NewPostgresImageMetadataManager initializes a new PostgreSQL metadata manager using the provided DBConfig.
func NewPostgresImageMetadataManager(config dbCommons.DBConfig) (*PostgresImageMetadataManager, error) {
	// Use the InitializeDB function for connection management.
	db, err := dbCommons.InitializeDB(config)
	if err != nil {
		return nil, err
	}

	manager := &PostgresImageMetadataManager{DB: db}
	if err := manager.Initialize(); err != nil {
		return nil, err
	}
	return manager, nil
}

// Initialize ensures the metadata table exists.
func (p *PostgresImageMetadataManager) Initialize() error {
	query := `CREATE TABLE IF NOT EXISTS image_metadata (
		image_id TEXT PRIMARY KEY,
		metadata JSONB NOT NULL
	)`

	_, err := p.DB.Exec(query)
	return err
}

// GetImageMetadata retrieves metadata for a given image ID.
func (p *PostgresImageMetadataManager) GetImageMetadata(imageID string) (map[string]string, error) {
	var jsonMetadata []byte
	query := `SELECT metadata FROM image_metadata WHERE image_id = $1`
	row := p.DB.QueryRow(query, imageID)
	if err := row.Scan(&jsonMetadata); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("metadata not found")
		}
		return nil, err
	}

	var metadata map[string]string
	err := json.Unmarshal(jsonMetadata, &metadata)
	if err != nil {
		return nil, err
	}

	return metadata, nil
}

// SetImageMetadata saves metadata for an image.
func (p *PostgresImageMetadataManager) SetImageMetadata(imageID string, metadata map[string]string) error {
	jsonMetadata, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	query := `INSERT INTO image_metadata (image_id, metadata) VALUES ($1, $2)
		ON CONFLICT (image_id) DO UPDATE SET metadata = EXCLUDED.metadata`
	_, err = p.DB.Exec(query, imageID, jsonMetadata)
	return err
}

// DeleteImageMetadata deletes metadata for a given image ID.
func (p *PostgresImageMetadataManager) DeleteImageMetadata(imageID string) error {
	query := `DELETE FROM image_metadata WHERE image_id = $1`
	_, err := p.DB.Exec(query, imageID)
	return err
}
