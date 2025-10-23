package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"mkanban/internal/application/dto"
	"mkanban/internal/di"
)

// Model represents the TUI state
type Model struct {
	board         *dto.BoardDTO
	container     *di.Container
	focusedColumn int   // which column is currently selected
	focusedTask   int   // which task in the current column is selected
	scrollOffsets []int // scroll offset for each column
	width         int
	height        int
}

// NewModel creates a new TUI model
func NewModel(board *dto.BoardDTO, container *di.Container) Model {
	// Initialize scroll offsets for each column
	scrollOffsets := make([]int, len(board.Columns))

	return Model{
		board:         board,
		container:     container,
		focusedColumn: 0,
		focusedTask:   0,
		scrollOffsets: scrollOffsets,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Helper to get task count in current column
func (m Model) currentColumnTaskCount() int {
	if m.focusedColumn < 0 || m.focusedColumn >= len(m.board.Columns) {
		return 0
	}
	return len(m.board.Columns[m.focusedColumn].Tasks)
}

// Helper to get current task
func (m Model) currentTask() *dto.TaskDTO {
	count := m.currentColumnTaskCount()
	if count == 0 || m.focusedTask < 0 || m.focusedTask >= count {
		return nil
	}
	return &m.board.Columns[m.focusedColumn].Tasks[m.focusedTask]
}

// Helper to get scroll offset for current column
func (m Model) currentScrollOffset() int {
	if m.focusedColumn < 0 || m.focusedColumn >= len(m.scrollOffsets) {
		return 0
	}
	return m.scrollOffsets[m.focusedColumn]
}

// Helper to update scroll position to keep focused task visible
func (m *Model) updateScroll(viewportHeight int) {
	if m.focusedColumn < 0 || m.focusedColumn >= len(m.scrollOffsets) {
		return
	}

	taskCount := m.currentColumnTaskCount()
	if taskCount == 0 {
		m.scrollOffsets[m.focusedColumn] = 0
		return
	}

	scrollOffset := m.scrollOffsets[m.focusedColumn]

	// Ensure focused task is visible
	if m.focusedTask < scrollOffset {
		// Focused task is above viewport
		m.scrollOffsets[m.focusedColumn] = m.focusedTask
	} else if m.focusedTask >= scrollOffset+viewportHeight {
		// Focused task is below viewport
		m.scrollOffsets[m.focusedColumn] = m.focusedTask - viewportHeight + 1
	}

	// Clamp scroll offset
	maxScroll := taskCount - viewportHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.scrollOffsets[m.focusedColumn] > maxScroll {
		m.scrollOffsets[m.focusedColumn] = maxScroll
	}
	if m.scrollOffsets[m.focusedColumn] < 0 {
		m.scrollOffsets[m.focusedColumn] = 0
	}
}
