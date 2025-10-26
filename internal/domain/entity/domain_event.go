package entity

import (
	"mkanban/internal/domain/valueobject"
	"time"
)

// DomainEvent represents an event that occurred in the domain
type DomainEvent struct {
	ID        string
	Type      valueobject.EventType
	Timestamp time.Time
	BoardID   string
	ColumnID  string
	TaskID    *valueobject.TaskID
	Metadata  map[string]interface{}
}

// NewDomainEvent creates a new domain event
func NewDomainEvent(
	eventType valueobject.EventType,
	boardID string,
	columnID string,
	taskID *valueobject.TaskID,
	metadata map[string]interface{},
) *DomainEvent {
	return &DomainEvent{
		ID:        generateEventID(),
		Type:      eventType,
		Timestamp: time.Now(),
		BoardID:   boardID,
		ColumnID:  columnID,
		TaskID:    taskID,
		Metadata:  metadata,
	}
}

// generateEventID generates a unique event ID
func generateEventID() string {
	// Simple implementation - in production might want UUID
	return time.Now().Format("20060102150405.000000")
}

// EventBus handles publishing and subscribing to domain events
type EventBus interface {
	Publish(event *DomainEvent)
	Subscribe(eventType valueobject.EventType, handler EventHandler)
	Unsubscribe(eventType valueobject.EventType, handler EventHandler)
}

// EventHandler is a function that handles domain events
type EventHandler func(event *DomainEvent)
