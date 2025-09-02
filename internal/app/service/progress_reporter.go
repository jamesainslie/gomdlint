package service

import (
	"context"
	"sync"
	"time"
)

// ProgressReporter handles progress reporting for fix operations.
type ProgressReporter struct {
	options *FixOptions

	// Progress tracking
	totalFiles     int
	processedFiles int
	startTime      time.Time

	// Callbacks for progress updates
	onStart    func(totalFiles int)
	onProgress func(filename string, processed int, total int)
	onComplete func(processed int, total int, duration time.Duration)

	// State management
	mu     sync.RWMutex
	active bool
	stopCh chan struct{}
	ticker *time.Ticker
}

// ProgressCallback represents a callback function for progress updates.
type ProgressCallback func(filename string, processed int, total int)

// StartCallback represents a callback function called when processing starts.
type StartCallback func(totalFiles int)

// CompleteCallback represents a callback function called when processing completes.
type CompleteCallback func(processed int, total int, duration time.Duration)

// NewProgressReporter creates a new progress reporter with the specified options.
func NewProgressReporter(options *FixOptions) *ProgressReporter {
	return &ProgressReporter{
		options: options,
		stopCh:  make(chan struct{}),
	}
}

// SetCallbacks sets the callback functions for progress reporting.
func (pr *ProgressReporter) SetCallbacks(
	onStart StartCallback,
	onProgress ProgressCallback,
	onComplete CompleteCallback,
) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	pr.onStart = onStart
	pr.onProgress = onProgress
	pr.onComplete = onComplete
}

// Start begins progress reporting for the specified number of files.
func (pr *ProgressReporter) Start(ctx context.Context, totalFiles int) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if pr.active {
		return // Already started
	}

	pr.totalFiles = totalFiles
	pr.processedFiles = 0
	pr.startTime = time.Now()
	pr.active = true

	// Call start callback
	if pr.onStart != nil {
		go pr.onStart(totalFiles)
	}

	// Start periodic progress updates if verbose logging is enabled
	if pr.options.VerboseLogging {
		pr.ticker = time.NewTicker(1 * time.Second)
		go pr.periodicUpdate(ctx)
	}
}

// ReportFile reports progress for a single file.
func (pr *ProgressReporter) ReportFile(filename string) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if !pr.active {
		return
	}

	pr.processedFiles++

	// Call progress callback
	if pr.onProgress != nil {
		go pr.onProgress(filename, pr.processedFiles, pr.totalFiles)
	}
}

// Stop stops progress reporting and calls the completion callback.
func (pr *ProgressReporter) Stop() {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if !pr.active {
		return
	}

	pr.active = false
	duration := time.Since(pr.startTime)

	// Stop ticker if running
	if pr.ticker != nil {
		pr.ticker.Stop()
		pr.ticker = nil
	}

	// Signal stop
	select {
	case pr.stopCh <- struct{}{}:
	default:
		// Channel might already be closed or full
	}

	// Call completion callback
	if pr.onComplete != nil {
		go pr.onComplete(pr.processedFiles, pr.totalFiles, duration)
	}
}

// GetProgress returns the current progress information.
func (pr *ProgressReporter) GetProgress() (processed int, total int, duration time.Duration, active bool) {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	duration = time.Since(pr.startTime)
	return pr.processedFiles, pr.totalFiles, duration, pr.active
}

// GetProgressPercentage returns the current progress as a percentage.
func (pr *ProgressReporter) GetProgressPercentage() float64 {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	if pr.totalFiles == 0 {
		return 0
	}

	return (float64(pr.processedFiles) / float64(pr.totalFiles)) * 100
}

// GetEstimatedTimeRemaining estimates how much time remains based on current progress.
func (pr *ProgressReporter) GetEstimatedTimeRemaining() time.Duration {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	if pr.processedFiles == 0 || pr.totalFiles == 0 {
		return 0
	}

	elapsed := time.Since(pr.startTime)
	rate := float64(pr.processedFiles) / elapsed.Seconds()
	remaining := pr.totalFiles - pr.processedFiles

	if rate <= 0 {
		return 0
	}

	return time.Duration(float64(remaining)/rate) * time.Second
}

// periodicUpdate runs periodic progress updates in verbose mode.
func (pr *ProgressReporter) periodicUpdate(ctx context.Context) {
	defer func() {
		if pr.ticker != nil {
			pr.ticker.Stop()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-pr.stopCh:
			return
		case <-pr.ticker.C:
			pr.mu.RLock()
			if pr.active && pr.onProgress != nil {
				processed := pr.processedFiles
				total := pr.totalFiles
				pr.mu.RUnlock()

				// Report periodic progress
				go pr.onProgress("", processed, total)
			} else {
				pr.mu.RUnlock()
			}
		}
	}
}

// IsActive returns true if progress reporting is currently active.
func (pr *ProgressReporter) IsActive() bool {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	return pr.active
}

// Reset resets the progress reporter to initial state.
func (pr *ProgressReporter) Reset() {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	pr.active = false
	pr.totalFiles = 0
	pr.processedFiles = 0
	pr.startTime = time.Time{}

	if pr.ticker != nil {
		pr.ticker.Stop()
		pr.ticker = nil
	}
}
