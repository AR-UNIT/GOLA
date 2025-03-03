package KafkaOperations

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/segmentio/kafka-go"

	metaDataManager "GOLA/ImageManagers/Metadata"
	rawStoreManager "GOLA/ImageManagers/RawStore"
	"GOLA/UserEventManagers"
	constants "GOLA/constants"
)

// KafkaConsumerConfig holds the configuration for the Kafka consumer.
type KafkaConsumerConfig struct {
	BrokerAddress        string
	Topic                string
	GroupID              string
	EventManager         UserEventManagers.EventManager       // Task/event manager instance.
	ImageMetadataManager metaDataManager.ImageMetadataManager // Image metadata manager.
	ImageStoreManager    rawStoreManager.ImageStoreManager    // Image store manager.
}

// StartKafkaConsumer initializes and starts the Kafka consumer.
func StartKafkaConsumer(config KafkaConsumerConfig) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{config.BrokerAddress},
		Topic:    config.Topic,
		GroupID:  config.GroupID,
		MaxBytes: 10e6, // 10MB max per message.
	})

	log.Printf("Kafka consumer started for topic %s on broker %s", config.Topic, config.BrokerAddress)

	for {
		msg, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Println("Error reading message:", err)
			continue
		}

		log.Printf("Message received: key=%s, value=%s", string(msg.Key), string(msg.Value))

		// Deserialize the Kafka message into a KafkaEvent.
		var event KafkaEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Println("Error deserializing Kafka message:", err)
			continue
		}

		// Process the Kafka event using the full configuration.
		handleKafkaEvent(event, config)
	}
}

// handleKafkaEvent processes a Kafka event based on its type.
func handleKafkaEvent(event KafkaEvent, config KafkaConsumerConfig) {
	log.Printf("Processing event: ID=%s, Type=%s, Endpoint=%s, ClientID=%s",
		event.EventID, event.EventType, event.Endpoint, event.ClientID)
	switch event.EventType {
	// IMAGE EVENTS
	case constants.IMAGE_UPLOAD:
		fmt.Println("Handle ImageUpload event")
		if err := HandleImageUploadEvent(event.Payload, config); err != nil {
			log.Printf("Error processing image upload event: %v", err)
		}
	case constants.IMAGE_UPDATE:
		fmt.Println("Handle ImageUpdate event")
		if err := HandleImageUpdateEvent(event.Payload, config); err != nil {
			log.Printf("Error processing image update event: %v", err)
		}
	case constants.IMAGE_STATS_UPDATE:
		fmt.Println("Handle ImageStatsUpdate event")
		if err := HandleImageStatsUpdateEvent(event.Payload, config); err != nil {
			log.Printf("Error processing image stats update event: %v", err)
		}
	default:
		log.Printf("Unhandled event type: %s", event.EventType)
	}
}

// HandleImageUploadEvent processes an image upload event.
func HandleImageUploadEvent(payload []byte, config KafkaConsumerConfig) error {
	type ImageUploadEvent struct {
		Filename  string `json:"filename"`   // e.g., "myimage.jpg"
		ImageData []byte `json:"image_data"` // raw image bytes (could be base64-encoded in production)
	}
	var event ImageUploadEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("failed to unmarshal image upload event: %w", err)
	}

	if config.ImageStoreManager == nil {
		return fmt.Errorf("image store manager not initialized")
	}

	if err := config.ImageStoreManager.UploadImage(event.Filename, event.ImageData); err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	return nil
}

// HandleImageUpdateEvent processes an image update event.
func HandleImageUpdateEvent(payload []byte, config KafkaConsumerConfig) error {
	type ImageUpdateEvent struct {
		ImageID      string `json:"image_id"`       // ID or filename of the existing image.
		NewImageData []byte `json:"new_image_data"` // New image bytes.
	}
	var event ImageUpdateEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("failed to unmarshal image update event: %w", err)
	}

	if config.ImageStoreManager == nil {
		return fmt.Errorf("image store manager not initialized")
	}

	// Delete the existing image.
	if err := config.ImageStoreManager.DeleteImage(event.ImageID); err != nil {
		return fmt.Errorf("failed to delete old image: %w", err)
	}

	// Upload the new image.
	if err := config.ImageStoreManager.UploadImage(event.ImageID, event.NewImageData); err != nil {
		return fmt.Errorf("failed to upload new image: %w", err)
	}

	return nil
}

// HandleImageStatsUpdateEvent processes an image statistics update event.
func HandleImageStatsUpdateEvent(payload []byte, config KafkaConsumerConfig) error {
	type ImageStatsUpdateEvent struct {
		ImageID  string `json:"image_id"`
		Likes    int    `json:"likes"`
		Dislikes int    `json:"dislikes"`
		Views    int    `json:"views"`
		Comments int    `json:"comments"`
	}
	var event ImageStatsUpdateEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("failed to unmarshal image stats update event: %w", err)
	}

	if config.ImageMetadataManager == nil {
		return fmt.Errorf("metadata manager not initialized")
	}

	// Retrieve the current metadata.
	meta, err := config.ImageMetadataManager.GetImageMetadata(event.ImageID)
	if err != nil {
		return fmt.Errorf("failed to retrieve metadata: %w", err)
	}

	// Update the stats (assuming metadata is stored as a map[string]string).
	meta["likes"] = strconv.Itoa(event.Likes)
	meta["dislikes"] = strconv.Itoa(event.Dislikes)
	meta["views"] = strconv.Itoa(event.Views)
	meta["comments"] = strconv.Itoa(event.Comments)

	if err := config.ImageMetadataManager.SetImageMetadata(event.ImageID, meta); err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	return nil
}
