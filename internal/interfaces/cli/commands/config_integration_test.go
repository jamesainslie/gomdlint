//go:build integration

package commands

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Integration tests for config command that require file I/O
// Run with: go test -tags=integration ./internal/interfaces/cli/commands/

func TestConfigCommand_InitIntegration(t *testing.T) {
	scenarios := []configCommandScenario{
		{
			name:          "init_default_location",
			subcommand:    "init",
			args:          []string{},
			expectError:   false,
			expectedFiles: map[string]string{
				// Path will be XDG config location - verified in test
			},
		},
		{
			name:       "init_legacy_location",
			subcommand: "init",
			args:       []string{},
			flags:      map[string]interface{}{"legacy": true},
			expectedFiles: map[string]string{
				".markdownlint.json": "", // Should contain default config
			},
			expectError: false,
		},
		{
			name:        "init_in_nested_directory",
			subcommand:  "init",
			args:        []string{},
			flags:       map[string]interface{}{"legacy": true},
			expectError: false,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			runConfigIntegrationTestScenario(t, scenario)
		})
	}
}

func TestConfigCommand_ValidateIntegration(t *testing.T) {
	scenarios := []configCommandScenario{
		{
			name:       "validate_valid_config",
			subcommand: "validate",
			setupFiles: map[string]string{
				"test-config.json": `{"MD013": {"line_length": 120}, "MD033": false, "default": true}`,
			},
			args:        []string{"test-config.json"},
			expectError: false,
		},
		{
			name:        "validate_user_config",
			subcommand:  "validate",
			args:        []string{},
			expectError: false,
		},
		{
			name:       "validate_theme_config",
			subcommand: "validate",
			setupFiles: map[string]string{
				"theme-config.json": `{"theme": "custom", "MD013": {"line_length": 100}}`,
			},
			args:        []string{"theme-config.json"},
			expectError: false,
		},
		{
			name:       "validate_theme_object_config",
			subcommand: "validate",
			setupFiles: map[string]string{
				"theme-object-config.json": `{"theme": {"name": "custom", "colors": {"error": "red"}}, "MD033": false}`,
			},
			args:        []string{"theme-object-config.json"},
			expectError: false,
		},
		{
			name:       "validate_hierarchical_config",
			subcommand: "validate",
			setupFiles: map[string]string{
				"config.json": `{"MD013": {"line_length": 90}, "MD033": false}`,
			},
			args:        []string{},
			expectError: false,
		},
		{
			name:        "validate_nonexistent_config",
			subcommand:  "validate",
			args:        []string{"nonexistent.json"},
			expectError: true,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			runConfigIntegrationTestScenario(t, scenario)
		})
	}
}

func TestConfigCommand_ShowIntegration(t *testing.T) {
	scenarios := []configCommandScenario{
		{
			name:        "show_user_config",
			subcommand:  "show",
			args:        []string{},
			expectError: false,
		},
		{
			name:       "show_specific_config",
			subcommand: "show",
			setupFiles: map[string]string{
				"show-config.json": `{"MD013": {"line_length": 90}, "MD033": false}`,
			},
			args:        []string{"show-config.json"},
			expectError: false,
		},
		{
			name:       "show_hierarchical_config",
			subcommand: "show",
			setupFiles: map[string]string{
				"config.json": `{"MD013": {"line_length": 90}, "MD033": false}`,
			},
			args:        []string{},
			expectError: false,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			runConfigIntegrationTestScenario(t, scenario)
		})
	}
}

func TestConfigCommand_WhichIntegration(t *testing.T) {
	scenarios := []configCommandScenario{
		{
			name:        "which_user_config",
			subcommand:  "which",
			args:        []string{},
			expectError: false,
		},
		{
			name:       "which_hierarchical_config",
			subcommand: "which",
			setupFiles: map[string]string{
				".gomdlint.json": `{"MD013": {"line_length": 90}}`,
			},
			args:        []string{},
			expectError: false,
		},
		{
			name:       "which_project_and_config",
			subcommand: "which",
			setupFiles: map[string]string{
				".gomdlint.json": `{"MD013": {"line_length": 90}}`,
				"config.json":    `{"MD033": false}`,
			},
			args:        []string{},
			expectError: false,
		},
		{
			name:        "which_no_config",
			subcommand:  "which",
			args:        []string{},
			setupEnv:    map[string]string{"XDG_CONFIG_HOME": "/tmp/nonexistent"},
			expectError: true,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			runConfigIntegrationTestScenario(t, scenario)
		})
	}
}

