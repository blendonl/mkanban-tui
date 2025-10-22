package external

import (
	"fmt"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// FSNotifyWatcher implements ChangeWatcher using fsnotify
type FSNotifyWatcher struct {
	watcher   *fsnotify.Watcher
	callbacks map[string]func()
	mu        sync.RWMutex
	done      chan struct{}
	started   bool
}

// NewFSNotifyWatcher creates a new FSNotifyWatcher
func NewFSNotifyWatcher() (*FSNotifyWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	w := &FSNotifyWatcher{
		watcher:   watcher,
		callbacks: make(map[string]func()),
		done:      make(chan struct{}),
		started:   false,
	}

	// Start the event loop
	go w.eventLoop()

	return w, nil
}

// Watch starts watching the specified path for changes
func (w *FSNotifyWatcher) Watch(path string, callback func()) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Add the callback
	w.callbacks[path] = callback

	// Start watching the path
	if err := w.watcher.Add(path); err != nil {
		delete(w.callbacks, path)
		return fmt.Errorf("failed to watch path %s: %w", path, err)
	}

	return nil
}

// Unwatch stops watching the specified path
func (w *FSNotifyWatcher) Unwatch(path string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Remove the callback
	delete(w.callbacks, path)

	// Stop watching the path
	if err := w.watcher.Remove(path); err != nil {
		return fmt.Errorf("failed to unwatch path %s: %w", path, err)
	}

	return nil
}

// Close stops all watchers and releases resources
func (w *FSNotifyWatcher) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Signal event loop to stop
	close(w.done)

	// Close the watcher
	if err := w.watcher.Close(); err != nil {
		return fmt.Errorf("failed to close watcher: %w", err)
	}

	return nil
}

// eventLoop processes file system events
func (w *FSNotifyWatcher) eventLoop() {
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Handle write and create events
			if event.Op&fsnotify.Write == fsnotify.Write ||
				event.Op&fsnotify.Create == fsnotify.Create ||
				event.Op&fsnotify.Remove == fsnotify.Remove {

				w.handleEvent(event.Name)
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			// Log error but continue watching
			// In production, we'd use a proper logger
			_ = err

		case <-w.done:
			return
		}
	}
}

// handleEvent triggers the callback for a path or its parent directory
func (w *FSNotifyWatcher) handleEvent(eventPath string) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// Check for exact path match
	if callback, exists := w.callbacks[eventPath]; exists {
		go callback()
		return
	}

	// Check if this event is for a file inside a watched directory
	// The callback is associated with the directory path
	for watchedPath, callback := range w.callbacks {
		// Simple contains check - in production we'd use filepath.Dir
		if len(eventPath) > len(watchedPath) && eventPath[:len(watchedPath)] == watchedPath {
			go callback()
			return
		}
	}
}
