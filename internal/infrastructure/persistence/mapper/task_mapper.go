package mapper

import (
	"fmt"
	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/valueobject"
	"mkanban/internal/infrastructure/serialization"
	"time"
)

// TaskStorage represents task storage format
type TaskStorage struct {
	ID            string     `yaml:"id"`
	Title         string     `yaml:"title"`
	Created       time.Time  `yaml:"created"`
	Modified      time.Time  `yaml:"modified"`
	DueDate       *time.Time `yaml:"due_date,omitempty"`
	CompletedDate *time.Time `yaml:"completed_date,omitempty"`
	Priority      string     `yaml:"priority"`
	Status        string     `yaml:"status"`
	Tags          []string   `yaml:"tags,omitempty"`
}

// TaskToStorage converts a Task entity to storage format
func TaskToStorage(task *entity.Task) (map[string]interface{}, string, error) {
	storage := TaskStorage{
		ID:            task.ID().ShortID(), // Store only PREFIX-NUMBER in metadata
		Title:         task.Title(),
		Created:       task.CreatedAt(),
		Modified:      task.ModifiedAt(),
		DueDate:       task.DueDate(),
		CompletedDate: task.CompletedDate(),
		Priority:      task.Priority().String(),
		Status:        task.Status().String(),
		Tags:          task.Tags(),
	}

	frontmatter := map[string]interface{}{
		"id":       storage.ID,
		"title":    storage.Title,
		"created":  storage.Created.Format(time.RFC3339),
		"modified": storage.Modified.Format(time.RFC3339),
		"priority": storage.Priority,
		"status":   storage.Status,
	}

	if storage.DueDate != nil {
		frontmatter["due_date"] = storage.DueDate.Format(time.RFC3339)
	}

	if storage.CompletedDate != nil {
		frontmatter["completed_date"] = storage.CompletedDate.Format(time.RFC3339)
	}

	if len(storage.Tags) > 0 {
		frontmatter["tags"] = storage.Tags
	}

	content := task.Description()

	return frontmatter, content, nil
}

// TaskFromStorage converts storage format to Task entity
// The taskID parameter comes from the folder name (PREFIX-NUMBER-slug format)
// while the metadata contains only the short ID (PREFIX-NUMBER)
func TaskFromStorage(doc *serialization.FrontmatterDocument, taskID *valueobject.TaskID) (*entity.Task, error) {
	// Validate that the short ID from metadata matches the provided taskID
	shortID := doc.GetString("id")
	if shortID != "" && shortID != taskID.ShortID() {
		return nil, fmt.Errorf("task ID mismatch: metadata has %s but folder indicates %s", shortID, taskID.ShortID())
	}

	// Get title from metadata
	title := doc.GetString("title")
	if title == "" {
		return nil, fmt.Errorf("missing task title")
	}

	// Parse priority
	priorityStr := doc.GetString("priority")
	if priorityStr == "" {
		priorityStr = "none"
	}
	priority, err := valueobject.ParsePriority(priorityStr)
	if err != nil {
		return nil, fmt.Errorf("invalid priority: %w", err)
	}

	// Parse status
	statusStr := doc.GetString("status")
	if statusStr == "" {
		statusStr = "todo"
	}
	status, err := valueobject.ParseStatus(statusStr)
	if err != nil {
		return nil, fmt.Errorf("invalid status: %w", err)
	}

	// Create task
	task, err := entity.NewTask(taskID, title, doc.Content, priority, status)
	if err != nil {
		return nil, err
	}

	// Parse dates (using reflection-like approach through frontmatter)
	if createdStr := doc.GetString("created"); createdStr != "" {
		// Already set in NewTask, but we could override if needed
	}

	// Parse optional dates
	if dueDateStr := doc.GetString("due_date"); dueDateStr != "" {
		dueDate, err := time.Parse(time.RFC3339, dueDateStr)
		if err == nil {
			_ = task.SetDueDate(dueDate)
		}
	}

	// Parse tags
	tags := doc.GetStringSlice("tags")
	for _, tag := range tags {
		task.AddTag(tag)
	}

	return task, nil
}
