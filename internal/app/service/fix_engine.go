package service

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// FixEngine provides a robust, safe, and performant markdown fix service.
// It orchestrates the entire fix process with safety mechanisms and concurrency.
type FixEngine struct {
	safetyManager    *SafetyManager
	fixCoordinator   *FixCoordinator
	fileManager      *FileManager
	progressReporter *ProgressReporter

	// Configuration
	options *FixOptions

	// Concurrency control
	maxConcurrency int
	semaphore      chan struct{}

	// State tracking
	mu               sync.RWMutex
	activeOperations map[string]*FixOperation
}

// NewFixEngine creates a new fix engine with the specified options.
func NewFixEngine(options *FixOptions) *FixEngine {
	if options == nil {
		options = NewFixOptions()
	}

	maxConcurrency := options.MaxConcurrency
	if maxConcurrency <= 0 {
		maxConcurrency = 4
	}

	return &FixEngine{
		safetyManager:    NewSafetyManager(options),
		fixCoordinator:   NewFixCoordinator(options),
		fileManager:      NewFileManager(options),
		progressReporter: NewProgressReporter(options),
		options:          options,
		maxConcurrency:   maxConcurrency,
		semaphore:        make(chan struct{}, maxConcurrency),
		activeOperations: make(map[string]*FixOperation),
	}
}

// FixFiles applies fixes to the specified files based on linting results.
func (fe *FixEngine) FixFiles(ctx context.Context, results interface{}) (*FixResult, error) {
	if results == nil {
		return &FixResult{
			TotalFiles:   0,
			FilesFixed:   0,
			FilesErrored: 0,
			Operations:   make(map[string]*FixOperation),
			DryRun:       fe.options.DryRun,
		}, nil
	}

	// Initialize result tracking
	fixResult := &FixResult{
		TotalFiles: 0,
		Operations: make(map[string]*FixOperation),
		DryRun:     fe.options.DryRun,
	}

	// Start progress reporting
	if fe.options.ReportProgress {
		fe.progressReporter.Start(ctx, 0)
		defer fe.progressReporter.Stop()
	}

	// Group fixable violations by file
	fixableFiles := fe.groupFixableViolations(results)

	if len(fixableFiles) == 0 {
		return fixResult, nil
	}

	// Process files concurrently
	return fe.processFiles(ctx, fixableFiles, fixResult)
}

// groupFixableViolations groups violations by file, filtering only fixable ones.
func (fe *FixEngine) groupFixableViolations(results interface{}) map[string][]interface{} {
	// For now, return empty map - this will be properly implemented later
	return make(map[string][]interface{})
}

// processFiles processes multiple files concurrently with proper error handling.
func (fe *FixEngine) processFiles(ctx context.Context, fixableFiles map[string][]interface{}, result *FixResult) (*FixResult, error) {
	var wg sync.WaitGroup
	errorCh := make(chan error, len(fixableFiles))

	// Process each file
	for filename, violations := range fixableFiles {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case fe.semaphore <- struct{}{}: // Acquire semaphore
		}

		wg.Add(1)
		go func(fn string, viols []interface{}) {
			defer wg.Done()
			defer func() { <-fe.semaphore }() // Release semaphore

			if err := fe.processFile(ctx, fn, viols, result); err != nil {
				if fe.options.StopOnError {
					errorCh <- err
					return
				}

				// Log error but continue processing
				fe.mu.Lock()
				result.Errors = append(result.Errors, err)
				result.FilesErrored++
				fe.mu.Unlock()
			}
		}(filename, violations)
	}

	// Wait for all operations to complete
	go func() {
		wg.Wait()
		close(errorCh)
	}()

	// Check for early termination errors
	for err := range errorCh {
		if err != nil && fe.options.StopOnError {
			return result, fmt.Errorf("fix operation failed: %w", err)
		}
	}

	return result, nil
}

