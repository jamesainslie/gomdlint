package service

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	"time"
)

// SafetyManager handles backup creation, validation, and recovery operations.
// It ensures that file operations are safe and can be rolled back if needed.
type SafetyManager struct {
	options *FixOptions
}

// NewSafetyManager creates a new safety manager with the specified options.
func NewSafetyManager(options *FixOptions) *SafetyManager {
	return &SafetyManager{
		options: options,
	}
}

// PrepareFile prepares a file for fixing by creating backups and safety checks.
func (sm *SafetyManager) PrepareFile(ctx context.Context, filename string, operation *FixOperation) error {
	// Check if file exists and is readable
	if err := sm.checkFileAccess(filename); err != nil {
		return fmt.Errorf("file access check failed: %w", err)
	}

	// Create backup if enabled
	if sm.options.CreateBackups {
		backupPath, err := sm.createBackup(ctx, filename)
		if err != nil {
			return fmt.Errorf("backup creation failed: %w", err)
		}
		operation.BackupPath = backupPath
	}

	return nil
}

// ValidateFile validates that the fixed file meets quality requirements.
func (sm *SafetyManager) ValidateFile(ctx context.Context, filename string, operation *FixOperation) error {
	// Read the fixed content
	fixedContent, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read fixed file for validation: %w", err)
	}

	// Basic structural validation
	if err := sm.validateFileStructure(string(fixedContent)); err != nil {
		return fmt.Errorf("file structure validation failed: %w", err)
	}

	// Content integrity checks
	if err := sm.validateContentIntegrity(operation.OriginalContent, string(fixedContent)); err != nil {
		return fmt.Errorf("content integrity validation failed: %w", err)
	}

	// Line ending preservation check
	if sm.options.PreserveLineEndings {
		if err := sm.validateLineEndings(operation.OriginalContent, string(fixedContent)); err != nil {
			return fmt.Errorf("line ending validation failed: %w", err)
		}
	}

	return nil
}

