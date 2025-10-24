package service

import "mkanban/internal/domain/entity"

// RepoPathResolver defines the interface for resolving repository paths
// from active terminal multiplexer sessions
type RepoPathResolver interface {
	// GetRepoPathForBoard returns the repository path for a given board
	// by inferring the associated session from the board context
	// Returns an error if the session is not active or cannot be found
	GetRepoPathForBoard(board *entity.Board) (string, error)

	// GetRepoPathForTask returns the repository path for a given task
	// by inferring the associated session from the task's board context
	// Returns an error if the session is not active or cannot be found
	GetRepoPathForTask(task *entity.Task) (string, error)
}
