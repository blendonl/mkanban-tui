package entity

import (
	"mkanban/internal/domain/valueobject"
	"time"
)

// Column represents a board column containing tasks
type Column struct {
	name        string
	description string
	order       int
	wipLimit    int
	color       *valueobject.Color
	tasks       []*Task
	createdAt   time.Time
	modifiedAt  time.Time
}

// NewColumn creates a new Column entity
func NewColumn(name string, description string, order int, wipLimit int, color *valueobject.Color) (*Column, error) {
	if name == "" {
		return nil, ErrEmptyColumnName
	}
	if wipLimit < 0 {
		return nil, ErrInvalidWIPLimit
	}

	now := time.Now()
	return &Column{
		name:        name,
		description: description,
		order:       order,
		wipLimit:    wipLimit,
		color:       color,
		tasks:       make([]*Task, 0),
		createdAt:   now,
		modifiedAt:  now,
	}, nil
}

// Name returns the column name
func (c *Column) Name() string {
	return c.name
}

// Description returns the column description
func (c *Column) Description() string {
	return c.description
}

// Order returns the column order
func (c *Column) Order() int {
	return c.order
}

// WIPLimit returns the work-in-progress limit
func (c *Column) WIPLimit() int {
	return c.wipLimit
}

// Color returns the column color
func (c *Column) Color() *valueobject.Color {
	return c.color
}

// Tasks returns a copy of the tasks
func (c *Column) Tasks() []*Task {
	tasksCopy := make([]*Task, len(c.tasks))
	copy(tasksCopy, c.tasks)
	return tasksCopy
}

// TaskCount returns the number of tasks in the column
func (c *Column) TaskCount() int {
	return len(c.tasks)
}

// CreatedAt returns when the column was created
func (c *Column) CreatedAt() time.Time {
	return c.createdAt
}

// ModifiedAt returns when the column was last modified
func (c *Column) ModifiedAt() time.Time {
	return c.modifiedAt
}

// UpdateName updates the column name
func (c *Column) UpdateName(name string) error {
	if name == "" {
		return ErrEmptyColumnName
	}
	c.name = name
	c.modifiedAt = time.Now()
	return nil
}

// UpdateDescription updates the column description
func (c *Column) UpdateDescription(description string) {
	c.description = description
	c.modifiedAt = time.Now()
}

// UpdateOrder updates the column order
func (c *Column) UpdateOrder(order int) {
	c.order = order
	c.modifiedAt = time.Now()
}

// UpdateWIPLimit updates the work-in-progress limit
func (c *Column) UpdateWIPLimit(limit int) error {
	if limit < 0 {
		return ErrInvalidWIPLimit
	}
	c.wipLimit = limit
	c.modifiedAt = time.Now()
	return nil
}

// UpdateColor updates the column color
func (c *Column) UpdateColor(color *valueobject.Color) {
	c.color = color
	c.modifiedAt = time.Now()
}

// AddTask adds a task to the column
func (c *Column) AddTask(task *Task) error {
	if task == nil {
		return ErrTaskNotFound
	}

	// Check WIP limit (0 means unlimited)
	if c.wipLimit > 0 && len(c.tasks) >= c.wipLimit {
		return ErrWIPLimitExceeded
	}

	// Check if task already exists
	for _, existingTask := range c.tasks {
		if existingTask.ID().Equal(task.ID()) {
			return ErrTaskAlreadyExists
		}
	}

	c.tasks = append(c.tasks, task)
	c.modifiedAt = time.Now()
	return nil
}

// RemoveTask removes a task from the column
func (c *Column) RemoveTask(taskID *valueobject.TaskID) (*Task, error) {
	for i, task := range c.tasks {
		if task.ID().Equal(taskID) {
			// Remove task from slice
			c.tasks = append(c.tasks[:i], c.tasks[i+1:]...)
			c.modifiedAt = time.Now()
			return task, nil
		}
	}
	return nil, ErrTaskNotFound
}

// GetTask retrieves a task by ID
func (c *Column) GetTask(taskID *valueobject.TaskID) (*Task, error) {
	for _, task := range c.tasks {
		if task.ID().Equal(taskID) {
			return task, nil
		}
	}
	return nil, ErrTaskNotFound
}

// HasTask checks if the column contains a specific task
func (c *Column) HasTask(taskID *valueobject.TaskID) bool {
	_, err := c.GetTask(taskID)
	return err == nil
}

// CanAddTask checks if a task can be added without violating WIP limit
func (c *Column) CanAddTask() bool {
	if c.wipLimit == 0 {
		return true // Unlimited
	}
	return len(c.tasks) < c.wipLimit
}

// IsWIPLimitReached checks if the WIP limit is reached
func (c *Column) IsWIPLimitReached() bool {
	if c.wipLimit == 0 {
		return false // Unlimited
	}
	return len(c.tasks) >= c.wipLimit
}
