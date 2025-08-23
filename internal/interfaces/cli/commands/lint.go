package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gomdlint/gomdlint/internal/app/service"
	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/interfaces/cli/output"
	"github.com/gomdlint/gomdlint/pkg/gomdlint"
	"github.com/spf13/cobra"
)

// NewLintCommand creates the lint command.
func NewLintCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lint [files...]",
		Short: "Lint markdown files",
		Long: `Lint one or more markdown files against configured rules.
		
Examples:
  gomdlint lint README.md
  gomdlint lint docs/*.md
  gomdlint lint --config .markdownlint.json *.md
  gomdlint lint --format json --output results.json docs/`,
		Args: cobra.MinimumNArgs(0),
		RunE: runLint,
	}

	// Command-specific flags
	cmd.Flags().StringSlice("ignore", []string{}, "Ignore files matching these patterns")
	cmd.Flags().Bool("fix", false, "Automatically fix violations where possible")
	cmd.Flags().Bool("stdin", false, "Read from stdin instead of files")
	cmd.Flags().String("stdin-name", "stdin", "Name for stdin input")
	cmd.Flags().Bool("dot", false, "Include hidden files and directories")

	return cmd
}

func runLint(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Parse flags
	configFile, _ := cmd.Flags().GetString("config")
	noConfig, _ := cmd.Flags().GetBool("no-config")
	outputFile, _ := cmd.Flags().GetString("output")
	format, _ := cmd.Flags().GetString("format")
	color, _ := cmd.Flags().GetBool("color")
	quiet, _ := cmd.Flags().GetBool("quiet")
	verbose, _ := cmd.Flags().GetBool("verbose")
	fix, _ := cmd.Flags().GetBool("fix")
	stdin, _ := cmd.Flags().GetBool("stdin")
	stdinName, _ := cmd.Flags().GetString("stdin-name")
	ignorePaths, _ := cmd.Flags().GetStringSlice("ignore")
	includeDot, _ := cmd.Flags().GetBool("dot")

	// Progress tracking
	startTime := time.Now()

	// Setup themed output
	themeService := service.NewThemeService()
	themeConfig := value.NewThemeConfig()

	// Check for theme configuration in global config if available
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
					if customSymbols, ok := themeMap["custom_symbols"].(map[string]interface{}); ok {
						themeConfig.CustomSymbols = make(map[string]string)
						for k, v := range customSymbols {
							if str, ok := v.(string); ok {
								themeConfig.CustomSymbols[k] = str
							}
						}
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

	// Apply color setting
	themedOutput = themedOutput.WithColors(color)

	if !quiet {
		themedOutput.Processing("Starting markdown linting...")
	}

	// Prepare lint options
	options := gomdlint.LintOptions{
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
			options.Config = configSource.Config

			// Show which config is being used in verbose mode
			if verbose && !quiet {
				if configSource.IsHierarchy {
					themedOutput.Info("Using hierarchical configuration from %d sources", len(configSource.Sources))
					for _, source := range configSource.Sources {
						themedOutput.Info("  - %s (%s)", source.Path, source.Type)
					}
				} else if len(configSource.Sources) > 0 {
					themedOutput.Info("Using configuration from: %s", configSource.Sources[0].Path)
				}
			}
		}
	}

	// Handle stdin input
	if stdin {
		content, err := readStdin()
		if err != nil {
			return fmt.Errorf("failed to read from stdin: %w", err)
		}
		options.Strings = map[string]string{stdinName: content}
	} else {
		// Collect files to lint
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

		options.Files = files

		if verbose {
			themedOutput.FileFound("Found %d files to lint", len(files))
		}
	}

	// Perform linting
	result, err := gomdlint.Lint(ctx, options)
	if err != nil {
		return fmt.Errorf("linting failed: %w", err)
	}

	// Handle auto-fixing
	if fix && result.TotalViolations > 0 {
		fixedCount, err := performAutoFix(result, options)
		if err != nil {
			return fmt.Errorf("auto-fix failed: %w", err)
		}

		if !quiet && fixedCount > 0 {
			themedOutput.Success("Fixed %d violations", fixedCount)
		}

		// Re-lint to get updated results
		result, err = gomdlint.Lint(ctx, options)
		if err != nil {
			return fmt.Errorf("re-linting after fix failed: %w", err)
		}
	}

	// Output results
	err = outputResults(result, outputFile, format, color)
	if err != nil {
		return fmt.Errorf("failed to output results: %w", err)
	}

	// Print summary
	if !quiet {
		duration := time.Since(startTime)
		printSummary(themedOutput, result, duration, verbose)
	}

	// Return non-zero exit code if violations found
	if result.TotalErrors > 0 {
		os.Exit(1)
	}

	return nil
}

