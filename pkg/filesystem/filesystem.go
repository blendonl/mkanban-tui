package filesystem

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// EnsureDir creates a directory if it doesn't exist
func EnsureDir(path string, perm fs.FileMode) error {
	if err := os.MkdirAll(path, perm); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}
	return nil
}

// Exists checks if a path exists
func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// SafeWrite writes data to a file atomically by writing to a temp file first
func SafeWrite(path string, data []byte, perm fs.FileMode) error {
	// Ensure the parent directory exists
	dir := filepath.Dir(path)
	if err := EnsureDir(dir, 0755); err != nil {
		return err
	}

	// Write to a temporary file first
	tempFile := path + ".tmp"
	if err := os.WriteFile(tempFile, data, perm); err != nil {
		return fmt.Errorf("failed to write temp file %s: %w", tempFile, err)
	}

	// Atomically rename the temp file to the target file
	if err := os.Rename(tempFile, path); err != nil {
		// Clean up temp file on error
		os.Remove(tempFile)
		return fmt.Errorf("failed to rename temp file to %s: %w", path, err)
	}

	return nil
}

// RemoveDir removes a directory and all its contents
func RemoveDir(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("failed to remove directory %s: %w", path, err)
	}
	return nil
}
