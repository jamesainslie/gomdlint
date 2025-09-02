package commands

import (
	"fmt"
	"os"
	"time"

	"github.com/gomdlint/gomdlint/internal/app/service"
	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/interfaces/cli/output"
	"github.com/gomdlint/gomdlint/pkg/gomdlint"
	"github.com/spf13/cobra"
)

// NewFixCommand creates the fix command for auto-fixing violations.
func NewFixCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fix [files...]",
		Short: "Automatically fix markdown violations",
		Long: `Automatically fix markdown violations where possible.

This command provides a robust, safe, and performant way to automatically fix
markdown issues. It includes:

Safety Features:
- Creates backup files before making changes
- Validates fixes to ensure file integrity  
- Atomic operations to prevent partial failures
- Recovery mechanisms in case of errors

Performance Features:
- Concurrent processing of multiple files
- Intelligent fix ordering to avoid conflicts
- Efficient batching and caching

Examples:
  gomdlint fix README.md
  gomdlint fix docs/*.md
  gomdlint fix --dry-run *.md
  gomdlint fix --no-backup --concurrency 8 *.md`,
		Args: cobra.ArbitraryArgs,
		RunE: runFix,
	}

	// Safety flags
	cmd.Flags().Bool("dry-run", false, "Show what would be fixed without making changes")
	cmd.Flags().Bool("no-backup", false, "Skip creating backup files")
	cmd.Flags().Bool("no-validate", false, "Skip validation after fixing")
	cmd.Flags().Bool("stop-on-error", false, "Stop processing on first error")

	// Performance flags
	cmd.Flags().Int("concurrency", 0, "Number of files to process concurrently (0 = auto)")
	cmd.Flags().Int("batch-size", 10, "Number of fixes to batch together")

	// File selection flags
	cmd.Flags().StringSlice("ignore", []string{}, "Ignore files matching these patterns")
	cmd.Flags().Bool("dot", false, "Include hidden files and directories")

	return cmd
}

