package UserEventManagers

import (
	"GOLA/commons"
	"fmt"
	"net/http"
)

// EventManager interface for handling user events.
type EventManager interface {
	Initialize()
	SaveEvents() error
	AddLike(event *commons.EventInputModel) (addedEvent *commons.Event)
	AddDislike(event *commons.EventInputModel) (addedEvent *commons.Event)
	AddView(event *commons.EventInputModel) (addedEvent *commons.Event)
	AddComment(event *commons.EventInputModel) (addedEvent *commons.Event)
	ListEvents(w http.ResponseWriter, r *http.Request)

	// GetStats fetches the aggregated stats (likes, dislikes, views, comments) for a given target.
	GetStats(targetID string) (*commons.EventStats, error)
	// GetStatsHandler is an HTTP handler that writes the stats for a given target.
	GetStatsHandler(w http.ResponseWriter, r *http.Request)

	// GetComments fetches all comment events for a given target.
	GetComments(targetID string) ([]commons.Event, error)
	// GetCommentsHandler is an HTTP handler that writes the comments for a given target.
	GetCommentsHandler(w http.ResponseWriter, r *http.Request)

	LazySave()
}

// Factory method to create the appropriate EventManager.
func GetEventManager(sourceType string) (EventManager, error) {
	switch sourceType {
	case "postgresDb":
		return &DatabaseEventManager{}, nil
	default:
		return nil, fmt.Errorf("unsupported event manager type: %s", sourceType)
	}
}
