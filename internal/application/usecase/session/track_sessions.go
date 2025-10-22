package session

import (
	"context"
	"fmt"
	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/service"
)

// TrackSessionsUseCase tracks all active sessions and syncs their boards
type TrackSessionsUseCase struct {
	sessionTracker service.SessionTracker
	syncUseCase    *SyncSessionBoardUseCase
}

// NewTrackSessionsUseCase creates a new TrackSessionsUseCase
func NewTrackSessionsUseCase(
	sessionTracker service.SessionTracker,
	syncUseCase *SyncSessionBoardUseCase,
) *TrackSessionsUseCase {
	return &TrackSessionsUseCase{
		sessionTracker: sessionTracker,
		syncUseCase:    syncUseCase,
	}
}

// Execute tracks all sessions and syncs their boards
// Returns the active session (if any) and any errors encountered
func (uc *TrackSessionsUseCase) Execute(ctx context.Context) (*entity.Session, error) {
	// Check if session tracker is available
	if !uc.sessionTracker.IsAvailable() {
		return nil, nil
	}

	// Get all active sessions
	sessions, err := uc.sessionTracker.ListSessions()
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	// Get active session
	activeSession, err := uc.sessionTracker.GetActiveSession()
	if err != nil {
		return nil, fmt.Errorf("failed to get active session: %w", err)
	}

	// Sync all sessions
	for _, session := range sessions {
		if err := uc.syncUseCase.Execute(ctx, session); err != nil {
			// Log error but continue with other sessions
			// In a real implementation, we'd use a proper logger here
			_ = err
			continue
		}
	}

	return activeSession, nil
}

// ExecuteForSession syncs a specific session
func (uc *TrackSessionsUseCase) ExecuteForSession(ctx context.Context, session *entity.Session) error {
	if session == nil {
		return fmt.Errorf("session cannot be nil")
	}

	return uc.syncUseCase.Execute(ctx, session)
}
