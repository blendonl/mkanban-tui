package strategy

import "mkanban/internal/domain/entity"

// BoardSyncStrategy defines the interface for different board synchronization strategies
// This allows for pluggable logic based on session type (git repo, general, future: svn, etc.)
type BoardSyncStrategy interface {
	// CanHandle determines if this strategy can handle the given session
	CanHandle(session *entity.Session) bool

	// GetBoardName returns the board name for this session
	// This is used to derive the board ID and find/create the appropriate board
	GetBoardName(session *entity.Session) string

	// Sync synchronizes the board state based on the session
	// This may create tasks, update task positions, modify metadata, etc.
	Sync(session *entity.Session, board *entity.Board) error

	// ShouldWatch returns true if this strategy needs file system watching
	// (e.g., git strategies watch for branch changes)
	ShouldWatch() bool

	// GetWatchPath returns the path to watch for changes (if ShouldWatch returns true)
	// Returns empty string if no watching is needed
	GetWatchPath(session *entity.Session) string
}