// RecoverFile recovers a file from backup in case of failure.
func (sm *SafetyManager) RecoverFile(ctx context.Context, filename string, operation *FixOperation) error {
	if operation.BackupPath == "" {
		return fmt.Errorf("no backup available for recovery of %s", filename)
	}

	// Copy backup back to original location
	backupContent, err := os.ReadFile(operation.BackupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	if err := os.WriteFile(filename, backupContent, 0644); err != nil {
		return fmt.Errorf("failed to restore from backup: %w", err)
	}

	return nil
}

// CleanupBackups removes backup files after successful operations.
func (sm *SafetyManager) CleanupBackups(ctx context.Context, operations map[string]*FixOperation) error {
	if !sm.options.CreateBackups {
		return nil // No backups to clean up
	}

	var errors []error

	for filename, operation := range operations {
		if operation.Status == FixStatusCompleted && operation.BackupPath != "" {
			if err := os.Remove(operation.BackupPath); err != nil {
				errors = append(errors, fmt.Errorf("failed to remove backup for %s: %w", filename, err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors: %v", errors)
	}

	return nil
}

// checkFileAccess verifies that the file exists and is accessible.
func (sm *SafetyManager) checkFileAccess(filename string) error {
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

	// Check write permission
	if !sm.options.DryRun && sm.options.OverwriteFiles {
		// Try to open for writing (this checks permissions without modifying)
		file, err := os.OpenFile(filename, os.O_WRONLY, 0)
		if err != nil {
			return fmt.Errorf("file is not writable: %w", err)
		}
		file.Close()
	}

	return nil
}

// createBackup creates a backup of the original file.
func (sm *SafetyManager) createBackup(ctx context.Context, filename string) (string, error) {
	// Generate backup filename
	backupPath := sm.generateBackupPath(filename)

	// Read original content
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read original file: %w", err)
	}

	// Write backup
	if err := os.WriteFile(backupPath, content, 0644); err != nil {
		return "", fmt.Errorf("failed to write backup file: %w", err)
	}

	return backupPath, nil
}

// generateBackupPath generates a unique backup path for the given file.
func (sm *SafetyManager) generateBackupPath(filename string) string {
	suffix := sm.options.BackupSuffix
	if suffix == "" {
		suffix = ".bak"
	}

	// Create unique backup name to avoid conflicts
	basePath := filename + suffix
	counter := 1

	for {
		if counter == 1 {
			if _, err := os.Stat(basePath); os.IsNotExist(err) {
				return basePath
			}
		} else {
			testPath := fmt.Sprintf("%s.%d", basePath, counter)
			if _, err := os.Stat(testPath); os.IsNotExist(err) {
				return testPath
			}
		}
		counter++

		// Prevent infinite loops
		if counter > 1000 {
			// Fallback to timestamp-based naming
			return fmt.Sprintf("%s.%d%s", filename, time.Now().UnixMilli(), suffix)
		}
	}
}

// validateFileStructure performs basic structural validation on the content.
func (sm *SafetyManager) validateFileStructure(content string) error {
	// Check for basic markdown structure integrity
	lines := strings.Split(content, "\n")

	// Ensure we don't have completely empty files unless original was empty
	if len(strings.TrimSpace(content)) == 0 {
		return fmt.Errorf("file appears to be empty after fixes")
	}

	// Check for malformed markdown that might indicate corruption
	openCodeBlocks := 0

	for i, line := range lines {
		// Count code block markers
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			if openCodeBlocks%2 == 0 {
				openCodeBlocks++
			} else {
				openCodeBlocks--
			}
		}

		// Check for extremely long lines that might indicate corruption
		if len(line) > 50000 { // Reasonable limit
			return fmt.Errorf("line %d is suspiciously long (%d characters), possible corruption", i+1, len(line))
		}

		// Check for null bytes or other control characters that shouldn't be in text
		for _, r := range line {
			if r == 0 {
				return fmt.Errorf("line %d contains null bytes, possible corruption", i+1)
			}
		}
	}

	// Warn about unclosed code blocks
	if openCodeBlocks%2 != 0 {
		return fmt.Errorf("unclosed code blocks detected, file structure may be compromised")
	}

	return nil
}

// validateContentIntegrity ensures that the fixed content hasn't been corrupted.
func (sm *SafetyManager) validateContentIntegrity(original, fixed string) error {
	// Check that the file size hasn't changed dramatically (could indicate corruption)
	originalSize := len(original)
	fixedSize := len(fixed)

	// Allow up to 50% size change (fixes can add/remove significant content)
	maxSizeChange := float64(originalSize) * 0.5
	sizeDiff := float64(abs(fixedSize - originalSize))

	if sizeDiff > maxSizeChange && originalSize > 100 { // Only apply to files with reasonable size
		return fmt.Errorf("file size changed too dramatically: %d -> %d bytes (%.1f%% change)",
			originalSize, fixedSize, (sizeDiff/float64(originalSize))*100)
	}

	// Check that basic content structure is preserved
	originalLines := strings.Split(original, "\n")
	fixedLines := strings.Split(fixed, "\n")

	// Count significant content lines (non-empty, non-whitespace)
	originalSignificantLines := countSignificantLines(originalLines)
	fixedSignificantLines := countSignificantLines(fixedLines)

	// Allow some variation in line count due to fixes
	if originalSignificantLines > 10 { // Only check for files with reasonable content
		maxLineChange := max(5, originalSignificantLines/2) // At least 5 lines or 50% of original
		lineDiff := abs(fixedSignificantLines - originalSignificantLines)

		if lineDiff > maxLineChange {
			return fmt.Errorf("significant content changed too much: %d -> %d lines",
				originalSignificantLines, fixedSignificantLines)
		}
	}

	return nil
}

// validateLineEndings ensures that line endings are preserved as requested.
func (sm *SafetyManager) validateLineEndings(original, fixed string) error {
	// Detect original line ending style
	originalHasCRLF := strings.Contains(original, "\r\n")
	originalHasLF := strings.Contains(original, "\n")
	originalHasCR := strings.Contains(original, "\r")

	// Detect fixed line ending style
	fixedHasCRLF := strings.Contains(fixed, "\r\n")
	fixedHasLF := strings.Contains(fixed, "\n")
	fixedHasCR := strings.Contains(fixed, "\r")

	// Check if line ending style has been preserved
	if originalHasCRLF && !fixedHasCRLF {
		return fmt.Errorf("CRLF line endings were not preserved")
	}

	if originalHasLF && !originalHasCRLF && !fixedHasLF {
		return fmt.Errorf("LF line endings were not preserved")
	}

	if originalHasCR && !originalHasCRLF && !fixedHasCR {
		return fmt.Errorf("CR line endings were not preserved")
	}

	return nil
}

// Helper functions

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func countSignificantLines(lines []string) int {
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

// calculateChecksum calculates a SHA256 checksum of the content.
func calculateChecksum(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)
}
