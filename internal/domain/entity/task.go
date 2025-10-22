package entity

import (
	"mkanban/internal/domain/valueobject"
	"time"
)

// Task represents a work item within a column
type Task struct {
	id            *valueobject.TaskID
	title         string
	description   string
	priority      valueobject.Priority
	status        valueobject.Status
	tags          []string
	createdAt     time.Time
	modifiedAt    time.Time
	dueDate       *time.Time
	completedDate *time.Time
}

// NewTask creates a new Task entity
func NewTask(
	id *valueobject.TaskID,
	title string,
	description string,
	priority valueobject.Priority,
	status valueobject.Status,
) (*Task, error) {
	if id == nil {
		return nil, ErrInvalidTaskID
	}
	if title == "" {
		return nil, ErrEmptyTaskName
	}
	if !priority.IsValid() {
		return nil, ErrInvalidPriority
	}
	if !status.IsValid() {
		return nil, ErrInvalidStatus
	}

	now := time.Now()
	return &Task{
		id:          id,
		title:       title,
		description: description,
		priority:    priority,
		status:      status,
		tags:        make([]string, 0),
		createdAt:   now,
		modifiedAt:  now,
	}, nil
}

// ID returns the task ID
func (t *Task) ID() *valueobject.TaskID {
	return t.id
}

// Title returns the task title
func (t *Task) Title() string {
	return t.title
}

// Description returns the task description
func (t *Task) Description() string {
	return t.description
}

// Priority returns the task priority
func (t *Task) Priority() valueobject.Priority {
	return t.priority
}

// Status returns the task status
func (t *Task) Status() valueobject.Status {
	return t.status
}

// Tags returns a copy of the task tags
func (t *Task) Tags() []string {
	tagsCopy := make([]string, len(t.tags))
	copy(tagsCopy, t.tags)
	return tagsCopy
}

// CreatedAt returns when the task was created
func (t *Task) CreatedAt() time.Time {
	return t.createdAt
}

// ModifiedAt returns when the task was last modified
func (t *Task) ModifiedAt() time.Time {
	return t.modifiedAt
}

// DueDate returns the task due date
func (t *Task) DueDate() *time.Time {
	if t.dueDate == nil {
		return nil
	}
	dueCopy := *t.dueDate
	return &dueCopy
}

// CompletedDate returns when the task was completed
func (t *Task) CompletedDate() *time.Time {
	if t.completedDate == nil {
		return nil
	}
	completedCopy := *t.completedDate
	return &completedCopy
}

// UpdateTitle updates the task title
func (t *Task) UpdateTitle(title string) error {
	if title == "" {
		return ErrEmptyTaskName
	}
	t.title = title
	t.modifiedAt = time.Now()
	return nil
}

// UpdateDescription updates the task description
func (t *Task) UpdateDescription(description string) {
	t.description = description
	t.modifiedAt = time.Now()
}

// UpdatePriority updates the task priority
func (t *Task) UpdatePriority(priority valueobject.Priority) error {
	if !priority.IsValid() {
		return ErrInvalidPriority
	}
	t.priority = priority
	t.modifiedAt = time.Now()
	return nil
}

// UpdateStatus updates the task status
func (t *Task) UpdateStatus(status valueobject.Status) error {
	if !status.IsValid() {
		return ErrInvalidStatus
	}
	t.status = status
	t.modifiedAt = time.Now()

	// Automatically set completed date when status changes to done
	if status == valueobject.StatusDone && t.completedDate == nil {
		now := time.Now()
		t.completedDate = &now
	}

	return nil
}

// SetDueDate sets the task due date
func (t *Task) SetDueDate(dueDate time.Time) error {
	if dueDate.Before(time.Now()) {
		return ErrDueDateInPast
	}
	t.dueDate = &dueDate
	t.modifiedAt = time.Now()
	return nil
}

// ClearDueDate removes the due date
func (t *Task) ClearDueDate() {
	t.dueDate = nil
	t.modifiedAt = time.Now()
}

// AddTag adds a tag to the task
func (t *Task) AddTag(tag string) {
	// Check if tag already exists
	for _, existingTag := range t.tags {
		if existingTag == tag {
			return
		}
	}
	t.tags = append(t.tags, tag)
	t.modifiedAt = time.Now()
}

// RemoveTag removes a tag from the task
func (t *Task) RemoveTag(tag string) {
	for i, existingTag := range t.tags {
		if existingTag == tag {
			t.tags = append(t.tags[:i], t.tags[i+1:]...)
			t.modifiedAt = time.Now()
			return
		}
	}
}

// MarkAsCompleted marks the task as completed
func (t *Task) MarkAsCompleted() error {
	if err := t.UpdateStatus(valueobject.StatusDone); err != nil {
		return err
	}
	if t.completedDate == nil {
		now := time.Now()
		t.completedDate = &now
	}
	return nil
}

// IsOverdue checks if the task is overdue
func (t *Task) IsOverdue() bool {
	if t.dueDate == nil || t.status == valueobject.StatusDone {
		return false
	}
	return t.dueDate.Before(time.Now())
}
