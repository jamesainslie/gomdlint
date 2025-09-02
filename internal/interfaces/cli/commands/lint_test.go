package commands

import (
	"bytes"
	"context"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// Test scenario structure following club/ standards
type lintCommandScenario struct {
	name             string
	args             []string
	flags            map[string]interface{}
	setupFiles       map[string]string // filename -> content
	setupConfig      string            // config file content
	expectError      bool
	expectedOutput   string
	expectedExitCode int
	expectFiles      []string // files that should be processed
}

// Test helper functions
func createTestCommand() *cobra.Command {
	cmd := NewLintCommand()

	// Add the missing flags as local flags to avoid conflicts
	cmd.Flags().StringP("config", "c", "", "Path to configuration file")
	cmd.Flags().Bool("no-config", false, "Ignore configuration files")
	cmd.Flags().StringP("output", "o", "", "Output file (default: stdout)")
	cmd.Flags().StringP("format", "f", "default", "Output format (default, json, junit, checkstyle)")
	cmd.Flags().Bool("color", true, "Enable colored output")
	cmd.Flags().Bool("quiet", false, "Suppress non-error output")
	cmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")

	return cmd
}

func TestNewLintCommand(t *testing.T) {
	t.Parallel()
	cmd := createTestCommand()

	assert.NotNil(t, cmd, "Command should not be nil")
	assert.Equal(t, "lint [files...]", cmd.Use)
	assert.Equal(t, "Lint markdown files", cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	// Check that expected flags are present
	expectedFlags := []string{"ignore", "fix", "stdin", "stdin-name", "dot"}
	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		assert.NotNil(t, flag, "Flag %s should exist", flagName)
	}
}

// Basic scenarios moved to lint_integration_test.go for better performance
// These are lightweight unit tests that don't require file I/O

// Flag scenarios moved to lint_integration_test.go for better performance

func TestLintCommand_StdinInput(t *testing.T) {
	t.Parallel()
	// This test would require more complex setup to mock stdin
	// For now, we test the flag parsing
	cmd := createTestCommand()

	flags := map[string]interface{}{
		"stdin":      true,
		"stdin-name": "custom-stdin",
	}

	// Just verify that flags can be set without error
	for name, value := range flags {
		switch v := value.(type) {
		case string:
			cmd.Flags().Set(name, v)
		case bool:
			if v {
				cmd.Flags().Set(name, "true")
			}
		}
	}

	stdinFlag, err := cmd.Flags().GetBool("stdin")
	assert.NoError(t, err)
	assert.True(t, stdinFlag)

	stdinName, err := cmd.Flags().GetString("stdin-name")
	assert.NoError(t, err)
	assert.Equal(t, "custom-stdin", stdinName)
}

// Output format tests moved to lint_integration_test.go for better performance

// File collection tests moved to lint_integration_test.go for better performance

func TestLintCommand_UtilityFunctions(t *testing.T) {
	t.Run("isMarkdownFile", func(t *testing.T) {
		t.Parallel()
		testCases := []struct {
			filename string
			expected bool
		}{
			{"test.md", true},
			{"test.markdown", true},
			{"test.mkd", true},
			{"test.mdown", true},
			{"test.MD", true}, // Case insensitive
			{"test.txt", false},
			{"test.html", false},
			{"test", false},
		}

		for _, tc := range testCases {
			result := isMarkdownFile(tc.filename)
			assert.Equal(t, tc.expected, result, "File %s should be %v", tc.filename, tc.expected)
		}
	})

	t.Run("shouldIgnore", func(t *testing.T) {
		t.Parallel()
		ignoreMap := map[string]bool{
			"node_modules/": true,
			".git/":         true,
			"temp":          true,
		}

		testCases := []struct {
			path     string
			expected bool
		}{
			{"node_modules/package.md", true},
			{".git/config", true},
			{"temp/file.md", true},
			{"src/main.md", false},
			{"docs/README.md", false},
		}

		for _, tc := range testCases {
			result := shouldIgnore(tc.path, ignoreMap)
			assert.Equal(t, tc.expected, result, "Path %s should be %v", tc.path, tc.expected)
		}
	})
}

// Configuration loading tests moved to lint_integration_test.go for better performance

// Integration tests moved to lint_integration_test.go for better performance

// Lightweight benchmark focused on command parsing logic, not I/O
func BenchmarkLintCommand_CommandCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := createTestCommand()
		_ = cmd // Use the command to avoid optimization
	}
}

// Helper function for benchmarks
func executeCommandForBench(b *testing.B, cmd *cobra.Command, args []string, flags map[string]interface{}) (*bytes.Buffer, *bytes.Buffer, error) {
	b.Helper()

	// Set up output capture
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs(args)

	// Set flags
	for name, value := range flags {
		switch v := value.(type) {
		case string:
			cmd.Flags().Set(name, v)
		case bool:
			if v {
				cmd.Flags().Set(name, "true")
			} else {
				cmd.Flags().Set(name, "false")
			}
		case []string:
			for _, val := range v {
				cmd.Flags().Set(name, val)
			}
		}
	}

	// Execute command
	err := cmd.ExecuteContext(context.Background())

	return stdout, stderr, err
}