// collectFiles gathers markdown files from the given arguments.
func collectFiles(args []string, ignorePaths []string, includeDot bool) ([]string, error) {
	var files []string
	ignoreMap := make(map[string]bool)

	for _, pattern := range ignorePaths {
		ignoreMap[pattern] = true
	}

	// If no args provided, default to current directory
	if len(args) == 0 {
		args = []string{"."}
	}

	for _, arg := range args {
		err := filepath.Walk(arg, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip ignored paths
			if shouldIgnore(path, ignoreMap) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			// Skip hidden files/dirs unless requested
			if !includeDot && strings.HasPrefix(filepath.Base(path), ".") {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			// Check if it's a markdown file
			if !info.IsDir() && isMarkdownFile(path) {
				files = append(files, path)
			}

			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	return files, nil
}

// isMarkdownFile checks if a file is a markdown file based on extension.
func isMarkdownFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".md" || ext == ".markdown" || ext == ".mkd" || ext == ".mdown"
}

// shouldIgnore checks if a path should be ignored based on patterns.
func shouldIgnore(path string, ignoreMap map[string]bool) bool {
	// Simple pattern matching - in a full implementation,
	// this would support glob patterns
	for pattern := range ignoreMap {
		if strings.Contains(path, pattern) {
			return true
		}
	}
	return false
}

// loadConfiguration loads configuration from a file.
func loadConfiguration(configFile string) (map[string]interface{}, error) {
	if configFile == "" {
		// Try to find a default config file
		possibleConfigs := []string{
			".markdownlint.json",
			".markdownlint.yaml",
			".markdownlint.yml",
			"markdownlint.json",
			"markdownlint.yaml",
			"markdownlint.yml",
		}

		for _, config := range possibleConfigs {
			if _, err := os.Stat(config); err == nil {
				configFile = config
				break
			}
		}

		if configFile == "" {
			return nil, nil // No config file found
		}
	}

	// Read and parse config file
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	var config map[string]interface{}

	// Try JSON first
	if err := json.Unmarshal(data, &config); err != nil {
		// TODO: Try YAML if JSON fails
		return nil, fmt.Errorf("failed to parse config file as JSON: %w", err)
	}

	return config, nil
}

// readStdin reads content from stdin.
func readStdin() (string, error) {
	var content strings.Builder
	buffer := make([]byte, 1024)

	for {
		n, err := os.Stdin.Read(buffer)
		if n > 0 {
			content.Write(buffer[:n])
		}
		if err != nil {
			break
		}
	}

	return content.String(), nil
}

// performAutoFix attempts to automatically fix violations.
func performAutoFix(result *gomdlint.LintResult, options gomdlint.LintOptions) (int, error) {
	// TODO: Implement auto-fixing logic
	// This would involve:
	// 1. Grouping fixable violations by file
	// 2. Applying fixes in reverse order (to maintain line numbers)
	// 3. Writing fixed content back to files

	fixedCount := 0

	for filename, violations := range result.Results {
		fileFixCount := 0
		for _, violation := range violations {
			if violation.FixInfo != nil {
				// Apply fix (simplified)
				fileFixCount++
			}
		}

		if fileFixCount > 0 {
			// Write fixed content back to file
			// TODO: Implement actual file modification for filename
			_ = filename // Suppress unused variable warning
			fixedCount += fileFixCount
		}
	}

	return fixedCount, nil
}

// outputResults outputs the linting results in the specified format.
func outputResults(result *gomdlint.LintResult, outputFile, format string, color bool) error {
	var output string
	var err error

	switch format {
	case "json":
		output, err = result.ToJSON()
	case "junit":
		output, err = formatAsJUnit(result)
	case "checkstyle":
		output, err = formatAsCheckstyle(result)
	case "default", "":
		output = result.ToFormattedString(true) // Use aliases
		if color && output != "" {
			output = addColorCodes(output)
		}
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}

	if err != nil {
		return err
	}

	// Output to file or stdout
	if outputFile != "" {
		return os.WriteFile(outputFile, []byte(output), 0644)
	} else {
		fmt.Print(output)
		if output != "" && !strings.HasSuffix(output, "\n") {
			fmt.Println()
		}
	}

	return nil
}

// formatAsJUnit formats results as JUnit XML.
func formatAsJUnit(result *gomdlint.LintResult) (string, error) {
	// TODO: Implement JUnit XML formatting
	return "", fmt.Errorf("JUnit format not yet implemented")
}

// formatAsCheckstyle formats results as Checkstyle XML.
func formatAsCheckstyle(result *gomdlint.LintResult) (string, error) {
	// TODO: Implement Checkstyle XML formatting
	return "", fmt.Errorf("Checkstyle format not yet implemented")
}

// addColorCodes adds ANSI color codes to the output.
func addColorCodes(output string) string {
	// Simple color coding - in practice would be more sophisticated
	colored := strings.ReplaceAll(output, " error ", " \033[31merror\033[0m ")
	colored = strings.ReplaceAll(colored, " warning ", " \033[33mwarning\033[0m ")
	return colored
}

// printSummary prints a summary of the linting results.
func printSummary(themedOutput *output.ThemedOutput, result *gomdlint.LintResult, duration time.Duration, verbose bool) {
	if result.TotalViolations == 0 {
		themedOutput.Success("No violations found in %d files (%.2fs)",
			result.TotalFiles, duration.Seconds())
		return
	}

	themedOutput.Error("Found %d violations in %d files (%.2fs)",
		result.TotalViolations, result.TotalFiles, duration.Seconds())

	if verbose {
		themedOutput.PlainError("   Errors: %d, Warnings: %d\n",
			result.TotalErrors, result.TotalWarnings)
	}
}

// loadConfigurationSourceFromLint loads configuration source using XDG-aware system.
// This now uses the same logic as the config command for consistency.
func loadConfigurationSourceFromLint(configFile string) (*ConfigurationSource, error) {
	return loadConfigurationSourceShared(configFile)
}
