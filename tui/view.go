package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"mkanban/internal/application/dto"
	"mkanban/tui/style"
)

// View renders the UI
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Calculate column width
	columnWidth := (m.width / len(m.board.Columns)) - 4

	// Render columns
	var columns []string
	for i, col := range m.board.Columns {
		columns = append(columns, m.renderColumn(col, i, columnWidth))
	}

	// Join columns horizontally
	board := lipgloss.JoinHorizontal(lipgloss.Top, columns...)

	// Render help text
	help := m.renderHelp()

	return lipgloss.JoinVertical(lipgloss.Left, board, help)
}

// renderColumn renders a single column
func (m Model) renderColumn(col dto.ColumnDTO, colIndex int, width int) string {
	// Determine if this column is focused
	isFocused := colIndex == m.focusedColumn

	// Column title
	title := style.ColumnTitleStyle.Width(width).Render(col.Name)

	// Tasks
	var tasks []string
	for i, task := range col.Tasks {
		taskText := task.Title
		if len(taskText) > width-4 {
			taskText = taskText[:width-4] + "..."
		}

		// Highlight selected task in focused column
		if isFocused && i == m.focusedTask {
			tasks = append(tasks, style.SelectedTaskStyle.Width(width).Render(taskText))
		} else {
			tasks = append(tasks, style.TaskStyle.Width(width).Render(taskText))
		}
	}

	// Add placeholder if no tasks
	if len(tasks) == 0 {
		tasks = append(tasks, style.TaskStyle.Width(width).Foreground(lipgloss.Color("240")).Render("(empty)"))
	}

	// Join title and tasks
	content := lipgloss.JoinVertical(lipgloss.Left, title, "", strings.Join(tasks, "\n"))

	// Apply column style
	if isFocused {
		return style.FocusedColumnStyle.Width(width).Height(m.height - 6).Render(content)
	}
	return style.ColumnStyle.Width(width).Height(m.height - 6).Render(content)
}

// renderHelp renders the help text at the bottom
func (m Model) renderHelp() string {
	helpText := []string{
		"Navigation: ←/h,→/l (columns)  ↑/k,↓/j (tasks)",
		"Actions: a (add)  d (delete)  m/enter (move)  q (quit)",
	}

	return style.HelpStyle.Render(strings.Join(helpText, "  •  "))
}

// statusMessage for debugging (optional)
func (m Model) statusMessage() string {
	return fmt.Sprintf("Column: %d/%d | Task: %d/%d | Size: %dx%d",
		m.focusedColumn+1, len(m.board.Columns),
		m.focusedTask+1, m.currentColumnTaskCount(),
		m.width, m.height)
}
