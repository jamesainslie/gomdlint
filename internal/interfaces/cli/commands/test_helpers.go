package commands

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// createTempTestFiles creates temporary test files for command testing
func createTempTestFiles(t *testing.T, files map[string]string) string {
	t.Helper()

	tmpDir := t.TempDir()
	for filename, content := range files {
		filePath := filepath.Join(tmpDir, filename)

		// Create directory if needed
		dir := filepath.Dir(filePath)
		if dir != tmpDir {
			err := os.MkdirAll(dir, 0755)
			require.NoError(t, err)
		}

		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)
	}

	return tmpDir
}

// executeCommand executes a command and captures output for testing
func executeCommand(t *testing.T, cmd *cobra.Command, args []string, flags map[string]interface{}) (*bytes.Buffer, *bytes.Buffer, error) {
	t.Helper()

	// Set up output capture
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	// Set args - check if this is a config command with a stored subcommand
	if cmd.Use == "config" && cmd.Annotations != nil {
		if subcommand := cmd.Annotations["subcommand"]; subcommand != "" {
			// This is a config subcommand test, prepend the subcommand to args
			allArgs := append([]string{subcommand}, args...)
			cmd.SetArgs(allArgs)
		} else {
			cmd.SetArgs(args)
		}
	} else {
		cmd.SetArgs(args)
	}

	// Set flags (try both local and persistent flags)
	for flagName, flagValue := range flags {
		var valueStr string
		switch v := flagValue.(type) {
		case bool:
			valueStr = fmt.Sprintf("%t", v)
		case string:
			valueStr = v
		case int:
			valueStr = fmt.Sprintf("%d", v)
		case []string:
			// Handle string slices by setting each value
			for _, val := range v {
				if localFlag := cmd.Flags().Lookup(flagName); localFlag != nil {
					cmd.Flags().Set(flagName, val)
				} else if persistentFlag := cmd.PersistentFlags().Lookup(flagName); persistentFlag != nil {
					cmd.PersistentFlags().Set(flagName, val)
				}
			}
			continue
		default:
			valueStr = fmt.Sprintf("%v", v)
		}

		// Try to set on local flags first, then persistent flags
		if localFlag := cmd.Flags().Lookup(flagName); localFlag != nil {
			cmd.Flags().Set(flagName, valueStr)
		} else if persistentFlag := cmd.PersistentFlags().Lookup(flagName); persistentFlag != nil {
			cmd.PersistentFlags().Set(flagName, valueStr)
		}
	}

	// Execute command
	err := cmd.ExecuteContext(context.Background())

	return stdout, stderr, err
}

// createConfigTestCommand creates a config command for testing specific subcommands
func createConfigTestCommand(subcommand string) *cobra.Command {
	configCmd := NewConfigCommand()
	
	// Store the subcommand for the test to use
	configCmd.Annotations = map[string]string{"subcommand": subcommand}
	
	return configCmd
}
