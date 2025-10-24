package service

import (
	"fmt"
	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/service"
)

// TmuxRepoPathResolver implements RepoPathResolver by inferring repository paths
// from active tmux sessions based on board names
type TmuxRepoPathResolver struct {
	sessionTracker service.SessionTracker
	vcsProvider    service.VCSProvider
}

// NewTmuxRepoPathResolver creates a new TmuxRepoPathResolver
func NewTmuxRepoPathResolver(
	sessionTracker service.SessionTracker,
	vcsProvider service.VCSProvider,
) *TmuxRepoPathResolver {
	return &TmuxRepoPathResolver{
		sessionTracker: sessionTracker,
		vcsProvider:    vcsProvider,
	}
}

// GetRepoPathForBoard returns the repository path for a board by finding the
// matching tmux session (board name should match session name) and getting
// the repo root from the session's working directory
func (r *TmuxRepoPathResolver) GetRepoPathForBoard(board *entity.Board) (string, error) {
	if !r.sessionTracker.IsAvailable() {
		return "", fmt.Errorf("tmux session tracker is not available")
	}

	boardName := board.Name()

	// Get all sessions
	sessions, err := r.sessionTracker.ListSessions()
	if err != nil {
		return "", fmt.Errorf("failed to list tmux sessions: %w", err)
	}

	// First, try to find a session with matching name
	var matchingSession *entity.Session
	for _, session := range sessions {
		if session.Name() == boardName {
			matchingSession = session
			break
		}
	}

	// If no exact match, try to get the active session as fallback
	if matchingSession == nil {
		activeSession, err := r.sessionTracker.GetActiveSession()
		if err != nil {
			return "", fmt.Errorf("failed to get active session: %w", err)
		}
		if activeSession == nil {
			return "", fmt.Errorf("no active tmux session found for board %s", boardName)
		}
		matchingSession = activeSession
	}

	// Get the working directory from the session
	workingDir := matchingSession.WorkingDir()

	// Check if it's a git repository
	if !r.vcsProvider.IsRepository(workingDir) {
		return "", fmt.Errorf("session working directory %s is not a git repository", workingDir)
	}

	// Get the repository root
	repoRoot, err := r.vcsProvider.GetRepositoryRoot(workingDir)
	if err != nil {
		return "", fmt.Errorf("failed to get repository root: %w", err)
	}

	return repoRoot, nil
}

// GetRepoPathForTask returns the repository path for a task
// Note: Tasks don't have a direct reference to their board, so this method
// is not currently used. Use GetRepoPathForBoard instead when you have the board.
func (r *TmuxRepoPathResolver) GetRepoPathForTask(task *entity.Task) (string, error) {
	return "", fmt.Errorf("GetRepoPathForTask is not implemented - use GetRepoPathForBoard instead")
}
