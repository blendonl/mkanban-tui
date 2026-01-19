package session

import (
	"context"
	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/repository"
	"mkanban/internal/domain/service"
	"mkanban/internal/domain/valueobject"
	"mkanban/pkg/slug"
)

// GetActiveSessionBoardUseCase retrieves the board ID for the active session
type GetActiveSessionBoardUseCase struct {
	sessionTracker service.SessionTracker
	boardRepo      repository.BoardRepository
	syncUseCase    *SyncSessionBoardUseCase
	boardPlanner   *SessionBoardPlanner
}

// NewGetActiveSessionBoardUseCase creates a new GetActiveSessionBoardUseCase
func NewGetActiveSessionBoardUseCase(
	sessionTracker service.SessionTracker,
	boardRepo repository.BoardRepository,
	syncUseCase *SyncSessionBoardUseCase,
	boardPlanner *SessionBoardPlanner,
) *GetActiveSessionBoardUseCase {
	return &GetActiveSessionBoardUseCase{
		sessionTracker: sessionTracker,
		boardRepo:      boardRepo,
		syncUseCase:    syncUseCase,
		boardPlanner:   boardPlanner,
	}
}

// Execute returns the board ID for the active session
// If sessionName is provided, it looks up that specific session
// Returns empty string if no active session or session tracking is unavailable
func (uc *GetActiveSessionBoardUseCase) Execute(ctx context.Context, sessionName string) (string, error) {
	// Check if session tracker is available
	if !uc.sessionTracker.IsAvailable() {
		return "", nil
	}

	var activeSession *entity.Session
	var err error

	if sessionName != "" {
		// Look up the specific session by name
		activeSession, err = uc.findSessionByName(sessionName)
	} else {
		// Fall back to getting the first attached session
		activeSession, err = uc.sessionTracker.GetActiveSession()
	}

	if err != nil {
		return "", err
	}
	if activeSession == nil {
		return "", nil
	}

	plan, err := uc.boardPlanner.Plan(activeSession)
	if err != nil {
		return "", err
	}

	boardID, err := valueobject.BuildBoardID(plan.ProjectID, slug.Generate(plan.SyncBoardName))
	if err != nil {
		return "", err
	}

	// Check if board exists
	exists, err := uc.boardRepo.Exists(ctx, boardID)
	if err != nil {
		return "", err
	}

	if !exists {
		// Board doesn't exist yet - sync it now
		if uc.syncUseCase != nil {
			if err := uc.syncUseCase.Execute(ctx, activeSession); err != nil {
				return "", err
			}
			return boardID, nil
		}
		return "", nil
	}

	return boardID, nil
}

// findSessionByName finds a session by its name
func (uc *GetActiveSessionBoardUseCase) findSessionByName(name string) (*entity.Session, error) {
	sessions, err := uc.sessionTracker.ListSessions()
	if err != nil {
		return nil, err
	}

	for _, session := range sessions {
		if session.Name() == name {
			return session, nil
		}
	}

	return nil, nil
}
