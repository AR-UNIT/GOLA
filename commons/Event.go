package commons

import "time"

/* only meant for consumption by specific function that generate the events at kafka producer */
type EventInputModel struct {
	UserID   string `json:"user_id"`
	TargetID string `json:"target_id"`
	// Comment is optional and used only for comment events.
	Comment string `json:"comment,omitempty"`
}

// Event represents an event record stored in the database.
/* this is what is closer to what is stored in the database */
type Event struct {
	ID        int       `json:"id"`
	EventType string    `json:"event_type"`
	UserID    string    `json:"user_id"`
	TargetID  string    `json:"target_id"`
	Comment   string    `json:"comment,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// EventStats aggregates the counts of different event types for a target.
/* this is what could be fetched upon every refresh of a page on an image */
type EventStats struct {
	Likes    int `json:"likes"`
	Dislikes int `json:"dislikes"`
	Views    int `json:"views"`
	Comments int `json:"comments"`
}
