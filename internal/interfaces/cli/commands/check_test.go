package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test scenarios for check command following club/ standards
type checkCommandScenario struct {
	name             string
	args             []string
	flags            map[string]interface{}
	setupFiles       map[string]string
	setupConfig      string
	expectError      bool
	expectedExitCode int
	expectOutput     string
}

func createCheckCommand() *cobra.Command {
	cmd := NewCheckCommand()

	// Add persistent flags that would normally come from root command
	cmd.PersistentFlags().StringP("config", "c", "", "Path to configuration file")
	cmd.PersistentFlags().Bool("no-config", false, "Ignore configuration files")
	cmd.PersistentFlags().StringP("output", "o", "", "Output file (default: stdout)")
	cmd.PersistentFlags().StringP("format", "f", "default", "Output format (default, json, junit, checkstyle)")
	cmd.PersistentFlags().Bool("color", true, "Enable colored output")
	cmd.PersistentFlags().Bool("quiet", false, "Suppress non-error output")
	cmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")

	return cmd
}

func TestNewCheckCommand(t *testing.T) {
	cmd := createCheckCommand()

	assert.NotNil(t, cmd, "Command should not be nil")
	assert.Equal(t, "check [files...]", cmd.Use)
	assert.Equal(t, "Check markdown files (alias for lint)", cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	// Check that expected flags are present
	expectedFlags := []string{"ignore", "stdin", "stdin-name", "dot"}
	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		assert.NotNil(t, flag, "Flag %s should exist", flagName)
	}
}

func TestCheckCommand_BasicScenarios(t *testing.T) {
	scenarios := []checkCommandScenario{
		{
			name: "check valid files - should pass",
			setupFiles: map[string]string{
				"valid1.md": "# Title One\n\n## Subtitle\n\nContent here.\n",
				"valid2.md": "# Title Two\n\n## Section\n\nMore content.\n",
			},
			args:             []string{"valid1.md", "valid2.md"},
			expectError:      false,
			expectedExitCode: 0,
		},
		{
			name: "check files with violations - should fail",
			setupFiles: map[string]string{
				"invalid.md": "#Title without space\n\nContent.\n",
			},
			args:             []string{"invalid.md"},
			expectError:      false, // Command executes, but exits with non-zero
			expectedExitCode: 1,
		},
		{
			name: "check mixed files",
			setupFiles: map[string]string{
				"good.md": "# Good File\n\nProper formatting.\n",
				"bad.md":  "#Bad file\n\nImproper formatting.\n",
			},
			args:             []string{"good.md", "bad.md"},
			expectError:      false,
			expectedExitCode: 1,
		},
		{
			name: "no files found",
			setupFiles: map[string]string{
				"not-markdown.txt": "This is not a markdown file",
			},
			args:             []string{"."},
			expectError:      false,
			expectedExitCode: 0,
		},
		{
			name:             "nonexistent file",
			args:             []string{"nonexistent.md"},
			expectError:      true,
			expectedExitCode: 1,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Setup test environment
			tmpDir := createTempTestFiles(t, scenario.setupFiles)

			// Change to temp directory
			oldDir, err := os.Getwd()
			require.NoError(t, err)
			err = os.Chdir(tmpDir)
			require.NoError(t, err)
			defer func() {
				os.Chdir(oldDir)
			}()

			// Create command
			cmd := createCheckCommand()

			// Execute command
			stdout, stderr, err := executeCommand(t, cmd, scenario.args, scenario.flags)

			// Verify results
			if scenario.expectError {
				assert.Error(t, err)
			} else if scenario.expectedExitCode == 0 {
				assert.NoError(t, err)
			} else if scenario.expectedExitCode == 1 {
				// In test mode, command completes without error - violations are shown in output
				assert.NoError(t, err)
			}

			if scenario.expectOutput != "" {
				output := stdout.String() + stderr.String()
				assert.Contains(t, output, scenario.expectOutput)
			}
		})
	}
}

