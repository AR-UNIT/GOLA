package UserEventManagers

import (
	"GOLA/DbQueryStrategies"
	"GOLA/DbQueryStrategies/PostgresDb"
	"GOLA/commons/db"

	"GOLA/commons"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // Import the Postgres driver
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

// SelectEventStrategy determines which DB strategy to use based on configuration.
// Here we reuse the same strategies as tasks. Adjust if you have event‑specific strategies.
func SelectEventStrategy(db *sql.DB, queryStrategy string) DbQueryStrategies.DatabaseQueryStrategy {
	switch queryStrategy {
	case "rowLockingStrategy":
		return &PostgresDb.RowLockingStrategy{
			DbContext: PostgresDb.DbContext{Db: db},
		}
	case "enhancedListStrategy":
		return &PostgresDb.EnhancedListStrategy{
			DbContext: PostgresDb.DbContext{Db: db},
		}
	case "combinedRowLockingEnhancedListStrategy":
		return &PostgresDb.CombinedRowLockingEnhancedListStrategy{
			RowLockingStrategy: PostgresDb.RowLockingStrategy{
				DbContext: PostgresDb.DbContext{Db: db},
			},
			EnhancedListStrategy: PostgresDb.EnhancedListStrategy{
				DbContext: PostgresDb.DbContext{Db: db},
			},
		}
	default:
		return &PostgresDb.DefaultPostgresStrategy{
			DbContext: PostgresDb.DbContext{Db: db},
		}
	}
}

// InitializeDB creates and returns a database connection using provided configuration.
func InitializeDB(config db.DBConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host,
		config.Port,
		config.User,
		config.Password,
		config.DbName,
		config.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("Successfully connected to the database!")
	return db, nil
}

// DatabaseEventManager implements the EventManager interface using a PostgreSQL database.
type DatabaseEventManager struct {
	Db       *sql.DB
	strategy DbQueryStrategies.DatabaseQueryStrategy
	mu       sync.Mutex
}

// Initialize loads environment variables, establishes the database connection,
// selects the query strategy, and creates the events table if needed.
func (dem *DatabaseEventManager) Initialize() {
	// Load .env file if present.
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, assuming environment variables are set")
	}

	// Read connection settings from environment variables.
	host := os.Getenv("DB_HOST")
	portStr := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	sslMode := os.Getenv("DB_SSLMODE")

	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatalf("Invalid DB_PORT value: %v", err)
	}

	// Build DB configuration.
	config := db.DBConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		DbName:   dbName,
		SSLMode:  sslMode,
	}

	// Initialize the database connection.
	db, err := InitializeDB(config)
	if err != nil || db == nil {
		log.Fatalf("Error initializing database: %v", err)
	}
	dem.Db = db
	log.Printf("Database connection initialized successfully to %s:%d", host, port)

	// Select the query strategy.
	dbQueryStrategy := os.Getenv("DB_QUERY_STRATEGY")
	dem.strategy = SelectEventStrategy(dem.Db, dbQueryStrategy)
	log.Println("Current query strategy: ", dbQueryStrategy)

	// Create the events table if it does not exist.
	dem.createEventsTable()
}

// createEventsTable creates the user_events table if it doesn't already exist.
func (dem *DatabaseEventManager) createEventsTable() {
	query := `
	CREATE TABLE IF NOT EXISTS user_events (
		id SERIAL PRIMARY KEY,
		event_type VARCHAR(50) NOT NULL,
		user_id VARCHAR(100) NOT NULL,
		target_id VARCHAR(100) NOT NULL,
		comment TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT NOW()
	);
	`
	if _, err := dem.Db.Exec(query); err != nil {
		log.Fatalf("Failed to create events table: %v", err)
	}
}

// SaveEvents flushes any buffered events if needed (here it is a no-op).
func (dem *DatabaseEventManager) SaveEvents() error {
	return nil
}

// LazySave can be used for deferred/batch persistence. Currently, it is a no-op.
func (dem *DatabaseEventManager) LazySave() {
	// Implement deferred save if necessary.
}

// AddLike records a "user-like" event into the database.
func (dem *DatabaseEventManager) AddLike(event *commons.EventInputModel) (addedEvent *commons.Event) {
	dem.mu.Lock()
	defer dem.mu.Unlock()

	query := `
		INSERT INTO user_events (event_type, user_id, target_id, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at;
	`
	var id int
	var createdAt time.Time
	if err := dem.Db.QueryRow(query, "user-like", event.UserID, event.TargetID, time.Now()).Scan(&id, &createdAt); err != nil {
		log.Printf("Error adding like event: %v", err)
		return nil
	}
	return &commons.Event{
		ID:        id,
		EventType: "user-like",
		UserID:    event.UserID,
		TargetID:  event.TargetID,
		CreatedAt: createdAt,
	}
}

// AddDislike records a "user-dislike" event.
func (dem *DatabaseEventManager) AddDislike(event *commons.EventInputModel) (addedEvent *commons.Event) {
	dem.mu.Lock()
	defer dem.mu.Unlock()

	query := `
		INSERT INTO user_events (event_type, user_id, target_id, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at;
	`
	var id int
	var createdAt time.Time
	if err := dem.Db.QueryRow(query, "user-dislike", event.UserID, event.TargetID, time.Now()).Scan(&id, &createdAt); err != nil {
		log.Printf("Error adding dislike event: %v", err)
		return nil
	}
	return &commons.Event{
		ID:        id,
		EventType: "user-dislike",
		UserID:    event.UserID,
		TargetID:  event.TargetID,
		CreatedAt: createdAt,
	}
}

