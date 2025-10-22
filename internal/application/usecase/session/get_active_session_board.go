package session

import (
	"context"
	"mkanban/internal/application/strategy"
	"mkanban/internal/domain/repository"
	"mkanban/internal/domain/service"
	"mkanban/pkg/slug"
)

// GetActiveSessionBoardUseCase retrieves the board ID for the active session
type GetActiveSessionBoardUseCase struct {
	sessionTracker service.SessionTracker
	boardRepo      repository.BoardRepository
	strategies     []strategy.BoardSyncStrategy
}

// NewGetActiveSessionBoardUseCase creates a new GetActiveSessionBoardUseCase
func NewGetActiveSessionBoardUseCase(
	sessionTracker service.SessionTracker,
	boardRepo repository.BoardRepository,
	strategies []strategy.BoardSyncStrategy,
) *GetActiveSessionBoardUseCase {
	return &GetActiveSessionBoardUseCase{
		sessionTracker: sessionTracker,
		boardRepo:      boardRepo,
		strategies:     strategies,
	}
}

// Execute returns the board ID for the active session
// Returns empty string if no active session or session tracking is unavailable
func (uc *GetActiveSessionBoardUseCase) Execute(ctx context.Context) (string, error) {
	// Check if session tracker is available
	if !uc.sessionTracker.IsAvailable() {
		return "", nil
	}

	// Get active session
	activeSession, err := uc.sessionTracker.GetActiveSession()
	if err != nil {
		return "", err
	}
	if activeSession == nil {
		return "", nil
	}

	// Find the appropriate strategy for this session
	var selectedStrategy strategy.BoardSyncStrategy
	for _, strat := range uc.strategies {
		if strat.CanHandle(activeSession) {
			selectedStrategy = strat
			break
		}
	}

	if selectedStrategy == nil {
		// No strategy can handle this session
		return "", nil
	}

	// Get board name from strategy
	boardName := selectedStrategy.GetBoardName(activeSession)
	if boardName == "" {
		return "", nil
	}

	// Generate board ID
	boardID := slug.Generate(boardName)

	// Check if board exists
	exists, err := uc.boardRepo.Exists(ctx, boardID)
	if err != nil {
		return "", err
	}

	if !exists {
		// Board doesn't exist yet
		// It will be created by the sync process
		return "", nil
	}

	return boardID, nil
}
