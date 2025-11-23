package tui

import (
	"context"
	"os"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"mkanban/internal/application/dto"
	"mkanban/internal/daemon"
)

// boardRefreshMsg is sent when a board refresh is completed
type boardRefreshMsg struct {
	board *dto.BoardDTO
}

// checkBoardChange checks if the active board has changed
func checkBoardChange(m Model) tea.Cmd {
	return func() tea.Msg {
		// Only check if we're in tmux
		if os.Getenv("TMUX") == "" {
			return nil
		}

		// Try to get active board from daemon
		client := daemon.NewClient(m.container.Config)

		// Check if daemon is running
		if err := client.Ping(); err != nil {
			return nil
		}

		// Get active board ID
		activeBoardID, err := client.GetActiveBoard()
		if err != nil || activeBoardID == "" {
			return nil
		}

		// Check if board changed
		if activeBoardID != m.lastBoardID {
			// Board changed, reload it
			ctx := context.Background()
			newBoard, err := m.container.GetBoardUseCase.Execute(ctx, activeBoardID)
			if err != nil {
				return nil
			}
			return boardRefreshMsg{board: newBoard}
		}

		return nil
	}
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		// Check for board changes and schedule next tick
		return m, tea.Batch(checkBoardChange(m), doTick())

	case boardRefreshMsg:
		// Board was refreshed due to session change
		if msg.board != nil {
			m.board = msg.board
			m.lastBoardID = msg.board.ID

			// Reset scroll offsets for new board
			m.scrollOffsets = make([]int, len(m.board.Columns))

			// Reset focus to safe values
			m.focusedColumn = 0
			m.focusedTask = 0
			m.horizontalScrollOffset = 0
			m.clampTaskFocus()
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Update horizontal scroll to keep focused column visible after resize
		m.updateHorizontalScroll(m.calculateVisibleColumns())
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, keys.Left):
			m.moveLeft()

		case key.Matches(msg, keys.Right):
			m.moveRight()

		case key.Matches(msg, keys.Up):
			m.moveUp()

		case key.Matches(msg, keys.Down):
			m.moveDown()

		case key.Matches(msg, keys.Move):
			m.moveTask()

		case key.Matches(msg, keys.Add):
			m.addTask()

		case key.Matches(msg, keys.Delete):
			m.deleteTask()
		}
	}

	return m, nil
}

// moveLeft moves focus to the left column
func (m *Model) moveLeft() {
	if m.focusedColumn > 0 {
		m.focusedColumn--
		m.focusedTask = 0
		m.clampTaskFocus()
		// Update vertical scroll for new column
		availableTaskHeight := m.height - 8
		maxVisibleTasks := availableTaskHeight / 6
		if maxVisibleTasks < 1 {
			maxVisibleTasks = 1
		}
		m.updateScroll(maxVisibleTasks)
		// Update horizontal scroll to keep column visible
		m.updateHorizontalScroll(m.calculateVisibleColumns())
	}
}

// moveRight moves focus to the right column
func (m *Model) moveRight() {
	if m.focusedColumn < len(m.board.Columns)-1 {
		m.focusedColumn++
		m.focusedTask = 0
		m.clampTaskFocus()
		// Update vertical scroll for new column
		availableTaskHeight := m.height - 8
		maxVisibleTasks := availableTaskHeight / 6
		if maxVisibleTasks < 1 {
			maxVisibleTasks = 1
		}
		m.updateScroll(maxVisibleTasks)
		// Update horizontal scroll to keep column visible
		m.updateHorizontalScroll(m.calculateVisibleColumns())
	}
}

// calculateVisibleColumns returns how many columns can fit in the terminal width
func (m *Model) calculateVisibleColumns() int {
	const minColumnWidth = 30
	const columnSpacing = 2
	columnWidthWithSpacing := minColumnWidth + columnSpacing
	indicatorWidth := 5
	availableWidthForColumns := m.width - (indicatorWidth * 2)
	maxVisibleColumns := availableWidthForColumns / columnWidthWithSpacing
	if maxVisibleColumns < 1 {
		maxVisibleColumns = 1
	}
	return maxVisibleColumns
}

// moveUp moves focus to the task above
func (m *Model) moveUp() {
	if m.focusedTask > 0 {
		m.focusedTask--
		// Update scroll to keep task visible
		availableTaskHeight := m.height - 8
		maxVisibleTasks := availableTaskHeight / 6
		if maxVisibleTasks < 1 {
			maxVisibleTasks = 1
		}
		m.updateScroll(maxVisibleTasks)
	}
}

