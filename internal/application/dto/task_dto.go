package dto

import "time"

// TaskDTO represents a task data transfer object
type TaskDTO struct {
	ID            string     `json:"id"`
	ShortID       string     `json:"short_id"`
	ProjectID     string     `json:"project_id,omitempty"`
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
	FilePath      string     `json:"file_path,omitempty"`
	ColumnName    string     `json:"column_name,omitempty"`

	EstimatedTime *time.Duration `json:"estimated_time,omitempty"`
	TrackedTime   time.Duration  `json:"tracked_time,omitempty"`
	LinkedNotes   []string       `json:"linked_notes,omitempty"`

	ScheduledDate *time.Time     `json:"scheduled_date,omitempty"`
	ScheduledTime *time.Time     `json:"scheduled_time,omitempty"`
	TimeBlock     *time.Duration `json:"time_block,omitempty"`

	TaskType    string       `json:"task_type,omitempty"`
	MeetingData *MeetingDTO  `json:"meeting_data,omitempty"`
}

type MeetingDTO struct {
	Attendees     []string `json:"attendees,omitempty"`
	Location      string   `json:"location,omitempty"`
	MeetingURL    string   `json:"meeting_url,omitempty"`
	GoogleEventID string   `json:"google_event_id,omitempty"`
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