func runFix(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Parse flags
	configFile, _ := cmd.Flags().GetString("config")
	noConfig, _ := cmd.Flags().GetBool("no-config")
	quiet, _ := cmd.Flags().GetBool("quiet")
	verbose, _ := cmd.Flags().GetBool("verbose")
	color, _ := cmd.Flags().GetBool("color")

	// Fix-specific flags
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	noBackup, _ := cmd.Flags().GetBool("no-backup")
	noValidate, _ := cmd.Flags().GetBool("no-validate")
	stopOnError, _ := cmd.Flags().GetBool("stop-on-error")
	concurrency, _ := cmd.Flags().GetInt("concurrency")
	batchSize, _ := cmd.Flags().GetInt("batch-size")
	ignorePaths, _ := cmd.Flags().GetStringSlice("ignore")
	includeDot, _ := cmd.Flags().GetBool("dot")

	// Progress tracking
	startTime := time.Now()

	// Setup themed output
	themeService := service.NewThemeService()
	themeConfig := value.NewThemeConfig()

	// Load theme configuration
	if !noConfig {
		if configSource, err := loadConfigurationSourceFromLint(configFile); err == nil && !configSource.IsDefault {
			if themeData, exists := configSource.Config["theme"]; exists {
				if themeMap, ok := themeData.(map[string]interface{}); ok {
					if themeName, ok := themeMap["theme"].(string); ok {
						themeConfig.ThemeName = themeName
					}
					if suppressEmojis, ok := themeMap["suppress_emojis"].(bool); ok {
						themeConfig.SuppressEmojis = suppressEmojis
					}
				}
			}
		}
	}

	themedOutput, err := output.NewThemedOutput(ctx, themeConfig, themeService)
	if err != nil {
		// Fall back to default theme on error
		defaultTheme := value.NewThemeConfig()
		themedOutput, _ = output.NewThemedOutput(ctx, defaultTheme, themeService)
	}
	themedOutput = themedOutput.WithColors(color)

	if !quiet {
		if dryRun {
			themedOutput.Processing("Starting dry-run fix analysis...")
		} else {
			themedOutput.Processing("Starting markdown auto-fix...")
		}
	}

	// Collect files to fix
	files, err := collectFiles(args, ignorePaths, includeDot)
	if err != nil {
		return fmt.Errorf("failed to collect files: %w", err)
	}

	if len(files) == 0 {
		if !quiet {
			fmt.Fprintf(os.Stderr, "No markdown files found.\n")
		}
		return nil
	}

	if verbose && !quiet {
		themedOutput.FileFound("Found %d files to fix", len(files))
	}

	// Prepare lint options
	lintOptions := gomdlint.LintOptions{
		Files:              files,
		Config:             make(map[string]interface{}),
		NoInlineConfig:     false,
		ResultVersion:      3,
		HandleRuleFailures: true,
	}

	// Load configuration if specified
	if !noConfig {
		configSource, err := loadConfigurationSourceFromLint(configFile)
		if err != nil && configFile != "" {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
		if !configSource.IsDefault {
			lintOptions.Config = configSource.Config

			if verbose && !quiet {
				if configSource.IsHierarchy {
					themedOutput.Info("Using hierarchical configuration from %d sources", len(configSource.Sources))
				} else if len(configSource.Sources) > 0 {
					themedOutput.Info("Using configuration from: %s", configSource.Sources[0].Path)
				}
			}
		}
	}

	// Perform initial linting to find violations
	lintResult, err := gomdlint.Lint(ctx, lintOptions)
	if err != nil {
		return fmt.Errorf("linting failed: %w", err)
	}

	if lintResult.TotalViolations == 0 {
		if !quiet {
			themedOutput.Success("No violations found - nothing to fix!")
		}
		return nil
	}

	// Create fix options
	fixOptions := service.NewFixOptions()
	fixOptions.DryRun = dryRun
	fixOptions.CreateBackups = !noBackup
	fixOptions.ValidateAfterFix = !noValidate
	fixOptions.StopOnError = stopOnError
	fixOptions.MaxConcurrency = concurrency
	fixOptions.BatchSize = batchSize
	fixOptions.ReportProgress = verbose && !quiet
	fixOptions.VerboseLogging = verbose

	// Create and configure fix engine
	fixEngine := service.NewFixEngine(fixOptions)

	// Set up progress reporting
	if verbose && !quiet {
		progressReporter := service.NewProgressReporter(fixOptions)
		progressReporter.SetCallbacks(
			func(totalFiles int) {
				themedOutput.Processing("Processing %d files...", totalFiles)
			},
			func(filename string, processed int, total int) {
				if filename != "" {
					themedOutput.Processing("Fixing %s (%d/%d)", filename, processed, total)
				}
			},
			func(processed int, total int, duration time.Duration) {
				themedOutput.Success("Processed %d/%d files in %v", processed, total, duration)
			},
		)
	}

	// Apply fixes
	if !quiet {
		if dryRun {
			themedOutput.Processing("Analyzing potential fixes...")
		} else {
			themedOutput.Processing("Applying fixes to %d files...", len(files))
		}
	}

	fixResult, err := fixEngine.FixFiles(ctx, lintResult)
	if err != nil {
		return fmt.Errorf("fix operation failed: %w", err)
	}

	// Report results
	if !quiet {
		duration := time.Since(startTime)

		if dryRun {
			themedOutput.Info("Dry-run analysis completed in %v", duration)
			themedOutput.Info("Would fix %d violations across %d files",
				fixResult.ViolationsFixed, fixResult.FilesFixed)
		} else {
			if fixResult.ViolationsFixed > 0 {
				themedOutput.Success("Fixed %d violations across %d files in %v",
					fixResult.ViolationsFixed, fixResult.FilesFixed, duration)
			} else {
				themedOutput.Info("No violations could be automatically fixed")
			}
		}

		if fixResult.FilesErrored > 0 {
			themedOutput.Warning("%d files had errors during fixing", fixResult.FilesErrored)
		}

		// Show detailed results in verbose mode
		if verbose {
			for filename, operation := range fixResult.Operations {
				status := operation.Status.String()
				switch operation.Status {
				case service.FixStatusCompleted:
					themedOutput.Success("%s: %s (%d violations fixed)", filename, status, operation.ViolationsFixed)
				case service.FixStatusFailed:
					themedOutput.Error("%s: %s - %v", filename, status, operation.Error)
				case service.FixStatusRolledBack:
					themedOutput.Warning("%s: %s (rolled back)", filename, status)
				default:
					themedOutput.Info("%s: %s", filename, status)
				}
			}
		}
	}

	// Exit with error code if there were errors
	if fixResult.FilesErrored > 0 {
		os.Exit(1)
	}

	return nil
}
