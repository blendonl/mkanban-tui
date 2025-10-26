package valueobject

import (
	"fmt"
	"time"
)

// ScheduleType represents the type of schedule
type ScheduleType string

const (
	// ScheduleTypeAbsolute represents a specific date/time
	ScheduleTypeAbsolute ScheduleType = "absolute"
	// ScheduleTypeRelativeDueDate represents time relative to due date
	ScheduleTypeRelativeDueDate ScheduleType = "relative_due_date"
	// ScheduleTypeRelativeCreation represents time relative to creation
	ScheduleTypeRelativeCreation ScheduleType = "relative_creation"
	// ScheduleTypeRecurring represents a recurring schedule (cron-like)
	ScheduleTypeRecurring ScheduleType = "recurring"
)

// Schedule represents when an action should be triggered
type Schedule struct {
	Type     ScheduleType
	// For absolute schedules
	Time *time.Time
	// For relative schedules (duration before/after)
	Offset *time.Duration
	// For recurring schedules (cron expression)
	CronExpr string
}

// NewAbsoluteSchedule creates a schedule for a specific time
func NewAbsoluteSchedule(t time.Time) *Schedule {
	return &Schedule{
		Type: ScheduleTypeAbsolute,
		Time: &t,
	}
}

// NewRelativeDueDateSchedule creates a schedule relative to due date
func NewRelativeDueDateSchedule(offset time.Duration) *Schedule {
	return &Schedule{
		Type:   ScheduleTypeRelativeDueDate,
		Offset: &offset,
	}
}

// NewRelativeCreationSchedule creates a schedule relative to creation time
func NewRelativeCreationSchedule(offset time.Duration) *Schedule {
	return &Schedule{
		Type:   ScheduleTypeRelativeCreation,
		Offset: &offset,
	}
}

// NewRecurringSchedule creates a recurring schedule with cron expression
func NewRecurringSchedule(cronExpr string) (*Schedule, error) {
	// TODO: Validate cron expression
	if cronExpr == "" {
		return nil, fmt.Errorf("cron expression cannot be empty")
	}
	return &Schedule{
		Type:     ScheduleTypeRecurring,
		CronExpr: cronExpr,
	}, nil
}

// IsValid checks if the schedule is valid
func (s *Schedule) IsValid() bool {
	switch s.Type {
	case ScheduleTypeAbsolute:
		return s.Time != nil
	case ScheduleTypeRelativeDueDate, ScheduleTypeRelativeCreation:
		return s.Offset != nil
	case ScheduleTypeRecurring:
		return s.CronExpr != ""
	default:
		return false
	}
}
