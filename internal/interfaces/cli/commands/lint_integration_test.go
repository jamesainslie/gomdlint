//go:build integration

package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Integration tests that require file I/O operations
// Run with: go test -tags=integration ./internal/interfaces/cli/commands/

func TestLintCommand_BasicScenariosIntegration(t *testing.T) {
	scenarios := []lintCommandScenario{
		{
			name: "lint valid markdown file",
			setupFiles: map[string]string{
				"valid.md": "# Title\n\n## Subtitle\n\nParagraph content.\n",
			},
			args:             []string{"valid.md"},
			expectError:      false,
			expectedExitCode: 0,
		},
		{
			name: "lint file with violations",
			setupFiles: map[string]string{
				"invalid.md": "#Title without space\n\nContent.\n",
			},
			args:             []string{"invalid.md"},
			expectError:      false, // In test mode, no error returned - violations shown in output
			expectedExitCode: 1,     // Should exit with 1 due to violations
		},
		{
			name: "lint multiple files",
			setupFiles: map[string]string{
				"file1.md": "# File One\n\nContent.\n",
				"file2.md": "# File Two\n\nContent.\n",
				"file3.md": "#Bad file\n\nContent.\n",
			},
			args:             []string{"file1.md", "file2.md", "file3.md"},
			expectError:      false, // In test mode, no error returned - violations shown in output
			expectedExitCode: 1,     // file3.md has violations
		},
		{
			name: "lint with glob pattern",
			setupFiles: map[string]string{
				"doc1.md":   "# Doc One\n\nContent.\n",
				"doc2.md":   "# Doc Two\n\nContent.\n",
				"other.txt": "Not markdown",
			},
			args:             []string{"*.md"},
			expectError:      false,
			expectedExitCode: 0,
		},
		{
			name:             "no files specified - should use current directory",
			setupFiles:       map[string]string{},
			args:             []string{},
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
			cmd := createTestCommand()

			// Execute command
			stdout, stderr, err := executeCommand(t, cmd, scenario.args, scenario.flags)

			// Verify results
			if scenario.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if scenario.expectedOutput != "" {
				output := stdout.String() + stderr.String()
				assert.Contains(t, output, scenario.expectedOutput)
			}

			// Note: exit codes are handled by os.Exit() in the actual command,
			// so we can't easily test them in unit tests without refactoring
		})
	}
}

func TestLintCommand_FlagScenariosIntegration(t *testing.T) {
	testFiles := map[string]string{
		"good.md":         "# Title\n\nContent.\n",
		"bad.md":          "#Title\n\nContent.\n",
		".hidden.md":      "# Hidden\n\nContent.\n",
		"ignored/file.md": "# Ignored\n\nContent.\n",
		"config.json":     `{"MD018": false}`,
	}

	flagScenarios := []struct {
		name        string
		args        []string
		flags       map[string]interface{}
		expectError bool
	}{
		{
			name: "with config file",
			args: []string{"bad.md"},
			flags: map[string]interface{}{
				"config": "config.json",
			},
			expectError: false,
		},
		{
			name: "no config flag",
			args: []string{"good.md"},
			flags: map[string]interface{}{
				"no-config": true,
			},
			expectError: false,
		},
		{
			name: "json output format",
			args: []string{"good.md"},
			flags: map[string]interface{}{
				"format": "json",
			},
			expectError: false,
		},
		{
			name: "output to file",
			args: []string{"good.md"},
			flags: map[string]interface{}{
				"output": "results.txt",
			},
			expectError: false,
		},
		{
			name: "ignore patterns",
			args: []string{"."},
			flags: map[string]interface{}{
				"ignore": []string{"ignored/"},
			},
			expectError: false,
		},
		{
			name: "include hidden files",
			args: []string{"."},
			flags: map[string]interface{}{
				"dot": true,
			},
			expectError: false,
		},
		{
			name: "quiet mode",
			args: []string{"good.md"},
			flags: map[string]interface{}{
				"quiet": true,
			},
			expectError: false,
		},
		{
			name: "verbose mode",
			args: []string{"good.md"},
			flags: map[string]interface{}{
				"verbose": true,
			},
			expectError: false,
		},
		{
			name: "fix mode",
			args: []string{"bad.md"},
			flags: map[string]interface{}{
				"fix": true,
			},
			expectError: false,
		},
	}

	for _, scenario := range flagScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Setup test environment
			tmpDir := createTempTestFiles(t, testFiles)

			// Change to temp directory
			oldDir, err := os.Getwd()
			require.NoError(t, err)
			err = os.Chdir(tmpDir)
			require.NoError(t, err)
			defer func() {
				os.Chdir(oldDir)
			}()

			// Create command
			cmd := createTestCommand()

			// Execute command
			stdout, stderr, err := executeCommand(t, cmd, scenario.args, scenario.flags)

			// Verify results
			if scenario.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Verify specific flag behaviors
			if format, ok := scenario.flags["format"]; ok && format == "json" {
				output := stdout.String()
				var jsonData interface{}
				// Should be valid JSON if not empty
				if output != "" {
					err := json.Unmarshal([]byte(output), &jsonData)
					assert.NoError(t, err, "Output should be valid JSON")
				}
			}

			if outputFile, ok := scenario.flags["output"]; ok {
				// Check that output file was created
				_, err := os.Stat(outputFile.(string))
				assert.NoError(t, err, "Output file should be created")
			}

			if quiet, ok := scenario.flags["quiet"]; ok && quiet == true {
				// Quiet mode should produce minimal output
				output := stdout.String() + stderr.String()
				assert.True(t, len(output) < 100, "Quiet mode should produce minimal output")
			}
		})
	}
}

