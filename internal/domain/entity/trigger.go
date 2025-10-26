package entity

import (
	"mkanban/internal/domain/valueobject"
	"time"
)

// TriggerType represents the type of trigger
type TriggerType string

const (
	TriggerTypeTime  TriggerType = "time"
	TriggerTypeEvent TriggerType = "event"
)

// Trigger represents when an action should be executed
type Trigger interface {
	Type() TriggerType
	ShouldTrigger(ctx *TriggerContext) bool
}

// TriggerContext contains information needed to evaluate triggers
type TriggerContext struct {
	CurrentTime time.Time
	Task        *Task
	Column      *Column
	Board       *Board
	Event       *DomainEvent
	LastRun     *time.Time
}

// TimeTrigger represents a time-based trigger
type TimeTrigger struct {
	schedule *valueobject.Schedule
}

// NewTimeTrigger creates a new time-based trigger
func NewTimeTrigger(schedule *valueobject.Schedule) (*TimeTrigger, error) {
	if !schedule.IsValid() {
		return nil, ErrInvalidSchedule
	}
	return &TimeTrigger{
		schedule: schedule,
	}, nil
}

// Type returns the trigger type
func (t *TimeTrigger) Type() TriggerType {
	return TriggerTypeTime
}

// ShouldTrigger evaluates if the time trigger should fire
func (t *TimeTrigger) ShouldTrigger(ctx *TriggerContext) bool {
	switch t.schedule.Type {
	case valueobject.ScheduleTypeAbsolute:
		return t.evaluateAbsolute(ctx)
	case valueobject.ScheduleTypeRelativeDueDate:
		return t.evaluateRelativeDueDate(ctx)
	case valueobject.ScheduleTypeRelativeCreation:
		return t.evaluateRelativeCreation(ctx)
	case valueobject.ScheduleTypeRecurring:
		return t.evaluateRecurring(ctx)
	default:
		return false
	}
}

// evaluateAbsolute checks if absolute time has been reached
func (t *TimeTrigger) evaluateAbsolute(ctx *TriggerContext) bool {
	if t.schedule.Time == nil {
		return false
	}
	// Trigger if current time is past scheduled time and hasn't run yet
	if ctx.CurrentTime.After(*t.schedule.Time) || ctx.CurrentTime.Equal(*t.schedule.Time) {
		// Check if already run (lastRun would be after scheduled time)
		if ctx.LastRun != nil && ctx.LastRun.After(*t.schedule.Time) {
			return false
		}
		return true
	}
	return false
}

// evaluateRelativeDueDate checks if time relative to due date has been reached
func (t *TimeTrigger) evaluateRelativeDueDate(ctx *TriggerContext) bool {
	if ctx.Task == nil || ctx.Task.DueDate() == nil || t.schedule.Offset == nil {
		return false
	}

	dueDate := ctx.Task.DueDate()
	triggerTime := dueDate.Add(-*t.schedule.Offset)

	if ctx.CurrentTime.After(triggerTime) || ctx.CurrentTime.Equal(triggerTime) {
		// Check if already triggered for this due date
		if ctx.LastRun != nil && ctx.LastRun.After(triggerTime) {
			return false
		}
		return true
	}
	return false
}

// evaluateRelativeCreation checks if time relative to creation has been reached
func (t *TimeTrigger) evaluateRelativeCreation(ctx *TriggerContext) bool {
	if ctx.Task == nil || t.schedule.Offset == nil {
		return false
	}

	triggerTime := ctx.Task.CreatedAt().Add(*t.schedule.Offset)

	if ctx.CurrentTime.After(triggerTime) || ctx.CurrentTime.Equal(triggerTime) {
		// Check if already triggered
		if ctx.LastRun != nil && ctx.LastRun.After(triggerTime) {
			return false
		}
		return true
	}
	return false
}

// evaluateRecurring checks if recurring schedule should trigger
func (t *TimeTrigger) evaluateRecurring(ctx *TriggerContext) bool {
	// TODO: Implement cron expression evaluation
	// For now, return false
	// Will need to use a library like robfig/cron for proper evaluation
	return false
}

// Schedule returns the schedule
func (t *TimeTrigger) Schedule() *valueobject.Schedule {
	return t.schedule
}

// EventTrigger represents an event-based trigger
type EventTrigger struct {
	eventType valueobject.EventType
}

// NewEventTrigger creates a new event-based trigger
func NewEventTrigger(eventType valueobject.EventType) (*EventTrigger, error) {
	if !eventType.IsValid() {
		return nil, ErrInvalidEventType
	}
	return &EventTrigger{
		eventType: eventType,
	}, nil
}

// Type returns the trigger type
func (t *EventTrigger) Type() TriggerType {
	return TriggerTypeEvent
}

// ShouldTrigger evaluates if the event trigger should fire
func (t *EventTrigger) ShouldTrigger(ctx *TriggerContext) bool {
	if ctx.Event == nil {
		return false
	}
	return ctx.Event.Type == t.eventType
}

// EventType returns the event type this trigger listens for
func (t *EventTrigger) EventType() valueobject.EventType {
	return t.eventType
}
