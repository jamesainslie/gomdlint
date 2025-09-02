package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// FileManager handles safe file I/O operations with support for atomic writes,
// locking, and proper error handling.
type FileManager struct {
	options *FixOptions

	// File locking to prevent concurrent access
	fileLocks  map[string]*sync.RWMutex
	locksMutex sync.RWMutex
}

// NewFileManager creates a new file manager with the specified options.
func NewFileManager(options *FixOptions) *FileManager {
	return &FileManager{
		options:   options,
		fileLocks: make(map[string]*sync.RWMutex),
	}
}

// ReadFile reads the content of a file safely with proper locking.
func (fm *FileManager) ReadFile(ctx context.Context, filename string) (string, error) {
	// Acquire read lock for the file
	lock := fm.getFileLock(filename)
	lock.RLock()
	defer lock.RUnlock()

	// Check context
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	// Read file content
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	return string(content), nil
}

// WriteFile writes content to a file safely with atomic operations if enabled.
func (fm *FileManager) WriteFile(ctx context.Context, filename string, content string) error {
	// Acquire write lock for the file
	lock := fm.getFileLock(filename)
	lock.Lock()
	defer lock.Unlock()

	// Check context
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if fm.options.AtomicOperations {
		return fm.writeFileAtomic(filename, content)
	}

	return fm.writeFileDirect(filename, content)
}

// writeFileAtomic writes content atomically using a temporary file and rename.
func (fm *FileManager) writeFileAtomic(filename string, content string) error {
	// Create a temporary file in the same directory
	dir := filepath.Dir(filename)
	base := filepath.Base(filename)

	tempFile, err := os.CreateTemp(dir, fmt.Sprintf(".%s.tmp.*", base))
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}

	tempPath := tempFile.Name()

	// Ensure cleanup on failure
	defer func() {
		if tempFile != nil {
			tempFile.Close()
			os.Remove(tempPath)
		}
	}()

	// Write content to temporary file
	if _, err := tempFile.WriteString(content); err != nil {
		return fmt.Errorf("failed to write to temporary file: %w", err)
	}

	// Sync to ensure data is written to disk
	if err := tempFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temporary file: %w", err)
	}

	// Close the temporary file
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}
	tempFile = nil // Prevent cleanup

	// Get original file permissions
	originalInfo, err := os.Stat(filename)
	if err != nil {
		return fmt.Errorf("failed to get original file info: %w", err)
	}

	// Set permissions on temporary file
	if err := os.Chmod(tempPath, originalInfo.Mode()); err != nil {
		os.Remove(tempPath) // Cleanup
		return fmt.Errorf("failed to set permissions on temporary file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, filename); err != nil {
		os.Remove(tempPath) // Cleanup
		return fmt.Errorf("failed to rename temporary file to final location: %w", err)
	}

	return nil
}

// writeFileDirect writes content directly to the file.
func (fm *FileManager) writeFileDirect(filename string, content string) error {
	// Get original file info for permissions
	originalInfo, err := os.Stat(filename)
	if err != nil {
		return fmt.Errorf("failed to get original file info: %w", err)
	}

	// Write content
	if err := os.WriteFile(filename, []byte(content), originalInfo.Mode()); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// getFileLock returns a lock for the specified file, creating one if it doesn't exist.
func (fm *FileManager) getFileLock(filename string) *sync.RWMutex {
	fm.locksMutex.Lock()
	defer fm.locksMutex.Unlock()

	// Normalize path
	absPath, err := filepath.Abs(filename)
	if err != nil {
		// Fall back to original filename if abs path fails
		absPath = filename
	}

	lock, exists := fm.fileLocks[absPath]
	if !exists {
		lock = &sync.RWMutex{}
		fm.fileLocks[absPath] = lock
	}

	return lock
}

// CleanupLocks removes locks for files that are no longer needed.
func (fm *FileManager) CleanupLocks(filenames []string) {
	fm.locksMutex.Lock()
	defer fm.locksMutex.Unlock()

	// Convert to set for faster lookup
	keepSet := make(map[string]bool)
	for _, filename := range filenames {
		absPath, err := filepath.Abs(filename)
		if err != nil {
			absPath = filename
		}
		keepSet[absPath] = true
	}

	// Remove locks for files not in the keep set
	for path := range fm.fileLocks {
		if !keepSet[path] {
			delete(fm.fileLocks, path)
		}
	}
}

// ValidateFileAccess checks if a file can be read and written.
func (fm *FileManager) ValidateFileAccess(filename string) error {
	// Check if file exists
	info, err := os.Stat(filename)
	if err != nil {
		return fmt.Errorf("file does not exist or is not accessible: %w", err)
	}

	// Check if it's a regular file
	if !info.Mode().IsRegular() {
		return fmt.Errorf("not a regular file: %s", filename)
	}

	// Check read permission
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("file is not readable: %w", err)
	}
	file.Close()

	// Check write permission if we need to modify files
	if !fm.options.DryRun && fm.options.OverwriteFiles {
		file, err := os.OpenFile(filename, os.O_WRONLY, 0)
		if err != nil {
			return fmt.Errorf("file is not writable: %w", err)
		}
		file.Close()
	}

	return nil
}