// processFile processes a single file, applying all its fixes atomically.
func (fe *FixEngine) processFile(ctx context.Context, filename string, violations []interface{}, result *FixResult) error {
	// Create and track operation
	operation := &FixOperation{
		Filename:  filename,
		Status:    FixStatusPending,
		StartTime: getCurrentTimestamp(),
	}

	fe.mu.Lock()
	fe.activeOperations[filename] = operation
	result.Operations[filename] = operation
	fe.mu.Unlock()

	operation.Status = FixStatusRunning

	// Report progress
	if fe.options.ReportProgress {
		fe.progressReporter.ReportFile(filename)
	}

	defer func() {
		operation.EndTime = getCurrentTimestamp()
		fe.mu.Lock()
		delete(fe.activeOperations, filename)
		fe.mu.Unlock()
	}()

	// Safety preparation
	if err := fe.safetyManager.PrepareFile(ctx, filename, operation); err != nil {
		operation.Status = FixStatusFailed
		operation.Error = err
		return fmt.Errorf("failed to prepare file for fixing: %w", err)
	}

	// Read original content
	originalContent, err := fe.fileManager.ReadFile(ctx, filename)
	if err != nil {
		operation.Status = FixStatusFailed
		operation.Error = err
		return fmt.Errorf("failed to read file %s: %w", filename, err)
	}
	operation.OriginalContent = originalContent

	// Apply fixes using the coordinator
	fixedContent, fixedCount, err := fe.fixCoordinator.ApplyFixes(ctx, originalContent, violations, filename)
	if err != nil {
		operation.Status = FixStatusFailed
		operation.Error = err

		// Attempt recovery
		if recoveryErr := fe.safetyManager.RecoverFile(ctx, filename, operation); recoveryErr != nil {
			return fmt.Errorf("failed to apply fixes and recovery failed: %w (original error: %v)", recoveryErr, err)
		}
		operation.Status = FixStatusRolledBack
		return fmt.Errorf("failed to apply fixes to %s (rolled back): %w", filename, err)
	}

	operation.FixedContent = fixedContent
	operation.ViolationsFixed = fixedCount

	// Write fixed content (or skip in dry run)
	if !fe.options.DryRun {
		if err := fe.fileManager.WriteFile(ctx, filename, fixedContent); err != nil {
			operation.Status = FixStatusFailed
			operation.Error = err

			// Attempt recovery
			if recoveryErr := fe.safetyManager.RecoverFile(ctx, filename, operation); recoveryErr != nil {
				return fmt.Errorf("failed to write fixed content and recovery failed: %w (original error: %v)", recoveryErr, err)
			}
			operation.Status = FixStatusRolledBack
			return fmt.Errorf("failed to write fixed content to %s (rolled back): %w", filename, err)
		}

		// Validate fixes if enabled
		if fe.options.ValidateAfterFix {
			if err := fe.safetyManager.ValidateFile(ctx, filename, operation); err != nil {
				operation.Status = FixStatusFailed
				operation.Error = err

				// Attempt recovery
				if recoveryErr := fe.safetyManager.RecoverFile(ctx, filename, operation); recoveryErr != nil {
					return fmt.Errorf("validation failed and recovery failed: %w (original error: %v)", recoveryErr, err)
				}
				operation.Status = FixStatusRolledBack
				return fmt.Errorf("validation failed for %s (rolled back): %w", filename, err)
			}
		}
	}

	operation.Status = FixStatusCompleted

	// Update results
	fe.mu.Lock()
	result.FilesFixed++
	result.ViolationsFixed += fixedCount
	fe.mu.Unlock()

	return nil
}

// GetActiveOperations returns the currently active fix operations.
func (fe *FixEngine) GetActiveOperations() map[string]*FixOperation {
	fe.mu.RLock()
	defer fe.mu.RUnlock()

	operations := make(map[string]*FixOperation)
	for k, v := range fe.activeOperations {
		opCopy := *v
		operations[k] = &opCopy
	}

	return operations
}

// Stop gracefully stops the fix engine, allowing active operations to complete.
func (fe *FixEngine) Stop(ctx context.Context) error {
	// Wait for all active operations to complete or context to be cancelled
	for {
		fe.mu.RLock()
		activeCount := len(fe.activeOperations)
		fe.mu.RUnlock()

		if activeCount == 0 {
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Brief sleep to avoid busy waiting
			time.Sleep(10 * time.Millisecond)
		}
	}

	return nil
}

// getCurrentTimestamp returns the current timestamp in milliseconds.
func getCurrentTimestamp() int64 {
	return time.Now().UnixMilli()
}
