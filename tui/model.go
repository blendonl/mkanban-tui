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
	focusedColumn int // which column is currently selected
	focusedTask   int // which task in the current column is selected
	width         int
	height        int
}

// NewModel creates a new TUI model
func NewModel(board *dto.BoardDTO, container *di.Container) Model {
	return Model{
		board:         board,
		container:     container,
		focusedColumn: 0,
		focusedTask:   0,
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
