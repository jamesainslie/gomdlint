package service

// FixOptions configures how fixes are applied.
type FixOptions struct {
	// Safety settings
	CreateBackups    bool   `json:"create_backups"`
	BackupSuffix     string `json:"backup_suffix"`
	ValidateAfterFix bool   `json:"validate_after_fix"`
	AtomicOperations bool   `json:"atomic_operations"`

	// Performance settings
	MaxConcurrency int `json:"max_concurrency"`
	BatchSize      int `json:"batch_size"`

	// Behavior settings
	DryRun              bool `json:"dry_run"`
	StopOnError         bool `json:"stop_on_error"`
	OverwriteFiles      bool `json:"overwrite_files"`
	PreserveLineEndings bool `json:"preserve_line_endings"`

	// Reporting settings
	ReportProgress bool `json:"report_progress"`
	VerboseLogging bool `json:"verbose_logging"`
}

// NewFixOptions creates default fix options.
func NewFixOptions() *FixOptions {
	return &FixOptions{
		CreateBackups:       true,
		BackupSuffix:        ".bak",
		ValidateAfterFix:    true,
		AtomicOperations:    true,
		MaxConcurrency:      4,
		BatchSize:           10,
		DryRun:              false,
		StopOnError:         false,
		OverwriteFiles:      true,
		PreserveLineEndings: true,
		ReportProgress:      true,
		VerboseLogging:      false,
	}
}

// FixOperationStatus represents the status of a fix operation.
type FixOperationStatus int

const (
	FixStatusPending FixOperationStatus = iota
	FixStatusRunning
	FixStatusCompleted
	FixStatusFailed
	FixStatusRolledBack
)

// String returns the string representation of the fix operation status.
func (s FixOperationStatus) String() string {
	switch s {
	case FixStatusPending:
		return "pending"
	case FixStatusRunning:
		return "running"
	case FixStatusCompleted:
		return "completed"
	case FixStatusFailed:
		return "failed"
	case FixStatusRolledBack:
		return "rolled_back"
	default:
		return "unknown"
	}
}

// FixOperation represents a single file fix operation in progress.
type FixOperation struct {
	Filename        string
	Status          FixOperationStatus
	StartTime       int64
	EndTime         int64
	ViolationsFixed int
	Error           error

	// Safety tracking
	BackupPath      string
	OriginalContent string
	FixedContent    string
}

// FixResult represents the result of a fix operation.
type FixResult struct {
	TotalFiles      int                      `json:"total_files"`
	FilesFixed      int                      `json:"files_fixed"`
	FilesErrored    int                      `json:"files_errored"`
	ViolationsFixed int                      `json:"violations_fixed"`
	Operations      map[string]*FixOperation `json:"operations"`
	Errors          []error                  `json:"errors,omitempty"`
	DryRun          bool                     `json:"dry_run"`
}
