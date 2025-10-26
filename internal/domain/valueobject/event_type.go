package valueobject

// EventType represents a domain event that can trigger an action
type EventType string

const (
	// Task lifecycle events
	EventTaskCreated EventType = "task.created"
	EventTaskUpdated EventType = "task.updated"
	EventTaskDeleted EventType = "task.deleted"
	EventTaskMoved   EventType = "task.moved"

	// Task state change events
	EventTaskStatusChanged   EventType = "task.status_changed"
	EventTaskPriorityChanged EventType = "task.priority_changed"
	EventTaskDueDateSet      EventType = "task.due_date_set"
	EventTaskDueDateChanged  EventType = "task.due_date_changed"
	EventTaskCompleted       EventType = "task.completed"

	// Due date events
	EventTaskDueApproaching EventType = "task.due_approaching"
	EventTaskOverdue        EventType = "task.overdue"
	EventTaskCompletedOnTime EventType = "task.completed_on_time"

	// Column events
	EventColumnCreated      EventType = "column.created"
	EventColumnDeleted      EventType = "column.deleted"
	EventColumnWIPReached   EventType = "column.wip_reached"
)

// IsValid checks if the event type is valid
func (e EventType) IsValid() bool {
	switch e {
	case EventTaskCreated, EventTaskUpdated, EventTaskDeleted, EventTaskMoved,
		EventTaskStatusChanged, EventTaskPriorityChanged, EventTaskDueDateSet,
		EventTaskDueDateChanged, EventTaskCompleted, EventTaskDueApproaching,
		EventTaskOverdue, EventTaskCompletedOnTime, EventColumnCreated,
		EventColumnDeleted, EventColumnWIPReached:
		return true
	default:
		return false
	}
}

// String returns the string representation of the event type
func (e EventType) String() string {
	return string(e)
}
