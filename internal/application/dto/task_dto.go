package dto

import "time"

// TaskDTO represents a task data transfer object
type TaskDTO struct {
	ID            string     `json:"id"`
	ShortID       string     `json:"short_id"`
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	Priority      string     `json:"priority"`
	Status        string     `json:"status"`
	Tags          []string   `json:"tags"`
	CreatedAt     time.Time  `json:"created_at"`
	ModifiedAt    time.Time  `json:"modified_at"`
	DueDate       *time.Time `json:"due_date,omitempty"`
	CompletedDate *time.Time `json:"completed_date,omitempty"`
	IsOverdue     bool       `json:"is_overdue"`
	FilePath      string     `json:"file_path,omitempty"` // Optional: path to task.md file
	ColumnName    string     `json:"column_name,omitempty"` // Optional: name of the column containing the task
}

// CreateTaskRequest represents a request to create a task
type CreateTaskRequest struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Priority    string    `json:"priority"`
	ColumnName  string    `json:"column_name"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
}

// UpdateTaskRequest represents a request to update a task
type UpdateTaskRequest struct {
	Title       *string   `json:"title,omitempty"`
	Description *string   `json:"description,omitempty"`
	Priority    *string   `json:"priority,omitempty"`
	Status      *string   `json:"status,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
}

// MoveTaskRequest represents a request to move a task
type MoveTaskRequest struct {
	TaskID           string `json:"task_id"`
	TargetColumnName string `json:"target_column_name"`
}
