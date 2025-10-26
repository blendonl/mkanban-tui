package service

import (
	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/valueobject"
	"sync"
)

// EventBusImpl implements the EventBus interface
type EventBusImpl struct {
	mu         sync.RWMutex
	handlers   map[valueobject.EventType][]entity.EventHandler
	allHandlers []entity.EventHandler // Handlers that receive all events
}

// NewEventBus creates a new event bus
func NewEventBus() entity.EventBus {
	return &EventBusImpl{
		handlers:    make(map[valueobject.EventType][]entity.EventHandler),
		allHandlers: make([]entity.EventHandler, 0),
	}
}

// Publish publishes an event to all subscribed handlers
func (bus *EventBusImpl) Publish(event *entity.DomainEvent) {
	bus.mu.RLock()
	defer bus.mu.RUnlock()

	// Call handlers for specific event type
	if handlers, exists := bus.handlers[event.Type]; exists {
		for _, handler := range handlers {
			// Execute handler in goroutine to not block publisher
			go handler(event)
		}
	}

	// Call handlers that receive all events
	for _, handler := range bus.allHandlers {
		go handler(event)
	}
}

// Subscribe subscribes a handler to a specific event type
func (bus *EventBusImpl) Subscribe(eventType valueobject.EventType, handler entity.EventHandler) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	bus.handlers[eventType] = append(bus.handlers[eventType], handler)
}

// Unsubscribe unsubscribes a handler from a specific event type
func (bus *EventBusImpl) Unsubscribe(eventType valueobject.EventType, handler entity.EventHandler) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	handlers := bus.handlers[eventType]
	// Find and remove the handler
	for i, h := range handlers {
		// Compare function pointers (note: this may not work for all cases)
		if &h == &handler {
			bus.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
			return
		}
	}
}

// SubscribeAll subscribes a handler to all events
func (bus *EventBusImpl) SubscribeAll(handler entity.EventHandler) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	bus.allHandlers = append(bus.allHandlers, handler)
}