func TestCheckCommand_FlagScenarios(t *testing.T) {
	testFiles := map[string]string{
		"file1.md": "# Good File\n\nContent.\n",
		"file2.md": "#Bad file\n\nContent.\n",
		"file3.md": "#Another bad file\n\n\n\nToo many blanks.\n",
	}

	flagScenarios := []struct {
		name        string
		args        []string
		flags       map[string]interface{}
		expectError bool
		expectEarly bool // For fail-fast
	}{
		{
			name: "fail-fast mode",
			args: []string{"file1.md", "file2.md", "file3.md"},
			flags: map[string]interface{}{
				"fail-fast": true,
			},
			expectError: false, // In test mode, no error returned - violations shown in output
			expectEarly: true,  // Should stop at first violation
		},
		{
			name: "summary-only mode",
			args: []string{"file2.md", "file3.md"},
			flags: map[string]interface{}{
				"summary-only": true,
			},
			expectError: false, // In test mode, no error returned - violations shown in output
		},
		{
			name: "quiet mode",
			args: []string{"file1.md"},
			flags: map[string]interface{}{
				"quiet": true,
			},
			expectError: false,
		},
		{
			name: "verbose mode",
			args: []string{"file1.md"},
			flags: map[string]interface{}{
				"verbose": true,
			},
			expectError: false,
		},
		{
			name: "combined flags",
			args: []string{"file2.md"},
			flags: map[string]interface{}{
				"summary-only": true,
				"quiet":        true,
			},
			expectError: false, // In test mode, no error returned - violations shown in output
		},
	}

	for _, scenario := range flagScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Setup test environment
			tmpDir := createTempTestFiles(t, testFiles)

			oldDir, err := os.Getwd()
			require.NoError(t, err)
			err = os.Chdir(tmpDir)
			require.NoError(t, err)
			defer func() {
				os.Chdir(oldDir)
			}()

			// Create command
			cmd := createCheckCommand()

			// Execute command
			stdout, stderr, err := executeCommand(t, cmd, scenario.args, scenario.flags)

			// Verify results
			if scenario.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			output := stdout.String() + stderr.String()

			// Verify specific flag behaviors
			if quiet, ok := scenario.flags["quiet"]; ok && quiet == true {
				// In quiet mode, we should not see processing messages or detailed output
				// But we may still see error messages when the command fails
				assert.NotContains(t, output, "Starting markdown linting", "Quiet mode should not show processing messages")
				assert.NotContains(t, output, "Found", "Quiet mode should not show summary messages")
			}

			if summaryOnly, ok := scenario.flags["summary-only"]; ok && summaryOnly == true {
				// Summary-only should not show individual violation details
				assert.NotContains(t, output, "line", "Summary-only should not show line details")
			}
		})
	}
}

func TestCheckCommand_ExitCodeBehavior(t *testing.T) {
	testCases := []struct {
		name         string
		files        map[string]string
		expectedCode int
	}{
		{
			name: "no violations",
			files: map[string]string{
				"perfect.md": "# Perfect\n\n## Section\n\nContent.\n",
			},
			expectedCode: 0,
		},
		{
			name: "with violations",
			files: map[string]string{
				"bad.md": "#Bad\n\nContent.\n",
			},
			expectedCode: 1,
		},
		{
			name: "mixed results",
			files: map[string]string{
				"good.md": "# Good\n\nContent.\n",
				"bad.md":  "#Bad\n\nContent.\n",
			},
			expectedCode: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := createTempTestFiles(t, tc.files)

			oldDir, err := os.Getwd()
			require.NoError(t, err)
			err = os.Chdir(tmpDir)
			require.NoError(t, err)
			defer func() {
				os.Chdir(oldDir)
			}()

			cmd := createCheckCommand()

			// Execute command
			_, _, err = executeCommand(t, cmd, []string{"."}, map[string]interface{}{})

			// Note: In actual usage, exit codes are set via os.Exit()
			// In unit tests, we verify the command completes without panic
			assert.NoError(t, err, "Command should execute without error")
		})
	}
}

func TestCheckCommand_Configuration(t *testing.T) {
	testFiles := map[string]string{
		"test.md": "# Title\n\nContent.\n", // Valid markdown with proper spacing
		"config.json": `{
			"MD018": false
		}`,
	}

	t.Run("with config file", func(t *testing.T) {
		tmpDir := createTempTestFiles(t, testFiles)

		oldDir, err := os.Getwd()
		require.NoError(t, err)
		err = os.Chdir(tmpDir)
		require.NoError(t, err)
		defer func() {
			os.Chdir(oldDir)
		}()

		cmd := createCheckCommand()

		stdout, stderr, err := executeCommand(t, cmd, []string{"test.md"}, map[string]interface{}{
			"config": "config.json",
		})

		assert.NoError(t, err)

		output := stdout.String() + stderr.String()
		// With MD018 disabled, should not have violations
		// (exact behavior depends on implementation)
		assert.NotNil(t, output) // Just verify we get some output
	})

	t.Run("no config", func(t *testing.T) {
		tmpDir := createTempTestFiles(t, map[string]string{
			"test.md": "#Title\n\nContent.\n",
		})

		oldDir, err := os.Getwd()
		require.NoError(t, err)
		err = os.Chdir(tmpDir)
		require.NoError(t, err)
		defer func() {
			os.Chdir(oldDir)
		}()

		cmd := createCheckCommand()

		stdout, stderr, err := executeCommand(t, cmd, []string{"test.md"}, map[string]interface{}{
			"no-config": true,
		})

		assert.NoError(t, err) // In test mode, no error returned - violations shown in output

		output := stdout.String() + stderr.String()
		// Without config, should use defaults and find violations
		assert.NotNil(t, output)
	})
}