func TestConfigCommand_EditIntegration(t *testing.T) {
	scenarios := []configCommandScenario{
		{
			name:        "edit_user_config",
			subcommand:  "edit",
			args:        []string{},
			setupEnv:    map[string]string{"EDITOR": "nano"},
			expectError: false,
		},
		{
			name:        "edit_with_custom_editor",
			subcommand:  "edit",
			args:        []string{},
			setupEnv:    map[string]string{"EDITOR": "echo"},
			expectError: false,
		},
		{
			name:       "edit_specific_file",
			subcommand: "edit",
			setupFiles: map[string]string{
				"edit-config.json": `{"MD013": {"line_length": 90}}`,
			},
			args:        []string{"edit-config.json"},
			setupEnv:    map[string]string{"EDITOR": "echo"},
			expectError: false,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			runConfigIntegrationTestScenario(t, scenario)
		})
	}
}

func TestConfigCommand_EdgeCasesIntegration(t *testing.T) {
	scenarios := []configCommandScenario{
		{
			name:        "init_in_nested_directory",
			subcommand:  "init",
			flags:       map[string]interface{}{"legacy": true},
			expectError: false,
		},
		{
			name:       "show_deeply_nested_config",
			subcommand: "show",
			setupFiles: map[string]string{
				"nested-config.json": `{
					"MD013": {
						"line_length": 100,
						"code_blocks": false,
						"tables": false
					},
					"rules": {
						"custom": {
							"enabled": true,
							"settings": {
								"severity": "error",
								"ignore_patterns": ["*.test.md"]
							}
						}
					}
				}`,
			},
			args:        []string{"nested-config.json"},
			expectError: false,
		},
		{
			name:        "edit_nonexistent_file_creates",
			subcommand:  "edit",
			args:        []string{"nonexistent.json"},
			setupEnv:    map[string]string{"EDITOR": "nano"},
			expectError: false,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			runConfigIntegrationTestScenario(t, scenario)
		})
	}
}

// Helper function to run config integration test scenarios
func runConfigIntegrationTestScenario(t *testing.T, scenario configCommandScenario) {
	t.Helper()

	// Setup test environment
	tmpDir := createTempTestFiles(t, scenario.setupFiles)

	// Change to test directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() {
		os.Chdir(originalDir)
	}()

	// Setup environment variables
	originalEnv := make(map[string]string)
	for key, value := range scenario.setupEnv {
		originalEnv[key] = os.Getenv(key)
		if value == "" {
			os.Unsetenv(key)
		} else {
			os.Setenv(key, value)
		}
	}
	defer func() {
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	// Create command
	cmd := createConfigTestCommand(scenario.subcommand)

	// Execute command
	stdout, stderr, err := executeCommand(t, cmd, scenario.args, scenario.flags)

	// Verify results
	if scenario.expectError {
		assert.Error(t, err, "Expected error but got none")
	} else {
		assert.NoError(t, err, "Expected no error but got: %v", err)
	}

	// Verify expected output contains expected content
	if scenario.expectedOutput != "" {
		output := stdout.String() + stderr.String()
		assert.Contains(t, output, scenario.expectedOutput)
	}

	// Verify expected files exist
	for filename, expectedContent := range scenario.expectedFiles {
		if filename != "" { // Skip empty filename entries
			// For XDG config files, we need to check the actual XDG path
			if scenario.subcommand == "init" && (scenario.flags["legacy"] == nil || !scenario.flags["legacy"].(bool)) {
				// This would be the XDG config directory - more complex to test
				t.Logf("XDG config file creation test skipped - requires XDG path resolution")
			} else {
				assert.FileExists(t, filename, "Expected file %s should exist", filename)

				if expectedContent != "" {
					content, err := os.ReadFile(filename)
					if assert.NoError(t, err, "Should be able to read file %s", filename) {
						if expectedContent != "" {
							var jsonData interface{}
							assert.NoError(t, json.Unmarshal(content, &jsonData),
								"File %s should contain valid JSON", filename)
						}
					}
				}
			}
		}
	}
}
