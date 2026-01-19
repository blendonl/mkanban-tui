package service

import (
	"fmt"
	"regexp"
	"strings"
)

// CheckboxState represents the state of a checkbox
type CheckboxState string

const (
	CheckboxTodo       CheckboxState = "[ ]"
	CheckboxInProgress CheckboxState = "[~]"
	CheckboxDone       CheckboxState = "[x]"
)

// SubtaskCheckbox represents a parsed checkbox item
type SubtaskCheckbox struct {
	OriginalLine string
	Title        string
	TaskID       string // Empty if not yet linked
	State        CheckboxState
}

// checkboxPattern matches markdown checkboxes: - [ ], - [x], - [~]
var checkboxPattern = regexp.MustCompile(`^(\s*)-\s+\[([ x~])\]\s+(.*)$`)

// linkedCheckboxPattern matches checkboxes with markdown links
var linkedCheckboxPattern = regexp.MustCompile(`^(\s*)-\s+\[([ x~])\]\s+\[([^\]]+)\]\(([^)]+)\)\s*(.*)$`)

// ParseSubtasks extracts subtask titles from description checkboxes
// Returns a list of subtask titles from lines matching "- [ ] {title}"
func ParseSubtasks(description string) []string {
	lines := strings.Split(description, "\n")
	subtasks := make([]string, 0)

	for _, line := range lines {
		matches := checkboxPattern.FindStringSubmatch(line)
		if matches != nil && len(matches) >= 4 {
			// matches[2] is the checkbox state, matches[3] is the content
			state := matches[2]
			content := strings.TrimSpace(matches[3])

			// Only extract unlinked checkboxes that are unchecked
			if state == " " && content != "" && !strings.Contains(content, "](") {
				subtasks = append(subtasks, content)
			}
		}
	}

	return subtasks
}

// UpdateCheckboxState updates a specific checkbox in the description
// It finds the checkbox linked to the given taskID and updates its state
func UpdateCheckboxState(description, taskID string, state CheckboxState) string {
	lines := strings.Split(description, "\n")
	updatedLines := make([]string, 0, len(lines))

	for _, line := range lines {
		matches := linkedCheckboxPattern.FindStringSubmatch(line)
		if matches != nil && len(matches) >= 5 {
			// matches[1] = leading whitespace
			// matches[2] = current state
			// matches[3] = link text
			// matches[4] = link URL
			// matches[5] = trailing text

			// Check if this checkbox links to the target taskID
			linkURL := matches[4]
			if strings.Contains(linkURL, taskID) {
				// Update the state
				indent := matches[1]
				linkText := matches[3]
				trailing := matches[5]
				updatedLine := fmt.Sprintf("%s- %s [%s](%s)%s", indent, state, linkText, linkURL, trailing)
				updatedLines = append(updatedLines, updatedLine)
				continue
			}
		}
		updatedLines = append(updatedLines, line)
	}

	return strings.Join(updatedLines, "\n")
}

// GetCheckboxStates returns a map of taskID -> state for all linked checkboxes
func GetCheckboxStates(description string) map[string]CheckboxState {
	lines := strings.Split(description, "\n")
	states := make(map[string]CheckboxState)

	for _, line := range lines {
		matches := linkedCheckboxPattern.FindStringSubmatch(line)
		if matches != nil && len(matches) >= 5 {
			state := matches[2]
			linkURL := matches[4]

			// Extract task ID from URL (e.g., "../../Todo/BOARD-2-title/task.md" -> "BOARD-2-title")
			parts := strings.Split(linkURL, "/")
			for _, part := range parts {
				if strings.Contains(part, "-") && len(part) > 3 {
					// This looks like a task ID
					var checkboxState CheckboxState
					switch state {
					case " ":
						checkboxState = CheckboxTodo
					case "x":
						checkboxState = CheckboxDone
					case "~":
						checkboxState = CheckboxInProgress
					default:
						checkboxState = CheckboxTodo
					}
					states[part] = checkboxState
					break
				}
			}
		}
	}

	return states
}

// AllCheckboxesComplete checks if all checkboxes in the description are marked as done
func AllCheckboxesComplete(description string) bool {
	states := GetCheckboxStates(description)

	// If there are no checkboxes, return false (not applicable)
	if len(states) == 0 {
		return false
	}

	// Check if all checkboxes are done
	for _, state := range states {
		if state != CheckboxDone {
			return false
		}
	}

	return true
}

// AddSubtaskLink converts a plain checkbox to a linked checkbox
// Replaces "- [ ] {title}" with "- [ ] [{title}]({taskLink})"
// The taskLink uses relative path format: ../../{columnName}/{taskID}/task.md
func AddSubtaskLink(description, subtaskTitle, taskID, columnName string) string {
	lines := strings.Split(description, "\n")
	updatedLines := make([]string, 0, len(lines))

	replaced := false
	for _, line := range lines {
		// Only replace the first matching unlinked checkbox with this title
		if !replaced {
			matches := checkboxPattern.FindStringSubmatch(line)
			if matches != nil && len(matches) >= 4 {
				state := matches[2]
				content := strings.TrimSpace(matches[3])
				indent := matches[1]

				// Check if this is the checkbox we want to replace
				if state == " " && content == subtaskTitle && !strings.Contains(content, "](") {
					// Create the linked version with relative path
					taskLink := fmt.Sprintf("../../%s/%s/task.md", columnName, taskID)
					updatedLine := fmt.Sprintf("%s- [ ] [%s](%s)", indent, subtaskTitle, taskLink)
					updatedLines = append(updatedLines, updatedLine)
					replaced = true
					continue
				}
			}
		}
		updatedLines = append(updatedLines, line)
	}

	return strings.Join(updatedLines, "\n")
}
