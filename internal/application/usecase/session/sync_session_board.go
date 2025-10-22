package session

import (
	"context"
	"fmt"
	"mkanban/internal/application/strategy"
	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/repository"
	"mkanban/internal/domain/service"
	"mkanban/pkg/slug"
)

// SyncSessionBoardUseCase synchronizes a session's board with its current state
type SyncSessionBoardUseCase struct {
	boardRepo         repository.BoardRepository
	boardService      *service.BoardService
	validationService *service.ValidationService
	strategies        []strategy.BoardSyncStrategy
}

// NewSyncSessionBoardUseCase creates a new SyncSessionBoardUseCase
func NewSyncSessionBoardUseCase(
	boardRepo repository.BoardRepository,
	boardService *service.BoardService,
	validationService *service.ValidationService,
	strategies []strategy.BoardSyncStrategy,
) *SyncSessionBoardUseCase {
	return &SyncSessionBoardUseCase{
		boardRepo:         boardRepo,
		boardService:      boardService,
		validationService: validationService,
		strategies:        strategies,
	}
}

// Execute synchronizes the board for the given session
func (uc *SyncSessionBoardUseCase) Execute(ctx context.Context, session *entity.Session) error {
	if session == nil {
		return fmt.Errorf("session cannot be nil")
	}

	// Find the appropriate strategy for this session
	var selectedStrategy strategy.BoardSyncStrategy
	for _, strat := range uc.strategies {
		if strat.CanHandle(session) {
			selectedStrategy = strat
			break
		}
	}

	if selectedStrategy == nil {
		return fmt.Errorf("no strategy found for session: %s", session.Name())
	}

	// Get board name from strategy
	boardName := selectedStrategy.GetBoardName(session)
	if boardName == "" {
		return fmt.Errorf("strategy returned empty board name for session: %s", session.Name())
	}

	// Generate board ID
	boardID := slug.Generate(boardName)

	// Get or create the board
	board, err := uc.getOrCreateBoard(ctx, boardID, boardName, session)
	if err != nil {
		return fmt.Errorf("failed to get or create board: %w", err)
	}

	// Run the strategy's sync logic
	if err := selectedStrategy.Sync(session, board); err != nil {
		return fmt.Errorf("failed to sync board: %w", err)
	}

	// Save the updated board
	if err := uc.boardRepo.Save(ctx, board); err != nil {
		return fmt.Errorf("failed to save board: %w", err)
	}

	return nil
}

// getOrCreateBoard retrieves an existing board or creates a new one
func (uc *SyncSessionBoardUseCase) getOrCreateBoard(
	ctx context.Context,
	boardID string,
	boardName string,
	session *entity.Session,
) (*entity.Board, error) {
	// Try to load existing board
	board, err := uc.boardRepo.FindByID(ctx, boardID)
	if err == nil {
		// Board exists
		return board, nil
	}

	// Board doesn't exist, create it
	if err != entity.ErrBoardNotFound {
		// Some other error occurred
		return nil, fmt.Errorf("failed to check for existing board: %w", err)
	}

	// Create description with session info
	description := fmt.Sprintf("Session: %s\nWorking Directory: %s",
		session.Name(), session.WorkingDir())

	// Create the board
	board, err = uc.boardService.CreateBoard(ctx, boardName, description)
	if err != nil {
		return nil, fmt.Errorf("failed to create board: %w", err)
	}

	// Add default columns
	if err := uc.addDefaultColumns(board); err != nil {
		return nil, fmt.Errorf("failed to add default columns: %w", err)
	}

	// Save the board with columns
	if err := uc.boardRepo.Save(ctx, board); err != nil {
		return nil, fmt.Errorf("failed to save board with columns: %w", err)
	}

	return board, nil
}

// addDefaultColumns adds the default kanban columns to a new board
func (uc *SyncSessionBoardUseCase) addDefaultColumns(board *entity.Board) error {
	// Default columns for a kanban board
	defaultColumns := []struct {
		name        string
		description string
		order       int
		wipLimit    int
	}{
		{"To Do", "Tasks to be started", 0, 0},
		{"In Progress", "Tasks currently being worked on", 1, 3},
		{"Done", "Completed tasks", 2, 0},
	}

	for _, col := range defaultColumns {
		column, err := entity.NewColumn(col.name, col.description, col.order, col.wipLimit, nil)
		if err != nil {
			return fmt.Errorf("failed to create column %s: %w", col.name, err)
		}

		if err := board.AddColumn(column); err != nil {
			return fmt.Errorf("failed to add column %s to board: %w", col.name, err)
		}
	}

	return nil
}
