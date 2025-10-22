package external

import (
	"bytes"
	"fmt"
	"mkanban/internal/domain/entity"
	"os/exec"
	"strings"
)

const sessionTypeTmux = "tmux"

// TmuxSessionTracker implements SessionTracker for tmux
type TmuxSessionTracker struct{}

// NewTmuxSessionTracker creates a new TmuxSessionTracker
func NewTmuxSessionTracker() *TmuxSessionTracker {
	return &TmuxSessionTracker{}
}

// IsAvailable checks if tmux is installed and available
func (t *TmuxSessionTracker) IsAvailable() bool {
	// Check if tmux is installed
	cmd := exec.Command("which", "tmux")
	if err := cmd.Run(); err != nil {
		return false
	}

	// Check if we're in a tmux session or can access tmux
	cmd = exec.Command("tmux", "list-sessions")
	if err := cmd.Run(); err != nil {
		// tmux is installed but not running
		return false
	}

	return true
}

// ListSessions returns all active tmux sessions
func (t *TmuxSessionTracker) ListSessions() ([]*entity.Session, error) {
	// Format: session_name:session_path
	// -F is the format string
	cmd := exec.Command("tmux", "list-sessions", "-F", "#{session_name}:#{pane_current_path}")

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to list tmux sessions: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	sessions := make([]*entity.Session, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		sessionName := parts[0]
		workingDir := parts[1]

		session, err := entity.NewSession(sessionName, workingDir, sessionTypeTmux)
		if err != nil {
			// Skip invalid sessions
			continue
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

// GetActiveSession returns the currently active/focused tmux session
// When running from the daemon, we query tmux for attached sessions
func (t *TmuxSessionTracker) GetActiveSession() (*entity.Session, error) {
	// Query tmux for sessions with their attachment status and working directory
	// Format: session_name:attached:pane_current_path
	cmd := exec.Command("tmux", "list-sessions", "-F", "#{session_name}:#{session_attached}:#{pane_current_path}")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to list tmux sessions: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")

	// Look for an attached session (attached=1)
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 3)
		if len(parts) != 3 {
			continue
		}

		sessionName := parts[0]
		attached := parts[1]
		workingDir := parts[2]

		// Check if this session is attached
		if attached == "1" {
			session, err := entity.NewSession(sessionName, workingDir, sessionTypeTmux)
			if err != nil {
				// Skip invalid sessions
				continue
			}
			return session, nil
		}
	}

	// No attached session found
	return nil, nil
}