// AddView records a "user-view" event.
func (dem *DatabaseEventManager) AddView(event *commons.EventInputModel) (addedEvent *commons.Event) {
	dem.mu.Lock()
	defer dem.mu.Unlock()

	query := `
		INSERT INTO user_events (event_type, user_id, target_id, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at;
	`
	var id int
	var createdAt time.Time
	if err := dem.Db.QueryRow(query, "user-view", event.UserID, event.TargetID, time.Now()).Scan(&id, &createdAt); err != nil {
		log.Printf("Error adding view event: %v", err)
		return nil
	}
	return &commons.Event{
		ID:        id,
		EventType: "user-view",
		UserID:    event.UserID,
		TargetID:  event.TargetID,
		CreatedAt: createdAt,
	}
}

// AddComment records a "user-comment" event, including the comment text.
func (dem *DatabaseEventManager) AddComment(event *commons.EventInputModel) (addedEvent *commons.Event) {
	dem.mu.Lock()
	defer dem.mu.Unlock()

	query := `
		INSERT INTO user_events (event_type, user_id, target_id, comment, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at;
	`
	var id int
	var createdAt time.Time
	if err := dem.Db.QueryRow(query, "user-comment", event.UserID, event.TargetID, event.Comment, time.Now()).Scan(&id, &createdAt); err != nil {
		log.Printf("Error adding comment event: %v", err)
		return nil
	}
	return &commons.Event{
		ID:        id,
		EventType: "user-comment",
		UserID:    event.UserID,
		TargetID:  event.TargetID,
		Comment:   event.Comment,
		CreatedAt: createdAt,
	}
}

// GetStats fetches aggregated event counts for a specific target.
func (dem *DatabaseEventManager) GetStats(targetID string) (*commons.EventStats, error) {
	query := `
		SELECT event_type, COUNT(*) 
		FROM user_events
		WHERE target_id = $1
		GROUP BY event_type
	`
	rows, err := dem.Db.Query(query, targetID)
	if err != nil {
		return nil, fmt.Errorf("error querying stats: %w", err)
	}
	defer rows.Close()

	stats := &commons.EventStats{}
	for rows.Next() {
		var eventType string
		var count int
		if err := rows.Scan(&eventType, &count); err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		switch eventType {
		case "user-like":
			stats.Likes = count
		case "user-dislike":
			stats.Dislikes = count
		case "user-view":
			stats.Views = count
		case "user-comment":
			stats.Comments = count
		}
	}
	return stats, nil
}

// GetStatsHandler is an HTTP handler that returns event stats for a given target.
func (dem *DatabaseEventManager) GetStatsHandler(w http.ResponseWriter, r *http.Request) {
	// Expect the target id as a query parameter.
	targetID := r.URL.Query().Get("target_id")
	if targetID == "" {
		http.Error(w, "target_id is required", http.StatusBadRequest)
		return
	}

	stats, err := dem.GetStats(targetID)
	if err != nil {
		log.Printf("Error fetching stats: %v", err)
		http.Error(w, "Failed to fetch stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}

// GetComments fetches all comment events for a specific target.
func (dem *DatabaseEventManager) GetComments(targetID string) ([]commons.Event, error) {
	query := `
		SELECT id, event_type, user_id, target_id, comment, created_at
		FROM user_events
		WHERE target_id = $1 AND event_type = 'user-comment'
		ORDER BY created_at ASC
	`
	rows, err := dem.Db.Query(query, targetID)
	if err != nil {
		return nil, fmt.Errorf("error querying comments: %w", err)
	}
	defer rows.Close()

	var comments []commons.Event
	for rows.Next() {
		var ev commons.Event
		if err := rows.Scan(&ev.ID, &ev.EventType, &ev.UserID, &ev.TargetID, &ev.Comment, &ev.CreatedAt); err != nil {
			return nil, fmt.Errorf("error scanning comment row: %w", err)
		}
		comments = append(comments, ev)
	}
	return comments, nil
}

// GetCommentsHandler is an HTTP handler that returns comments for a given target.
func (dem *DatabaseEventManager) GetCommentsHandler(w http.ResponseWriter, r *http.Request) {
	targetID := r.URL.Query().Get("target_id")
	if targetID == "" {
		http.Error(w, "target_id is required", http.StatusBadRequest)
		return
	}

	comments, err := dem.GetComments(targetID)
	if err != nil {
		log.Printf("Error fetching comments: %v", err)
		http.Error(w, "Failed to fetch comments", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(comments); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}

// ListEvents retrieves all events from the database and writes them as JSON.
func (dem *DatabaseEventManager) ListEvents(w http.ResponseWriter, r *http.Request) {
	dem.mu.Lock()
	defer dem.mu.Unlock()

	rows, err := dem.Db.Query(`
		SELECT id, event_type, user_id, target_id, comment, created_at
		FROM user_events
		ORDER BY created_at DESC
	`)
	if err != nil {
		http.Error(w, "Failed to query events", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var events []commons.Event
	for rows.Next() {
		var ev commons.Event
		if err := rows.Scan(&ev.ID, &ev.EventType, &ev.UserID, &ev.TargetID, &ev.Comment, &ev.CreatedAt); err != nil {
			log.Printf("Error scanning event: %v", err)
			continue
		}
		events = append(events, ev)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}
