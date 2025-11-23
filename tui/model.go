package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"mkanban/internal/application/dto"
	"mkanban/internal/di"
)

// Model represents the TUI state
type Model struct {
	board                  *dto.BoardDTO
	container              *di.Container
	focusedColumn          int   // which column is currently selected
	focusedTask            int   // which task in the current column is selected
	scrollOffsets          []int // scroll offset for each column (vertical)
	horizontalScrollOffset int   // horizontal scroll offset for columns
	width                  int
	height                 int
	lastBoardID            string // track the last board ID to detect changes
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
		lastBoardID:   board.ID,
	}
}

// tickMsg is sent when the ticker fires
type tickMsg time.Time

// doTick returns a command that waits for a tick
func doTick() tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	// Start the ticker for periodic board refresh checks
	return doTick()
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

// Helper to update horizontal scroll to keep focused column visible
func (m *Model) updateHorizontalScroll(visibleColumns int) {
	if visibleColumns <= 0 {
		visibleColumns = 1
	}

	totalColumns := len(m.board.Columns)
	if totalColumns == 0 {
		m.horizontalScrollOffset = 0
		return
	}

	// Ensure focused column is visible
	if m.focusedColumn < m.horizontalScrollOffset {
		// Focused column is to the left of viewport
		m.horizontalScrollOffset = m.focusedColumn
	} else if m.focusedColumn >= m.horizontalScrollOffset+visibleColumns {
		// Focused column is to the right of viewport
		m.horizontalScrollOffset = m.focusedColumn - visibleColumns + 1
	}

	// Clamp horizontal scroll offset
	maxScroll := totalColumns - visibleColumns
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.horizontalScrollOffset > maxScroll {
		m.horizontalScrollOffset = maxScroll
	}
	if m.horizontalScrollOffset < 0 {
		m.horizontalScrollOffset = 0
	}
}
