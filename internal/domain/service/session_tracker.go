package service

import "mkanban/internal/domain/entity"

// SessionTracker defines the interface for tracking terminal multiplexer sessions
// This abstraction allows for different implementations (tmux, zellij, screen, etc.)
type SessionTracker interface {
	// ListSessions returns all active sessions
	ListSessions() ([]*entity.Session, error)

	// GetActiveSession returns the currently active/focused session
	GetActiveSession() (*entity.Session, error)

	// IsAvailable checks if the session tracker is available on the system
	// (e.g., tmux is installed and running)
	IsAvailable() bool
}
