package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"golang.org/x/time/rate"

	"GOLA/Handlers/auth"
	"GOLA/Middleware/Authenticators/jwt"
	"GOLA/Middleware/Messengers/KafkaOperations"
	"GOLA/Middleware/MetricsCollectors/Prometheus"
	"GOLA/Middleware/RateLimiters"
	"GOLA/caches/Redis"
	"GOLA/constants"
	"GOLA/storage"
)

// Error handler function
func errorHandler(err error, errorType string) {
	if err != nil {
		fmt.Println(errorType, err)
		return
	}
}

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file: %v", err)
	}

	// Initialize image storage (MinIO or AWS S3)
	storageType := os.Getenv("STORAGE_TYPE") // "minio" or "s3"
	imageManager, err := storage.GetImageManager(storageType)
	errorHandler(err, "ERROR_CREATING_IMAGE_MANAGER")
	imageManager.Initialize()

	// Kafka configuration
	kafkaBrokerAddress := os.Getenv("KAFKA_BROKER_ADDRESS")
	kafkaTopic := os.Getenv("KAFKA_TOPIC")
	kafkaConsumerGroupId := os.Getenv("KAFKA_CONSUMER_GROUP_ID")

	err = KafkaOperations.InitKafkaProducer(kafkaBrokerAddress, kafkaTopic)
	if err != nil {
		log.Fatalf("Failed to initialize Kafka producer: %v", err)
	}
	defer KafkaOperations.CloseProducer()

	// Kafka consumer configuration
	kafkaConfig := KafkaOperations.KafkaConsumerConfig{
		BrokerAddress: kafkaBrokerAddress,
		Topic:         kafkaTopic,
		GroupID:       kafkaConsumerGroupId,
		ImageManager:  imageManager,
	}
	go KafkaOperations.StartKafkaConsumer(kafkaConfig)

	// Rate limiting
	rateLimit, _ := strconv.Atoi(os.Getenv("RATE_LIMITER_LIMIT"))
	rateBurst, _ := strconv.Atoi(os.Getenv("RATE_LIMITER_BURST"))
	rateLimiter := RateLimiters.NewRateLimiter(rate.Limit(rateLimit), rateBurst)

	// Redis cache initialization
	Redis.InitRedis()
	go Prometheus.ExposeMetrics()

	// Authentication route
	http.Handle("/api/authenticate", rateLimiter.Apply(http.HandlerFunc(auth.AuthenticateAndProvideJWT)))

	// Image upload endpoint
	http.Handle("/images/upload",
		Prometheus.CountRequests(
			rateLimiter.Apply(
				jwt.AuthenticateJWT(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodPost {
							KafkaOperations.ImageHandler(constants.UPLOAD_IMAGE, w, r)
						} else {
							http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
						}
					}),
				),
			),
		),
	)

	// Image retrieval endpoint
	http.Handle("/images",
		Prometheus.CountRequests(
			rateLimiter.Apply(
				jwt.AuthenticateJWT(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodGet {
							imageID := r.URL.Query().Get("id")

							// Check Redis cache first
							image, err := Redis.GetImageFromCache(imageID)
							if err != nil {
								fmt.Println("Error while making cache call for images")
							}
							if image != nil {
								w.WriteHeader(http.StatusOK)
								json.NewEncoder(w).Encode(image)
							} else {
								// Cache miss, fetch from DB
								imageManager.GetImageMetadata(w, r)
							}
						} else {
							http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
						}
					}),
				),
			),
		),
	)

	// Image delete endpoint
	http.Handle("/images/delete",
		Prometheus.CountRequests(
			rateLimiter.Apply(
				jwt.AuthenticateJWT(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if r.Method == http.MethodDelete {
							KafkaOperations.ImageHandler(constants.DELETE_IMAGE, w, r)
						} else {
							http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
						}
					}),
				),
			),
		),
	)

	// Start the server
	apiPort := os.Getenv("API_PORT")
	fmt.Printf("Server started on http://localhost%s\n", apiPort)
	if err := http.ListenAndServe(apiPort, nil); err != nil {
		fmt.Println("Error starting server:", err)
		os.Exit(1)
	}
}
