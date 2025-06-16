package worker

import (
	"os"
	"path/filepath"
)

// ensureDirectory creates a directory if it doesn't exist
func ensureDirectory(dir string) error {
	if dir == "" {
		return nil
	}

	// Clean the path
	dir = filepath.Clean(dir)

	// Check if directory exists
	if _, err := os.Stat(dir); err == nil {
		return nil // Directory already exists
	}

	// Create directory with proper permissions
	return os.MkdirAll(dir, 0755)
}
