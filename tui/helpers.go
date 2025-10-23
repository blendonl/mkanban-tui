package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"mkanban/internal/application/dto"
	"mkanban/internal/infrastructure/config"
	"mkanban/tui/style"
)

// getPriorityIcon returns an icon and color based on priority level
func getPriorityIcon(priority string) string {
	switch strings.ToLower(priority) {
	case "high", "medium", "low":
		return "⚫"
	default:
		return "⚪"
	}
}

// getPriorityColor returns a color based on priority level
func getPriorityColor(priority string, cfg *config.Config) lipgloss.Color {
	priorityColors := cfg.TUI.Styles.Priority
	switch strings.ToLower(priority) {
	case "high":
		return lipgloss.Color(priorityColors.High)
	case "medium":
		return lipgloss.Color(priorityColors.Medium)
	case "low":
		return lipgloss.Color(priorityColors.Low)
	default:
		return lipgloss.Color(priorityColors.Default)
	}
}

// formatDueDate formats the due date with relative time and returns the formatted string and style color
func formatDueDate(dueDate *time.Time, isOverdue bool, cfg *config.Config) (string, lipgloss.Color) {
	if dueDate == nil {
		return "", lipgloss.Color("")
	}

	dueDateColors := cfg.TUI.Styles.DueDateUrgency
	now := time.Now()
	diff := dueDate.Sub(now)

	var relativeTime string
	var color lipgloss.Color
	var prefix string

	if isOverdue {
		days := int(-diff.Hours() / 24)
		if days == 0 {
			relativeTime = "overdue today"
		} else if days == 1 {
			relativeTime = "overdue 1 day"
		} else {
			relativeTime = fmt.Sprintf("overdue %d days", days)
		}
		color = lipgloss.Color(dueDateColors.Overdue)
		prefix = "⚠️ "
	} else {
		days := int(diff.Hours() / 24)

		if days == 0 {
			relativeTime = "due today"
			color = lipgloss.Color(dueDateColors.DueSoon)
		} else if days == 1 {
			relativeTime = "due tomorrow"
			color = lipgloss.Color(dueDateColors.DueSoon)
		} else if days <= 3 {
			relativeTime = fmt.Sprintf("due in %d days", days)
			color = lipgloss.Color(dueDateColors.DueSoon)
		} else if days <= 7 {
			relativeTime = fmt.Sprintf("due in %d days", days)
			color = lipgloss.Color(dueDateColors.Upcoming)
		} else {
			relativeTime = fmt.Sprintf("due in %d days", days)
			color = lipgloss.Color(dueDateColors.FarFuture)
		}
		prefix = ""
	}

	dateStr := dueDate.Format("Jan 02")
	return fmt.Sprintf("%s📅 %s (%s)", prefix, dateStr, relativeTime), color
}

// formatTags formats tags with icon and handles truncation
func formatTags(tags []string, maxWidth int) string {
	if len(tags) == 0 {
		return ""
	}

	prefix := "🏷️  "
	availableWidth := maxWidth - len(prefix)

	if availableWidth <= 10 {
		// Not enough space to show tags meaningfully
		return ""
	}

	var displayTags []string
	currentLength := 0

	for i, tag := range tags {
		tagLen := len(tag) + 2 // +2 for spacing
		if currentLength+tagLen > availableWidth {
			if i == 0 {
				// At least show part of the first tag
				truncated := tag[:min(len(tag), availableWidth-3)] + "..."
				displayTags = append(displayTags, truncated)
			} else {
				// Show how many more tags there are
				remaining := len(tags) - i
				displayTags = append(displayTags, fmt.Sprintf("+%d", remaining))
			}
			break
		}
		displayTags = append(displayTags, tag)
		currentLength += tagLen
	}

	return prefix + strings.Join(displayTags, "  ")
}

// truncateDescription extracts and truncates the description preview
func truncateDescription(desc string, maxLen int) string {
	if desc == "" {
		return ""
	}

	// Get first line only
	lines := strings.Split(desc, "\n")
	firstLine := strings.TrimSpace(lines[0])

	if firstLine == "" && len(lines) > 1 {
		// Try second line if first is empty
		firstLine = strings.TrimSpace(lines[1])
	}

	if len(firstLine) > maxLen {
		return firstLine[:maxLen-3] + "..."
	}

	return firstLine
}

// renderTaskCard renders a complete task card with all components
func renderTaskCard(task dto.TaskDTO, width int, isSelected bool, cfg *config.Config) string {
	var lines []string

	// Calculate content width (account for padding and borders)
	contentWidth := width - 4
	if contentWidth < 20 {
		contentWidth = 20
	}

	// Line 1: Priority icon + Title
	priorityIcon := getPriorityIcon(task.Priority)
	titleMaxWidth := contentWidth - 3 // -3 for icon and spaces
	title := task.Title
	if len(title) > titleMaxWidth {
		title = title[:titleMaxWidth-3] + "..."
	}

	priorityColor := getPriorityColor(task.Priority, cfg)
	titleLine := lipgloss.NewStyle().
		Foreground(priorityColor).
		Bold(true).
		Render(priorityIcon) + " " +
		lipgloss.NewStyle().Bold(true).Render(title)

	lines = append(lines, titleLine)

	// Line 2: Description (if exists)
	if task.Description != "" {
		descMaxLen := contentWidth
		desc := truncateDescription(task.Description, descMaxLen)
		if desc != "" {
			descLine := style.DescriptionStyle.
				Width(contentWidth).
				Render(desc)
			lines = append(lines, descLine)
		}
	}

	// Line 3: Tags (if exist)
	if len(task.Tags) > 0 {
		tagsStr := formatTags(task.Tags, contentWidth)
		if tagsStr != "" {
			tagLine := style.TagStyle.
				Width(contentWidth).
				Render(tagsStr)
			lines = append(lines, tagLine)
		}
	}

	// Line 4: Due date (if exists)
	if task.DueDate != nil {
		dueDateStr, dueDateColor := formatDueDate(task.DueDate, task.IsOverdue, cfg)
		if dueDateStr != "" {
			dueDateStyle := style.DueDateStyle.Foreground(dueDateColor)
			if task.IsOverdue {
				dueDateStyle = style.OverdueStyle
			}
			dueLine := dueDateStyle.
				Width(contentWidth).
				Render(dueDateStr)
			lines = append(lines, dueLine)
		}
	}

	// Join all lines with small spacing
	cardContent := strings.Join(lines, "\n")

	// Apply card container style with explicit width to ensure background fills properly
	if isSelected {
		return style.SelectedTaskCardStyle.Width(width).Render(cardContent)
	}

	return style.TaskCardStyle.Width(width).Render(cardContent)
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
