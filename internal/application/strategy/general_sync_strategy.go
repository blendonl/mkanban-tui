package strategy

import "mkanban/internal/domain/entity"

// GeneralSyncStrategy handles non-git sessions
// All non-git sessions share a single "General Tasks" board
type GeneralSyncStrategy struct {
	generalBoardName string
}

// NewGeneralSyncStrategy creates a new GeneralSyncStrategy
func NewGeneralSyncStrategy(generalBoardName string) *GeneralSyncStrategy {
	if generalBoardName == "" {
		generalBoardName = "General Tasks"
	}
	return &GeneralSyncStrategy{
		generalBoardName: generalBoardName,
	}
}

// CanHandle returns true for all sessions (this is the fallback strategy)
func (s *GeneralSyncStrategy) CanHandle(session *entity.Session) bool {
	// This is a catch-all strategy that handles any session
	// It's typically used as the last strategy in the chain
	return true
}

// Sync does nothing for general sessions
// The board exists, but no automatic tasks are created
// Users can manually add tasks to this shared board
func (s *GeneralSyncStrategy) Sync(session *entity.Session, board *entity.Board) error {
	// No automatic synchronization needed for general sessions
	// The board is just a shared space for manual task management
	return nil
}

// ShouldWatch returns false as general boards don't need file system watching
func (s *GeneralSyncStrategy) ShouldWatch() bool {
	return false
}

// GetWatchPath returns empty string as no watching is needed
func (s *GeneralSyncStrategy) GetWatchPath(session *entity.Session) string {
	return ""
}
