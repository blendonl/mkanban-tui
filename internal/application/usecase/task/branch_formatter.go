package task

import (
	"mkanban/internal/domain/valueobject"
	"mkanban/pkg/slug"
	"strings"
)

// FormatBranchName formats a branch name using a template and task information
// Supported placeholders:
//   - {id}: Full task ID (e.g., "FOR-001-test-task")
//   - {short-id}: Short task ID (e.g., "FOR-001")
//   - {title}: Task title in kebab-case (e.g., "test-task")
//
// Example formats:
//   - "feat/{id}" -> "feat/FOR-001-test-task"
//   - "{short-id}-{title}" -> "FOR-001-test-task"
//   - "feature/{title}" -> "feature/test-task"
func FormatBranchName(format string, taskID *valueobject.TaskID, title string) string {
	// If format is empty, default to just the task ID
	if format == "" {
		format = "{id}"
	}

	// Convert title to kebab-case
	titleKebab := slug.Generate(title)

	// Replace placeholders
	result := format
	result = strings.ReplaceAll(result, "{id}", taskID.String())
	result = strings.ReplaceAll(result, "{short-id}", taskID.ShortID())
	result = strings.ReplaceAll(result, "{title}", titleKebab)

	return result
}
