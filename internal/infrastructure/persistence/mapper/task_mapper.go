package mapper

import (
	"fmt"
	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/valueobject"
	"mkanban/internal/infrastructure/serialization"
	"time"
)

// GitMetadata represents git-related task metadata
type GitMetadata struct {
	Branch          string `yaml:"branch,omitempty"`
	IsCurrentBranch string `yaml:"is_current_branch,omitempty"`
}

// TaskStorage represents task storage format
type TaskStorage struct {
	ID            string         `yaml:"id"`
	ParentID      string         `yaml:"parent_id,omitempty"`
	Created       time.Time      `yaml:"created"`
	Modified      time.Time      `yaml:"modified"`
	DueDate       *time.Time     `yaml:"due_date,omitempty"`
	CompletedDate *time.Time     `yaml:"completed_date,omitempty"`
	Priority      string         `yaml:"priority"`
	Status        string         `yaml:"status"`
	Tags          []string       `yaml:"tags,omitempty"`
	Git           *GitMetadata   `yaml:"git,omitempty"`
	ScheduledDate *time.Time     `yaml:"scheduled_date,omitempty"`
	ScheduledTime *time.Time     `yaml:"scheduled_time,omitempty"`
	TimeBlock     *time.Duration `yaml:"time_block,omitempty"`
	TaskType      string         `yaml:"task_type,omitempty"`
}

// TaskToStorage converts a Task entity to storage format
// Returns: metadata (for metadata.yml), markdownContent (for task.md), error
func TaskToStorage(task *entity.Task) (*TaskStorage, []byte, error) {
	storage := &TaskStorage{
		ID:            task.ID().ShortID(), // Store only PREFIX-NUMBER in metadata
		Created:       task.CreatedAt(),
		Modified:      task.ModifiedAt(),
		DueDate:       task.DueDate(),
		CompletedDate: task.CompletedDate(),
		Priority:      task.Priority().String(),
		Status:        task.Status().String(),
		Tags:          task.Tags(),
		ScheduledDate: task.ScheduledDate(),
		ScheduledTime: task.ScheduledTime(),
		TimeBlock:     task.TimeBlock(),
	}

	if task.TaskType() != entity.TaskTypeRegular {
		storage.TaskType = string(task.TaskType())
	}

	// Store parent ID if this is a subtask
	if task.ParentID() != nil {
		storage.ParentID = task.ParentID().ShortID()
	}

	// Extract git metadata if present
	gitBranch, hasGitBranch := task.GetMetadata("git_branch")
	isCurrentBranch, hasIsCurrentBranch := task.GetMetadata("is_current_branch")

	if hasGitBranch || hasIsCurrentBranch {
		storage.Git = &GitMetadata{
			Branch:          gitBranch,
			IsCurrentBranch: isCurrentBranch,
		}
	}

	// Serialize markdown with title and description
	markdownContent := serialization.SerializeMarkdownWithTitle(task.Title(), task.Description())

	return storage, markdownContent, nil
}

// TaskFromStorage converts storage format to Task entity
// The taskID parameter comes from the folder name (PREFIX-NUMBER-slug format)
// while the metadata contains only the short ID (PREFIX-NUMBER)
// metadata is the parsed TaskStorage struct, markdownContent is the raw markdown file
func TaskFromStorage(metadata *TaskStorage, markdownContent []byte, taskID *valueobject.TaskID) (*entity.Task, error) {
	// Validate that the short ID from metadata matches the provided taskID
	if metadata.ID != "" && metadata.ID != taskID.ShortID() {
		return nil, fmt.Errorf("task ID mismatch: metadata has %s but folder indicates %s", metadata.ID, taskID.ShortID())
	}

	// Parse markdown to get title and description
	mdDoc, err := serialization.ParseMarkdownWithTitle(markdownContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse markdown: %w", err)
	}

	if mdDoc.Title == "" {
		return nil, fmt.Errorf("missing task title in markdown")
	}

	// Parse priority
	priorityStr := metadata.Priority
	if priorityStr == "" {
		priorityStr = "none"
	}
	priority, err := valueobject.ParsePriority(priorityStr)
	if err != nil {
		return nil, fmt.Errorf("invalid priority: %w", err)
	}

	// Parse status
	statusStr := metadata.Status
	if statusStr == "" {
		statusStr = "todo"
	}
	status, err := valueobject.ParseStatus(statusStr)
	if err != nil {
		return nil, fmt.Errorf("invalid status: %w", err)
	}

	// Create task
	task, err := entity.NewTask(taskID, mdDoc.Title, mdDoc.Content, priority, status)
	if err != nil {
		return nil, err
	}

	// Parse optional dates
	if metadata.DueDate != nil {
		_ = task.SetDueDate(*metadata.DueDate)
	}
	if metadata.ScheduledDate != nil {
		task.SetScheduledDate(*metadata.ScheduledDate)
	}
	if metadata.ScheduledTime != nil {
		task.SetScheduledTime(*metadata.ScheduledTime)
	}
	if metadata.TimeBlock != nil {
		task.SetTimeBlock(*metadata.TimeBlock)
	}
	if metadata.TaskType != "" {
		task.SetTaskType(entity.TaskType(metadata.TaskType))
	}

	// Parse tags
	for _, tag := range metadata.Tags {
		task.AddTag(tag)
	}

	// Parse parent ID if present
	if metadata.ParentID != "" {
		parentID, err := valueobject.ParseTaskID(metadata.ParentID)
		if err == nil {
			task.SetParentID(parentID)
		}
	}

	// Parse git metadata if present
	if metadata.Git != nil {
		if metadata.Git.Branch != "" {
			task.SetMetadata("git_branch", metadata.Git.Branch)
		}
		if metadata.Git.IsCurrentBranch != "" {
			task.SetMetadata("is_current_branch", metadata.Git.IsCurrentBranch)
		}
	}

	return task, nil
}
