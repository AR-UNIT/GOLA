package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"GOLA/Deserializers"
	"GOLA/Handlers/auth"
	metadataManager "GOLA/ImageManagers/Metadata"
	rawStoreManager "GOLA/ImageManagers/RawStore"
	"GOLA/Middleware/Authenticators/jwt"
	"GOLA/Middleware/Messengers/KafkaOperations"
	"GOLA/Middleware/MetricsCollectors/Prometheus"
	"GOLA/Middleware/RateLimiters"
	userEventsManager "GOLA/UserEventManagers"
	redisCache "GOLA/caches/Redis"
	"GOLA/constants"

	"github.com/joho/godotenv"
	"golang.org/x/time/rate"
)

// errorHandler prints error messages based on type.
func errorHandler(err error, errorType string) {
	if err != nil {
		fmt.Println(errorType, err)
	}
}

func main() {
	// Load environment variables.
	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file: %v", err)
	}

	// Initialize image storage (MinIO or AWS S3).
	rawImageStoreType := os.Getenv("RAW_IMAGE_STORAGE_TYPE") // e.g. "minio" or "s3"
	imageStoreManager, err := rawStoreManager.GetImageStoreManager(rawImageStoreType)
	errorHandler(err, "ERROR FETCHING IMAGE STORAGE CLIENT")
	err = imageStoreManager.Initialize()
	errorHandler(err, "ERROR INITIALIZING IMAGE STORAGE CLIENT")

	// Initialize image metadata manager (e.g. PostgreSQL).
	metadataImageStoreType := os.Getenv("IMAGE_METADATA_STORAGE_TYPE") // e.g. "postgres"
	imageMetadataManager, err := metadataManager.GetImageMetadataManager(metadataImageStoreType)
	errorHandler(err, "ERROR CREATING IMAGE METADATA MANAGER")
	err = imageMetadataManager.Initialize()
	errorHandler(err, "ERROR INITIALIZING IMAGE METADATA MANAGER")

	// Initialize user events manager (e.g. PostgreSQL).
	userEventsStore := os.Getenv("USER_EVENTS_STORE") // e.g. "postgres"
	eventsManager, err := userEventsManager.GetEventManager(userEventsStore)
	errorHandler(err, "ERROR CREATING USER EVENTS STORE")
	eventsManager.Initialize()

	// Kafka configuration.
	kafkaBrokerAddress := os.Getenv("KAFKA_BROKER_ADDRESS")
	kafkaTopic := os.Getenv("KAFKA_TOPIC")
	kafkaConsumerGroupId := os.Getenv("KAFKA_CONSUMER_GROUP_ID")
	err = KafkaOperations.InitKafkaProducer(kafkaBrokerAddress, kafkaTopic)
	if err != nil {
		log.Fatalf("Failed to initialize Kafka producer: %v", err)
	}
	defer KafkaOperations.CloseProducer()

	// Kafka consumer configuration.
	kafkaConfig := KafkaOperations.KafkaConsumerConfig{
		BrokerAddress:        kafkaBrokerAddress,
		Topic:                kafkaTopic,
		GroupID:              kafkaConsumerGroupId,
		EventManager:         eventsManager,        // Task/event manager instance.
		ImageMetadataManager: imageMetadataManager, // Image metadata manager.
		ImageStoreManager:    imageStoreManager,    // Image store manager.
	}
	go KafkaOperations.StartKafkaConsumer(kafkaConfig)

	// Rate limiting.
	rateLimit, _ := strconv.Atoi(os.Getenv("RATE_LIMITER_LIMIT"))
	rateBurst, _ := strconv.Atoi(os.Getenv("RATE_LIMITER_BURST"))
	rateLimiter := RateLimiters.NewRateLimiter(rate.Limit(rateLimit), rateBurst)

	// Initialize Redis cache (if needed elsewhere).
	redisCache.InitRedis()

	go Prometheus.ExposeMetrics()

	// Authentication endpoint.
	http.Handle("/api/authenticate", rateLimiter.Apply(http.HandlerFunc(auth.AuthenticateAndProvideJWT)))

	// IMAGE UPLOAD endpoint.
	http.Handle("/images/upload",
		Prometheus.CountRequests(
			rateLimiter.Apply(
				jwt.AuthenticateJWT(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodPost {
							KafkaOperations.ImageHandler(constants.IMAGE_UPLOAD, w, r)
						} else {
							http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
						}
					}),
				),
			),
		),
	)

	// IMAGE RETRIEVAL endpoint (fetch binary image data directly from storage).
	http.Handle("/images",
		Prometheus.CountRequests(
			rateLimiter.Apply(
				jwt.AuthenticateJWT(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodGet {
							imageID := r.URL.Query().Get("id")
							if imageID == "" {
								http.Error(w, "Image ID is required", http.StatusBadRequest)
								return
							}

							imageBytes, err := imageStoreManager.FetchImage(imageID)
							if err != nil {
								http.Error(w, "Image not found", http.StatusNotFound)
								return
							}
							w.Header().Set("Content-Type", "image/jpeg")
							w.WriteHeader(http.StatusOK)
							w.Write(imageBytes)
						} else {
							http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
						}
					}),
				),
			),
		),
	)

	// IMAGE DELETE endpoint.
	http.Handle("/images/delete",
		Prometheus.CountRequests(
			rateLimiter.Apply(
				jwt.AuthenticateJWT(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodDelete {
							KafkaOperations.ImageHandler(constants.IMAGE_DELETE, w, r)
						} else {
							http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
						}
					}),
				),
			),
		),
	)

	// IMAGE METADATA endpoint for fetching (GET) and updating (PUT) metadata.
	http.Handle("/images/metadata",
		Prometheus.CountRequests(
			rateLimiter.Apply(
				jwt.AuthenticateJWT(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						switch r.Method {
						case http.MethodGet:
							imageID := r.URL.Query().Get("id")
							if imageID == "" {
								http.Error(w, "Image ID is required", http.StatusBadRequest)
								return
							}
							meta, err := imageMetadataManager.GetImageMetadata(imageID)
							if err != nil {
								http.Error(w, "Error fetching metadata", http.StatusInternalServerError)
								return
							}
							w.Header().Set("Content-Type", "application/json")
							json.NewEncoder(w).Encode(meta)
						case http.MethodPut:
							var payload struct {
								ImageID  string            `json:"image_id"`
								Metadata map[string]string `json:"metadata"`
							}
							if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
								http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
								return
							}
							if payload.ImageID == "" {
								http.Error(w, "Image ID is required", http.StatusBadRequest)
								return
							}
							if err := imageMetadataManager.SetImageMetadata(payload.ImageID, payload.Metadata); err != nil {
								http.Error(w, "Error updating metadata", http.StatusInternalServerError)
								return
							}
							w.WriteHeader(http.StatusOK)
							fmt.Fprintln(w, "Metadata updated successfully")
						default:
							http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
						}
					}),
				),
			),
		),
	)

	// Endpoint to add a like.
	http.Handle("/images/like",
		Prometheus.CountRequests(
			rateLimiter.Apply(
				jwt.AuthenticateJWT(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if r.Method != http.MethodPost {
							http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
							return
						}
						// Read the request body and use the deserializer.
						bodyBytes, err := io.ReadAll(r.Body)
						if err != nil {
							http.Error(w, "Unable to read request body", http.StatusBadRequest)
							return
						}
						eventInput, err := Deserializers.DeserializeEventInput(bodyBytes)
						if err != nil {
							http.Error(w, "Invalid JSON input", http.StatusBadRequest)
							return
						}
						addedEvent := eventsManager.AddLike(eventInput)
						w.Header().Set("Content-Type", "application/json")
						json.NewEncoder(w).Encode(addedEvent)
					}),
				),
			),
		),
	)

	// Endpoint to add a dislike.
	http.Handle("/images/dislike",
		Prometheus.CountRequests(
			rateLimiter.Apply(
				jwt.AuthenticateJWT(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if r.Method != http.MethodPost {
							http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
							return
						}
						bodyBytes, err := io.ReadAll(r.Body)
						if err != nil {
							http.Error(w, "Unable to read request body", http.StatusBadRequest)
							return
						}
						eventInput, err := Deserializers.DeserializeEventInput(bodyBytes)
						addedEvent := eventsManager.AddDislike(eventInput)
						w.Header().Set("Content-Type", "application/json")
						json.NewEncoder(w).Encode(addedEvent)
					}),
				),
			),
		),
	)

	// Endpoint to add a view.
	http.Handle("/images/view",
		Prometheus.CountRequests(
			rateLimiter.Apply(
				jwt.AuthenticateJWT(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if r.Method != http.MethodPost {
							http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
							return
						}
						bodyBytes, err := io.ReadAll(r.Body)
						if err != nil {
							http.Error(w, "Unable to read request body", http.StatusBadRequest)
							return
						}
						eventInput, err := Deserializers.DeserializeEventInput(bodyBytes)
						if err != nil {
							http.Error(w, "Invalid JSON input", http.StatusBadRequest)
							return
						}
						addedEvent := eventsManager.AddView(eventInput)
						w.Header().Set("Content-Type", "application/json")
						json.NewEncoder(w).Encode(addedEvent)
					}),
				),
			),
		),
	)

	// Endpoint to add a comment.
	http.Handle("/images/comment",
		Prometheus.CountRequests(
			rateLimiter.Apply(
				jwt.AuthenticateJWT(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if r.Method != http.MethodPost {
							http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
							return
						}
						bodyBytes, err := io.ReadAll(r.Body)
						if err != nil {
							http.Error(w, "Unable to read request body", http.StatusBadRequest)
							return
						}
						eventInput, err := Deserializers.DeserializeEventInput(bodyBytes)
						if err != nil {
							http.Error(w, "Invalid JSON input", http.StatusBadRequest)
							return
						}
						addedEvent := eventsManager.AddComment(eventInput)
						w.Header().Set("Content-Type", "application/json")
						json.NewEncoder(w).Encode(addedEvent)
					}),
				),
			),
		),
	)

	// Start the server.
	apiPort := os.Getenv("API_PORT")
	fmt.Printf("Server started on http://localhost:%s\n", apiPort)
	if err := http.ListenAndServe(":"+apiPort, nil); err != nil {
		fmt.Println("Error starting server:", err)
		os.Exit(1)
	}
}