// moveDown moves focus to the task below
func (m *Model) moveDown() {
	taskCount := m.currentColumnTaskCount()
	if m.focusedTask < taskCount-1 {
		m.focusedTask++
		// Update scroll to keep task visible
		availableTaskHeight := m.height - 8
		maxVisibleTasks := availableTaskHeight / 6
		if maxVisibleTasks < 1 {
			maxVisibleTasks = 1
		}
		m.updateScroll(maxVisibleTasks)
	}
}

// moveTask moves the currently focused task to the next column
func (m *Model) moveTask() {
	// Check if there's a task to move
	if m.currentColumnTaskCount() == 0 {
		return
	}

	// Can't move from the last column
	if m.focusedColumn >= len(m.board.Columns)-1 {
		return
	}

	// Get the current task
	task := m.board.Columns[m.focusedColumn].Tasks[m.focusedTask]

	// Get target column name
	targetColumnName := m.board.Columns[m.focusedColumn+1].Name

	// Use the MoveTask use case
	ctx := context.Background()
	moveReq := dto.MoveTaskRequest{
		TaskID:           task.ID,
		TargetColumnName: targetColumnName,
	}

	updatedBoard, err := m.container.MoveTaskUseCase.Execute(ctx, m.board.ID, moveReq)
	if err != nil {
		// Handle error (for now, just return)
		return
	}

	// Update local state
	m.board = updatedBoard

	// Ensure scroll offsets array matches board columns
	if len(m.scrollOffsets) != len(m.board.Columns) {
		m.scrollOffsets = make([]int, len(m.board.Columns))
	}

	// Move focus to next column
	m.focusedColumn++
	m.focusedTask = len(m.board.Columns[m.focusedColumn].Tasks) - 1
	m.clampTaskFocus()

	// Update vertical scroll for new position
	availableTaskHeight := m.height - 8
	maxVisibleTasks := availableTaskHeight / 6
	if maxVisibleTasks < 1 {
		maxVisibleTasks = 1
	}
	m.updateScroll(maxVisibleTasks)
	// Update horizontal scroll to keep column visible
	m.updateHorizontalScroll(m.calculateVisibleColumns())
}

// addTask adds a new task to the current column
func (m *Model) addTask() {
	// Get current column name
	columnName := m.board.Columns[m.focusedColumn].Name

	// Use the CreateTask use case
	ctx := context.Background()
	createReq := dto.CreateTaskRequest{
		Title:       "New Task",
		Description: "Edit this task",
		Priority:    "none",
		ColumnName:  columnName,
	}

	_, err := m.container.CreateTaskUseCase.Execute(ctx, m.board.ID, createReq)
	if err != nil {
		// Handle error (for now, just return)
		return
	}

	// Reload the board to get updated state
	updatedBoard, err := m.container.GetBoardUseCase.Execute(ctx, m.board.ID)
	if err != nil {
		return
	}

	m.board = updatedBoard

	// Ensure scroll offsets array matches board columns
	if len(m.scrollOffsets) != len(m.board.Columns) {
		m.scrollOffsets = make([]int, len(m.board.Columns))
	}

	// Focus the new task
	m.focusedTask = len(m.board.Columns[m.focusedColumn].Tasks) - 1

	// Update scroll to show the new task
	availableTaskHeight := m.height - 8
	maxVisibleTasks := availableTaskHeight / 6
	if maxVisibleTasks < 1 {
		maxVisibleTasks = 1
	}
	m.updateScroll(maxVisibleTasks)
}

// deleteTask removes the currently focused task
func (m *Model) deleteTask() {
	// Check if there's a task to delete
	if m.currentColumnTaskCount() == 0 {
		return
	}

	// For now, just skip delete as we need to implement DeleteTaskUseCase
	// TODO: Implement DeleteTaskUseCase and call it here

	// Adjust focus
	m.clampTaskFocus()
}

// clampTaskFocus ensures the task focus is within valid bounds
func (m *Model) clampTaskFocus() {
	taskCount := m.currentColumnTaskCount()
	if taskCount == 0 {
		m.focusedTask = 0
	} else if m.focusedTask >= taskCount {
		m.focusedTask = taskCount - 1
	}
}