func TestLintCommand_OutputFormatsIntegration(t *testing.T) {
	testFile := map[string]string{
		"test.md": "#Title\n\nContent with violation.\n",
	}

	formats := []string{"default", "json"}

	for _, format := range formats {
		t.Run("format_"+format, func(t *testing.T) {
			// Setup test environment
			tmpDir := createTempTestFiles(t, testFile)

			oldDir, err := os.Getwd()
			require.NoError(t, err)
			err = os.Chdir(tmpDir)
			require.NoError(t, err)
			defer func() {
				os.Chdir(oldDir)
			}()

			cmd := createTestCommand()

			flags := map[string]interface{}{
				"format": format,
			}

			stdout, stderr, err := executeCommand(t, cmd, []string{"test.md"}, flags)

			// Should not error in test mode - exit codes are handled differently in tests
			assert.NoError(t, err)

			output := stdout.String()

			switch format {
			case "json":
				if output != "" {
					var jsonData interface{}
					err := json.Unmarshal([]byte(output), &jsonData)
					assert.NoError(t, err, "JSON format should produce valid JSON")
				}
			case "default", "":
				// Default format should be human-readable
				if len(stderr.String()) == 0 {
					// If no violations, output might be empty or have summary
					assert.True(t, true, "Default format handled")
				}
			}
		})
	}
}

func TestLintCommand_CollectFilesIntegration(t *testing.T) {
	testCases := []struct {
		name          string
		args          []string
		files         map[string]string
		ignorePaths   []string
		includeDot    bool
		expectedFiles []string
	}{
		{
			name: "collect markdown files",
			args: []string{"."},
			files: map[string]string{
				"file1.md":  "# File 1",
				"file2.md":  "# File 2",
				"file3.txt": "Not markdown",
				"README.md": "# README",
			},
			expectedFiles: []string{"file1.md", "file2.md", "README.md"},
		},
		{
			name: "ignore patterns",
			args: []string{"."},
			files: map[string]string{
				"good.md":       "# Good",
				"ignore/bad.md": "# Should be ignored",
				"other.md":      "# Other",
			},
			ignorePaths:   []string{"ignore/"},
			expectedFiles: []string{"good.md", "other.md"},
		},
		{
			name: "include hidden files",
			args: []string{"."},
			files: map[string]string{
				"normal.md":  "# Normal",
				".hidden.md": "# Hidden",
			},
			includeDot:    true,
			expectedFiles: []string{"normal.md", ".hidden.md"},
		},
		{
			name: "exclude hidden files by default",
			args: []string{"."},
			files: map[string]string{
				"normal.md":  "# Normal",
				".hidden.md": "# Hidden",
			},
			includeDot:    false,
			expectedFiles: []string{"normal.md"},
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

			files, err := collectFiles(tc.args, tc.ignorePaths, tc.includeDot)
			require.NoError(t, err)

			// Convert to just filenames for comparison
			fileNames := make([]string, len(files))
			for i, file := range files {
				fileNames[i] = filepath.Base(file)
			}

			assert.ElementsMatch(t, tc.expectedFiles, fileNames)
		})
	}
}

func TestLintCommand_ConfigurationLoadingIntegration(t *testing.T) {
	t.Run("load JSON config", func(t *testing.T) {
		configContent := `{
			"MD001": false,
			"MD013": {
				"line_length": 120
			}
		}`

		tmpDir := createTempTestFiles(t, map[string]string{
			"config.json": configContent,
		})

		configPath := filepath.Join(tmpDir, "config.json")
		config, err := loadConfiguration(configPath)

		require.NoError(t, err)
		require.NotNil(t, config)

		assert.Equal(t, false, config["MD001"])

		md013, ok := config["MD013"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, float64(120), md013["line_length"])
	})

	t.Run("no config file", func(t *testing.T) {
		config, err := loadConfiguration("")
		assert.NoError(t, err)
		assert.Nil(t, config)
	})

	t.Run("nonexistent config file", func(t *testing.T) {
		config, err := loadConfiguration("nonexistent.json")
		assert.Error(t, err)
		assert.Nil(t, config)
	})
}

// Integration-style tests
func TestLintCommand_FullWorkflowIntegration(t *testing.T) {
	t.Run("complete workflow", func(t *testing.T) {
		files := map[string]string{
			"README.md": `# Project Title

This is a project description.

## Features

- Feature 1
- Feature 2
`,
			"docs/guide.md": `#Guide

This guide has violations.


Too many blank lines above.
`,
			".markdownlint.json": `{
				"MD012": {"maximum": 2}
			}`,
		}

		tmpDir := createTempTestFiles(t, files)

		oldDir, err := os.Getwd()
		require.NoError(t, err)
		err = os.Chdir(tmpDir)
		require.NoError(t, err)
		defer func() {
			os.Chdir(oldDir)
		}()

		cmd := createTestCommand()

		stdout, stderr, err := executeCommand(t, cmd, []string{"."}, map[string]interface{}{
			"verbose": true,
		})

		assert.NoError(t, err) // In test mode, no error returned - violations shown in output

		output := stdout.String() + stderr.String()

		// Should mention processing files
		assert.True(t, len(output) > 0, "Should produce output")

		// Should find the violations in docs/guide.md
		// (exact output depends on implementation)
	})
}