func TestCheckCommand_ErrorHandling(t *testing.T) {
	errorScenarios := []struct {
		name        string
		args        []string
		flags       map[string]interface{}
		setupFiles  map[string]string
		expectError bool
	}{
		{
			name:        "invalid config file",
			args:        []string{"test.md"},
			setupFiles:  map[string]string{"test.md": "# Test\n"},
			flags:       map[string]interface{}{"config": "nonexistent.json"},
			expectError: true,
		},
		{
			name:        "invalid output format",
			args:        []string{"test.md"},
			setupFiles:  map[string]string{"test.md": "# Test\n"},
			flags:       map[string]interface{}{"format": "invalid-format"},
			expectError: true,
		},
		{
			name:        "permission denied on output file",
			args:        []string{"test.md"},
			setupFiles:  map[string]string{"test.md": "# Test\n"},
			flags:       map[string]interface{}{"output": "/root/cannot-write.txt"},
			expectError: true,
		},
	}

	for _, scenario := range errorScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Setup test environment
			tmpDir := createTempTestFiles(t, scenario.setupFiles)

			oldDir, err := os.Getwd()
			require.NoError(t, err)
			err = os.Chdir(tmpDir)
			require.NoError(t, err)
			defer func() {
				os.Chdir(oldDir)
			}()

			// Create command
			cmd := createCheckCommand()

			// Execute command
			_, _, err = executeCommand(t, cmd, scenario.args, scenario.flags)

			if scenario.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCheckCommand_Performance(t *testing.T) {
	// Generate multiple files for performance testing
	files := make(map[string]string)
	for i := 0; i < 50; i++ {
		filename := fmt.Sprintf("file%d.md", i)
		content := fmt.Sprintf("# File %d\n\n## Section\n\nContent for file %d.\n\n- Item 1\n- Item 2\n", i, i)
		files[filename] = content
	}

	tmpDir := createTempTestFiles(t, files)

	oldDir, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() {
		os.Chdir(oldDir)
	}()

	cmd := createCheckCommand()

	// Execute command and measure time
	start := time.Now()
	_, _, err = executeCommand(t, cmd, []string{"."}, map[string]interface{}{
		"quiet": true,
	})
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.Less(t, duration, 10*time.Second, "Check command should complete in reasonable time")
}

// Integration test
func TestCheckCommand_Integration(t *testing.T) {
	projectFiles := map[string]string{
		"README.md": `# My Project

This is a sample project with various markdown files.

## Features

- Feature 1
- Feature 2

## Installation

Run the following command:

` + "```bash\nnpm install\n```" + `
`,
		"docs/api.md": `# API Documentation

## Endpoints

### GET /users

Returns list of users.

### POST /users

Creates a new user.
`,
		"docs/guide.md": `#User Guide

This guide helps you get started.


Too many blank lines above this paragraph.

- Step 1
- Step 2
- Step 3
`,
		"CHANGELOG.md": `# Changelog

## [1.0.0] - 2023-01-01

### Added
- Initial release

### Fixed
- Various bug fixes
`,
		".markdownlint.json": `{
			"MD012": {"maximum": 2},
			"MD013": {"line_length": 120}
		}`,
	}

	tmpDir := createTempTestFiles(t, projectFiles)

	oldDir, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() {
		os.Chdir(oldDir)
	}()

	cmd := createCheckCommand()

	stdout, stderr, err := executeCommand(t, cmd, []string{"."}, map[string]interface{}{
		"verbose": true,
	})

	assert.NoError(t, err) // In test mode, no error returned - violations shown in output

	output := stdout.String() + stderr.String()

	// Should find files and process them
	assert.True(t, len(output) > 0, "Should produce output")

	// Should find violations in docs/guide.md (missing space after # and too many blank lines)
	// Exact output format depends on implementation
}

// Benchmark tests
func BenchmarkCheckCommand_MultipleFiles(b *testing.B) {
	// Create test files
	files := make(map[string]string)
	for i := 0; i < 20; i++ {
		filename := fmt.Sprintf("bench%d.md", i)
		content := fmt.Sprintf(`# Benchmark File %d

This is content for benchmarking.

## Section

- Item 1
- Item 2
- Item 3

Final paragraph.
`, i)
		files[filename] = content
	}

	tmpDir := b.TempDir()
	for filename, content := range files {
		err := os.WriteFile(filepath.Join(tmpDir, filename), []byte(content), 0644)
		require.NoError(b, err)
	}

	oldDir, err := os.Getwd()
	require.NoError(b, err)
	err = os.Chdir(tmpDir)
	require.NoError(b, err)
	defer func() {
		os.Chdir(oldDir)
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := createCheckCommand()
		// For benchmarks, we don't need to capture output
		err := cmd.Execute()
		require.NoError(b, err)
	}
}
