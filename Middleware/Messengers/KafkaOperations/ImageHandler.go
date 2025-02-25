package KafkaOperations

import (
	"GOLA/ImageManagers/Metadata"
	"GOLA/ImageManagers/RawStore"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
)

// ImageTask represents an image-related event.
type ImageTask struct {
	Action  string `json:"action"` // "upload" or "delete"
	ImageID string `json:"image_id"`
	UserID  string `json:"user_id"`
}

// ImageHandler processes image-related HTTP requests.
func ImageHandler(action string, w http.ResponseWriter, r *http.Request) {
	switch action {
	case "upload":
		handleImageUpload(w, r)
	case "delete":
		handleImageDelete(w, r)
	case "metadata":
		handleImageMetadata(w, r)
	default:
		http.Error(w, "Invalid action", http.StatusBadRequest)
	}
}

// handleImageUpload processes image uploads.
func handleImageUpload(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20) // 10MB limit
	if err != nil {
		http.Error(w, "File too large", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Failed to read image", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Read image data
	imageData, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to process image", http.StatusInternalServerError)
		return
	}

	// Get storage type from environment variable
	storageType := os.Getenv("IMAGE_STORAGE_TYPE") // Expected values: "minio", "s3", etc.
	if storageType == "" {
		http.Error(w, "Storage type not configured", http.StatusInternalServerError)
		return
	}

	// Get ImageStoreManager based on storageType
	imageManager, err := RawStore.GetImageStoreManager(storageType, nil, "images")
	if err != nil {
		log.Printf("Failed to get storage manager: %v", err)
		http.Error(w, "Invalid storage type", http.StatusBadRequest)
		return
	}

	// Upload to storage
	err = imageManager.UploadImage(header.Filename, imageData)
	if err != nil {
		http.Error(w, "Upload failed", http.StatusInternalServerError)
		return
	}

	// Get Metadata Manager dynamically
	metadataManager, err := Metadata.GetMetadataManager(storageType, nil, "images")
	if err != nil {
		log.Printf("Failed to get metadata manager: %v", err)
		http.Error(w, "Metadata error", http.StatusInternalServerError)
		return
	}

	// Fetch Metadata
	meta, err := metadataManager.GetImageMetadata(header.Filename)
	if err != nil {
		log.Printf("Failed to retrieve metadata: %v", err)
		http.Error(w, "Metadata retrieval failed", http.StatusInternalServerError)
		return
	}

	// Convert Metadata to JSON
	jsonResponse, _ := json.Marshal(meta)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}

// handleImageDelete processes image deletion.
func handleImageDelete(w http.ResponseWriter, r *http.Request) {
	imageID := r.URL.Query().Get("id")
	if imageID == "" {
		http.Error(w, "Image ID required", http.StatusBadRequest)
		return
	}

	// Get storage type from environment variable
	storageType := os.Getenv("IMAGE_STORAGE_TYPE")
	if storageType == "" {
		http.Error(w, "Storage type not configured", http.StatusInternalServerError)
		return
	}

	// Get ImageStoreManager based on storageType
	imageManager, err := RawStore.GetImageStoreManager(storageType, nil, "images")
	if err != nil {
		http.Error(w, "Invalid storage type", http.StatusBadRequest)
		return
	}

	// Delete the image
	err = imageManager.DeleteImage(imageID)
	if err != nil {
		http.Error(w, "Failed to delete image", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Image deleted successfully"))
}

// handleImageMetadata retrieves image metadata.
func handleImageMetadata(w http.ResponseWriter, r *http.Request) {
	imageID := r.URL.Query().Get("id")
	if imageID == "" {
		http.Error(w, "Image ID required", http.StatusBadRequest)
		return
	}

	// Get storage type from environment variable
	storageType := os.Getenv("IMAGE_STORAGE_TYPE")
	if storageType == "" {
		http.Error(w, "Storage type not configured", http.StatusInternalServerError)
		return
	}

	// Get Metadata Manager dynamically
	metadataManager, err := Metadata.GetMetadataManager(storageType, nil, "images")
	if err != nil {
		http.Error(w, "Metadata error", http.StatusInternalServerError)
		return
	}

	// Fetch Metadata
	meta, err := metadataManager.GetImageMetadata(imageID)
	if err != nil {
		http.Error(w, "Failed to retrieve metadata", http.StatusInternalServerError)
		return
	}

	// Convert Metadata to JSON
	jsonResponse, _ := json.Marshal(meta)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}
