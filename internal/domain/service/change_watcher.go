package service

// ChangeWatcher defines the interface for watching file system changes
// This abstraction allows for different implementations and easier testing
type ChangeWatcher interface {
	// Watch starts watching the specified path for changes
	// The callback function is called when changes are detected
	Watch(path string, callback func()) error

	// Unwatch stops watching the specified path
	Unwatch(path string) error

	// Close stops all watchers and releases resources
	Close() error
}
