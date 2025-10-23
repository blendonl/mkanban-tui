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

	// Calculate column width - account for borders, padding, and spacing
	numColumns := len(m.board.Columns)
	if numColumns == 0 {
		return "No columns"
	}
	// Each column has 2 border chars + 2 padding = 4 chars overhead per column
	// Add some margin between columns
	totalOverhead := numColumns * 6 // 4 for border+padding, 2 for margin
	availableWidth := m.width - totalOverhead
	if availableWidth < numColumns*20 {
		// Ensure minimum width per column
		availableWidth = numColumns * 20
	}
	columnWidth := availableWidth / numColumns

	// Calculate available height for task content in columns
	// Subtract: help (3 lines), column title (1 line), spacing (2 lines), borders (2 lines)
	availableTaskHeight := m.height - 8

	// Render columns
	var columns []string
	for i, col := range m.board.Columns {
		columns = append(columns, m.renderColumn(col, i, columnWidth, availableTaskHeight))
	}

	// Join columns horizontally
	board := lipgloss.JoinHorizontal(lipgloss.Top, columns...)

	// Render help text
	help := m.renderHelp()

	return lipgloss.JoinVertical(lipgloss.Left, board, help)
}

// renderColumn renders a single column with scrolling support
func (m Model) renderColumn(col dto.ColumnDTO, colIndex int, width int, viewportHeight int) string {
	// Determine if this column is focused
	isFocused := colIndex == m.focusedColumn

	// Column title
	title := style.ColumnTitleStyle.Width(width).Render(col.Name)

	// Get scroll offset for this column
	scrollOffset := 0
	if colIndex < len(m.scrollOffsets) {
		scrollOffset = m.scrollOffsets[colIndex]
	}

	// Estimate how many tasks can fit (rough estimate: ~6 lines per task card)
	maxVisibleTasks := viewportHeight / 6
	if maxVisibleTasks < 1 {
		maxVisibleTasks = 1
	}

	// Calculate visible range
	totalTasks := len(col.Tasks)
	startIdx := scrollOffset
	endIdx := scrollOffset + maxVisibleTasks
	if endIdx > totalTasks {
		endIdx = totalTasks
	}

	// Show scroll indicators
	showUpIndicator := scrollOffset > 0
	showDownIndicator := endIdx < totalTasks

	// Tasks - render visible cards only
	var tasks []string

	// Add up scroll indicator
	if showUpIndicator {
		indicator := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true).
			Width(width).
			Align(lipgloss.Center).
			Render("▲ more above ▲")
		tasks = append(tasks, indicator)
	}

	// Render visible tasks
	for i := startIdx; i < endIdx; i++ {
		task := col.Tasks[i]
		// Determine if this task is selected
		isSelected := isFocused && i == m.focusedTask

		// Render task card with all components
		taskCard := renderTaskCard(task, width, isSelected)
		tasks = append(tasks, taskCard)
	}

	// Add down scroll indicator
	if showDownIndicator {
		indicator := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true).
			Width(width).
			Align(lipgloss.Center).
			Render("▼ more below ▼")
		tasks = append(tasks, indicator)
	}

	// Add placeholder if no tasks
	if len(col.Tasks) == 0 {
		tasks = append(tasks, style.TaskStyle.Width(width).Foreground(lipgloss.Color("240")).Render("(empty)"))
	}

	// Join title and tasks with spacing
	content := lipgloss.JoinVertical(lipgloss.Left, title, "", strings.Join(tasks, "\n"))

	// Apply column style (don't set width here - it's already set on content)
	if isFocused {
		return style.FocusedColumnStyle.Height(m.height - 6).Render(content)
	}
	return style.ColumnStyle.Height(m.height - 6).Render(content)
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
