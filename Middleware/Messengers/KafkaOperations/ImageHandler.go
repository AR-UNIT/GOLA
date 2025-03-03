package KafkaOperations

import (
	"GOLA/ImageManagers/Metadata"
	"GOLA/ImageManagers/RawStore"
	"GOLA/constants"
	"encoding/json"
	"io"
	"net/http"
)

// Global managers initialized in main.
var imageStoreManager RawStore.ImageStoreManager
var imageMetadataManager Metadata.ImageMetadataManager

// SetManagers is called from main after initialization to set the managers.
func SetManagers(storeManager RawStore.ImageStoreManager, metadataManager Metadata.ImageMetadataManager) {
	imageStoreManager = storeManager
	imageMetadataManager = metadataManager
}

// ImageTask represents an image-related event.
type ImageTask struct {
	Action  string `json:"action"` // "upload", "delete", or "metadata"
	ImageID string `json:"image_id"`
	UserID  string `json:"user_id"`
}

// ImageHandler processes image-related HTTP requests.
func ImageHandler(action string, w http.ResponseWriter, r *http.Request) {
	switch action {
	case constants.IMAGE_UPLOAD:
		handleImageUpload(w, r)
	case constants.IMAGE_DELETE:
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

	// Read image data.
	imageData, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to process image", http.StatusInternalServerError)
		return
	}

	// Ensure the ImageStoreManager is initialized.
	if imageStoreManager == nil {
		http.Error(w, "Image store manager not initialized", http.StatusInternalServerError)
		return
	}

	// Upload the image.
	err = imageStoreManager.UploadImage(header.Filename, imageData)
	if err != nil {
		http.Error(w, "Upload failed", http.StatusInternalServerError)
		return
	}

	// Ensure the ImageMetadataManager is initialized.
	if imageMetadataManager == nil {
		http.Error(w, "Metadata manager not initialized", http.StatusInternalServerError)
		return
	}

	// Retrieve metadata for the uploaded image.
	meta, err := imageMetadataManager.GetImageMetadata(header.Filename)
	if err != nil {
		http.Error(w, "Metadata retrieval failed", http.StatusInternalServerError)
		return
	}

	// Convert metadata to JSON and send response.
	jsonResponse, err := json.Marshal(meta)
	if err != nil {
		http.Error(w, "JSON conversion failed", http.StatusInternalServerError)
		return
	}
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

	// Ensure the ImageStoreManager is initialized.
	if imageStoreManager == nil {
		http.Error(w, "Image store manager not initialized", http.StatusInternalServerError)
		return
	}

	// Delete the image.
	err := imageStoreManager.DeleteImage(imageID)
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

	// Ensure the ImageMetadataManager is initialized.
	if imageMetadataManager == nil {
		http.Error(w, "Metadata manager not initialized", http.StatusInternalServerError)
		return
	}

	// Retrieve metadata for the specified image.
	meta, err := imageMetadataManager.GetImageMetadata(imageID)
	if err != nil {
		http.Error(w, "Failed to retrieve metadata", http.StatusInternalServerError)
		return
	}

	// Convert metadata to JSON and send response.
	jsonResponse, err := json.Marshal(meta)
	if err != nil {
		http.Error(w, "JSON conversion failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}
