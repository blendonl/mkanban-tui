package daemon

import (
	"context"
	"fmt"
	"mkanban/internal/application/strategy"
	"mkanban/internal/application/usecase/session"
	"mkanban/internal/domain/entity"
	"mkanban/internal/domain/service"
	"mkanban/internal/infrastructure/config"
	"sync"
	"time"
)

// SessionManager manages session tracking and board synchronization
type SessionManager struct {
	config          *config.Config
	trackSessionsUC *session.TrackSessionsUseCase
	sessionTracker  service.SessionTracker
	changeWatcher   service.ChangeWatcher
	strategies      []strategy.BoardSyncStrategy
	activeSession   *entity.Session
	watchedPaths    map[string]bool
	mu              sync.RWMutex
	stopChan        chan struct{}
	stopped         bool
}

// NewSessionManager creates a new SessionManager
func NewSessionManager(
	config *config.Config,
	trackSessionsUC *session.TrackSessionsUseCase,
	sessionTracker service.SessionTracker,
	changeWatcher service.ChangeWatcher,
	strategies []strategy.BoardSyncStrategy,
) *SessionManager {
	return &SessionManager{
		config:          config,
		trackSessionsUC: trackSessionsUC,
		sessionTracker:  sessionTracker,
		changeWatcher:   changeWatcher,
		strategies:      strategies,
		watchedPaths:    make(map[string]bool),
		stopChan:        make(chan struct{}),
		stopped:         false,
	}
}

// Start begins the session tracking loop
func (sm *SessionManager) Start(ctx context.Context) error {
	// Check if session tracking is enabled
	if !sm.config.SessionTracking.Enabled {
		fmt.Println("[SessionManager] Session tracking is disabled in config")
		return nil
	}

	// Check if session tracker is available
	if !sm.sessionTracker.IsAvailable() {
		fmt.Println("[SessionManager] Session tracker is not available (tmux may not be running)")
		return nil
	}

	fmt.Printf("[SessionManager] Starting session tracking (poll interval: %ds)\n", sm.config.SessionTracking.PollInterval)

	// Start polling loop in background
	go sm.pollLoop(ctx)

	return nil
}

// Stop stops the session tracking loop
func (sm *SessionManager) Stop() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.stopped {
		return nil
	}

	sm.stopped = true
	close(sm.stopChan)

	// Close the change watcher
	if sm.changeWatcher != nil {
		if err := sm.changeWatcher.Close(); err != nil {
			return fmt.Errorf("failed to close change watcher: %w", err)
		}
	}

	return nil
}

// GetActiveSession returns the currently active session
func (sm *SessionManager) GetActiveSession() *entity.Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.activeSession
}

// pollLoop runs the periodic session tracking
func (sm *SessionManager) pollLoop(ctx context.Context) {
	// Run initial sync
	sm.syncSessions(ctx)

	// Create ticker for polling
	pollInterval := time.Duration(sm.config.SessionTracking.PollInterval) * time.Second
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sm.syncSessions(ctx)

		case <-sm.stopChan:
			return

		case <-ctx.Done():
			return
		}
	}
}

// syncSessions synchronizes all sessions and updates watchers
func (sm *SessionManager) syncSessions(ctx context.Context) {
	// Track all sessions and get active session
	activeSession, err := sm.trackSessionsUC.Execute(ctx)
	if err != nil {
		fmt.Printf("[SessionManager] Error tracking sessions: %v\n", err)
		return
	}

	// Check if active session changed
	sm.mu.Lock()
	previousSession := sm.activeSession
	sm.activeSession = activeSession
	sm.mu.Unlock()

	// Log session changes
	if previousSession == nil && activeSession != nil {
		fmt.Printf("[SessionManager] Active session detected: %s (working dir: %s)\n",
			activeSession.Name(), activeSession.WorkingDir())
	} else if previousSession != nil && activeSession == nil {
		fmt.Printf("[SessionManager] Active session ended: %s\n", previousSession.Name())
	} else if previousSession != nil && activeSession != nil && previousSession.Name() != activeSession.Name() {
		fmt.Printf("[SessionManager] Active session changed: %s -> %s (working dir: %s)\n",
			previousSession.Name(), activeSession.Name(), activeSession.WorkingDir())
	}

	// Setup watchers if enabled
	if sm.config.SessionTracking.GitSync.WatchForChanges {
		sm.setupWatchers(ctx, activeSession)
	}
}

// setupWatchers sets up file system watchers for git repositories
func (sm *SessionManager) setupWatchers(ctx context.Context, activeSession *entity.Session) {
	if activeSession == nil || sm.changeWatcher == nil {
		return
	}

	// Find the strategy that can handle this session
	var selectedStrategy strategy.BoardSyncStrategy
	for _, strat := range sm.strategies {
		if strat.CanHandle(activeSession) {
			selectedStrategy = strat
			break
		}
	}

	if selectedStrategy == nil || !selectedStrategy.ShouldWatch() {
		return
	}

	// Get the watch path
	watchPath := selectedStrategy.GetWatchPath(activeSession)
	if watchPath == "" {
		return
	}

	// Check if already watching
	sm.mu.RLock()
	alreadyWatching := sm.watchedPaths[watchPath]
	sm.mu.RUnlock()

	if alreadyWatching {
		return
	}

	// Setup the watcher
	callback := func() {
		fmt.Printf("[SessionManager] File changes detected in: %s\n", watchPath)
		// Re-sync this session when changes are detected
		if err := sm.trackSessionsUC.ExecuteForSession(ctx, activeSession); err != nil {
			fmt.Printf("[SessionManager] Error syncing session after file change: %v\n", err)
		}
	}

	if err := sm.changeWatcher.Watch(watchPath, callback); err != nil {
		fmt.Printf("[SessionManager] Error setting up watcher for %s: %v\n", watchPath, err)
		return
	}

	fmt.Printf("[SessionManager] Now watching for changes in: %s\n", watchPath)

	// Mark as watching
	sm.mu.Lock()
	sm.watchedPaths[watchPath] = true
	sm.mu.Unlock()
}
