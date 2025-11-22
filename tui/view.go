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
	totalColumns := len(m.board.Columns)
	if totalColumns == 0 {
		return "No columns"
	}

	// Column width constraints (content width, not including borders/padding)
	const minColumnWidth = 25   // Minimum content width
	const maxColumnWidth = 60   // Maximum content width
	const columnSpacing = 2     // Space between columns

	// Column overhead: borders (2 chars) + horizontal padding (2*2 = 4 chars)
	const columnOverhead = 6

	// Reserve space for scroll indicators when needed
	indicatorWidth := 5
	availableWidthForColumns := m.width - (indicatorWidth * 2)

	// Calculate optimal column width to fit all columns if possible
	// Total space needed = (renderedColumnWidth × totalColumns) + (spacing × (totalColumns - 1))
	// Where renderedColumnWidth = columnWidth + columnOverhead
	var columnWidth int
	spacingNeeded := columnSpacing * (totalColumns - 1)
	minRenderedWidth := minColumnWidth + columnOverhead
	minSpaceNeeded := (minRenderedWidth * totalColumns) + spacingNeeded

	if availableWidthForColumns >= minSpaceNeeded {
		// All columns can fit - calculate width to distribute remaining space
		spaceForColumns := availableWidthForColumns - spacingNeeded
		proposedRenderedWidth := spaceForColumns / totalColumns

		// Convert to content width by removing overhead
		proposedWidth := proposedRenderedWidth - columnOverhead

		if proposedWidth > maxColumnWidth {
			columnWidth = maxColumnWidth
		} else if proposedWidth < minColumnWidth {
			columnWidth = minColumnWidth
		} else {
			columnWidth = proposedWidth
		}
	} else {
		// Not all columns can fit - use minimum width and enable scrolling
		columnWidth = minColumnWidth
	}

	// Calculate how many columns can fit with the determined width
	// For N columns: (renderedColumnWidth × N) + (spacing × (N-1)) ≤ availableWidth
	renderedColumnWidth := columnWidth + columnOverhead
	maxVisibleColumns := (availableWidthForColumns + columnSpacing) / (renderedColumnWidth + columnSpacing)
	if maxVisibleColumns < 1 {
		maxVisibleColumns = 1
	}
	if maxVisibleColumns > totalColumns {
		maxVisibleColumns = totalColumns
	}

	// Calculate visible range based on current horizontal scroll
	startCol := m.horizontalScrollOffset
	endCol := startCol + maxVisibleColumns
	if endCol > totalColumns {
		endCol = totalColumns
	}

	// Show scroll indicators
	showLeftIndicator := startCol > 0
	showRightIndicator := endCol < totalColumns

	// Calculate available height for task content in columns
	// Subtract: help (3 lines), column title (1 line), spacing (2 lines), borders (2 lines)
	availableTaskHeight := m.height - 8

	// Render visible columns
	var columnsToRender []string

	// Add left scroll indicator if needed
	if showLeftIndicator {
		indicator := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Bold(true).
			Height(m.height - 6).
			Width(indicatorWidth).
			AlignVertical(lipgloss.Center).
			Render("◄")
		columnsToRender = append(columnsToRender, indicator)
	}

	// Render only visible columns
	for i := startCol; i < endCol; i++ {
		col := m.board.Columns[i]
		columnsToRender = append(columnsToRender, m.renderColumn(col, i, columnWidth, availableTaskHeight))
	}

	// Add right scroll indicator if needed
	if showRightIndicator {
		indicator := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Bold(true).
			Height(m.height - 6).
			Width(indicatorWidth).
			AlignVertical(lipgloss.Center).
			Render("►")
		columnsToRender = append(columnsToRender, indicator)
	}

	// Join columns horizontally
	board := lipgloss.JoinHorizontal(lipgloss.Top, columnsToRender...)

	// Center the board if there's extra space on the right
	// Calculate actual width used by visible columns
	numVisibleColumns := endCol - startCol
	totalColumnWidth := (renderedColumnWidth * numVisibleColumns) + (columnSpacing * (numVisibleColumns - 1))

	// Add indicator widths if they're shown
	if showLeftIndicator {
		totalColumnWidth += indicatorWidth
	}
	if showRightIndicator {
		totalColumnWidth += indicatorWidth
	}

	// Calculate left margin for centering
	leftMargin := 0
	if totalColumnWidth < m.width {
		remainingSpace := m.width - totalColumnWidth
		leftMargin = remainingSpace / 2
	}

	// Apply left margin using lipgloss style to all content
	if leftMargin > 0 {
		marginStyle := lipgloss.NewStyle().PaddingLeft(leftMargin)
		board = marginStyle.Render(board)
	}

	// Render help text with scroll info
	help := m.renderHelp()
	if totalColumns > maxVisibleColumns {
		scrollInfo := fmt.Sprintf("Columns: %d-%d of %d", startCol+1, endCol, totalColumns)
		help = help + "\n" + style.HelpStyle.Faint(true).Render(scrollInfo)
	}

	// Apply same left margin to help text
	if leftMargin > 0 {
		marginStyle := lipgloss.NewStyle().PaddingLeft(leftMargin)
		help = marginStyle.Render(help)
	}

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
		indicator := style.ScrollIndicatorStyle.
			Width(width).
			Render("▲ more above ▲")
		tasks = append(tasks, indicator)
	}

	// Render visible tasks
	for i := startIdx; i < endIdx; i++ {
		task := col.Tasks[i]
		// Determine if this task is selected
		isSelected := isFocused && i == m.focusedTask

		// Render task card with all components
		taskCard := renderTaskCard(task, width, isSelected, m.config)
		tasks = append(tasks, taskCard)
	}

	// Add down scroll indicator
	if showDownIndicator {
		indicator := style.ScrollIndicatorStyle.
			Width(width).
			Render("▼ more below ▼")
		tasks = append(tasks, indicator)
	}

	// Add placeholder if no tasks
	if len(col.Tasks) == 0 {
		emptyStyle := style.TaskStyle.Width(width)
		if m.config.TUI.Styles.ScrollIndicator.Foreground != "" {
			emptyStyle = emptyStyle.Foreground(lipgloss.Color(m.config.TUI.Styles.ScrollIndicator.Foreground))
		}
		tasks = append(tasks, emptyStyle.Render("(empty)"))
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
