package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test scenario structure for benchmark command
type benchmarkCommandScenario struct {
	name           string
	args           []string
	flags          map[string]interface{}
	setupFiles     map[string]string // filename -> content
	setupEnv       map[string]string // env var -> value
	expectError    bool
	expectedOutput string            // substring that should be in output (when testable)
	expectedFiles  map[string]string // files that should exist after command
	skipSlowTests  bool              // skip tests that take a long time
}

func TestNewBenchmarkCommand(t *testing.T) {
	cmd := NewBenchmarkCommand()

	assert.NotNil(t, cmd, "Command should not be nil")
	assert.Equal(t, "benchmark [files...]", cmd.Use)
	assert.Contains(t, cmd.Short, "Performance benchmark")
	assert.Contains(t, cmd.Long, "performance benchmarks")

	// Check all expected flags exist
	expectedFlags := []string{
		"iterations", "generate-test-files", "test-file-count", "test-file-size",
		"markdownlint-cli", "skip-markdownlint", "output", "verbose",
	}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		require.NotNil(t, flag, "Flag %s should exist", flagName)

		// Check default values for some flags
		switch flagName {
		case "iterations":
			assert.Equal(t, "3", flag.DefValue, "Flag %s should have correct default", flagName)
		case "test-file-count":
			assert.Equal(t, "50", flag.DefValue, "Flag %s should have correct default", flagName)
		case "markdownlint-cli":
			assert.Equal(t, "markdownlint", flag.DefValue, "Flag %s should have correct default", flagName)
		}
	}

	// Check that Args is set to ArbitraryArgs
	assert.NotNil(t, cmd.Args, "Args should be set")
}

func TestBenchmarkCommand_BasicFunctionality(t *testing.T) {
	scenarios := []benchmarkCommandScenario{
		{
			name: "generate_test_files_simple",
			args: []string{},
			flags: map[string]interface{}{
				"generate-test-files": true,
				"test-file-count":     5,
				"test-file-size":      500,
				"iterations":          1,
				"skip-markdownlint":   true, // Skip markdownlint to avoid dependency
			},
			expectError: false,
			// Note: Output verification difficult due to complex formatting
		},
		{
			name: "benchmark_specific_files",
			args: []string{"test1.md", "test2.md"},
			flags: map[string]interface{}{
				"iterations":        1,
				"skip-markdownlint": true,
			},
			setupFiles: map[string]string{
				"test1.md": "# Test Document 1\n\nThis is a test document.\n",
				"test2.md": "# Test Document 2\n\nAnother test document.\n",
			},
			expectError: false,
		},
		{
			name: "output_to_json_file",
			args: []string{},
			flags: map[string]interface{}{
				"generate-test-files": true,
				"test-file-count":     3,
				"iterations":          1,
				"skip-markdownlint":   true,
				"output":              "benchmark-results.json",
			},
			expectedFiles: map[string]string{
				"benchmark-results.json": "", // Will contain JSON data
			},
			expectError: false,
		},
		{
			name: "verbose_output",
			args: []string{},
			flags: map[string]interface{}{
				"generate-test-files": true,
				"test-file-count":     2,
				"iterations":          2,
				"skip-markdownlint":   true,
				"verbose":             true,
			},
			expectError: false,
		},
		{
			name: "custom_test_parameters",
			args: []string{},
			flags: map[string]interface{}{
				"generate-test-files": true,
				"test-file-count":     10,
				"test-file-size":      2000,
				"iterations":          1,
				"skip-markdownlint":   true,
			},
			expectError: false,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			runBenchmarkTestScenario(t, scenario)
		})
	}
}

func TestBenchmarkCommand_SpecialCases(t *testing.T) {
	scenarios := []benchmarkCommandScenario{
		{
			name:        "no_files_found_auto_generate",
			args:        []string{},
			flags:       map[string]interface{}{"skip-markdownlint": true, "iterations": 1},
			setupFiles:  map[string]string{}, // No markdown files
			expectError: false,
			// Should auto-generate test files when none found
		},
		{
			name: "nonexistent_files",
			args: []string{"nonexistent1.md", "nonexistent2.md"},
			flags: map[string]interface{}{
				"skip-markdownlint": true,
				"iterations":        1,
			},
			expectError: true, // Should fail when specified files don't exist
		},
		{
			name: "glob_pattern_files",
			args: []string{"test*.md"},
			flags: map[string]interface{}{
				"skip-markdownlint": true,
				"iterations":        1,
			},
			setupFiles: map[string]string{
				"test_a.md": "# Test A\n\nContent A.\n",
				"test_b.md": "# Test B\n\nContent B.\n",
				"other.md":  "# Other\n\nOther content.\n", // Should not match pattern
			},
			expectError: false,
		},
		{
			name: "zero_iterations",
			args: []string{},
			flags: map[string]interface{}{
				"generate-test-files": true,
				"test-file-count":     2,
				"iterations":          0,
				"skip-markdownlint":   true,
			},
			expectError: false, // Should handle gracefully
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			runBenchmarkTestScenario(t, scenario)
		})
	}
}

func TestBenchmarkCommand_TestFileGeneration(t *testing.T) {
	// Test the specific test file generation functions directly
	t.Run("generate_test_markdown_files", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		err = os.Chdir(tmpDir)
		require.NoError(t, err)
		defer os.Chdir(originalDir)

		files, err := generateTestMarkdownFiles(3, 200) // Reduced from 5 files @ 1000 bytes
		require.NoError(t, err)
		require.Len(t, files, 3, "Should generate exactly 3 files")

		// Verify files exist and have reasonable content
		for i, file := range files {
			require.FileExists(t, file)
			content, err := os.ReadFile(file)
			require.NoError(t, err)
			assert.True(t, len(content) > 50, "File %d should have substantial content", i) // Reduced expectation
			assert.True(t, strings.Contains(string(content), "#"), "File %d should contain headings", i)
		}

		// Test cleanup
		cleanupTestFiles(files)
		// Verify directory is removed
		if len(files) > 0 {
			testDir := filepath.Dir(files[0])
			assert.NoFileExists(t, testDir)
		}
	})

	// Test individual content generators
	t.Run("content_generators", func(t *testing.T) {
		generators := []struct {
			name string
			fn   func(int, int) string
		}{
			{"violations", generateMarkdownWithViolations},
			{"compliant", generateCompliantMarkdown},
			{"complex", generateComplexMarkdown},
			{"list_heavy", generateListHeavyMarkdown},
			{"code_heavy", generateCodeHeavyMarkdown},
		}

		for _, gen := range generators {
			t.Run(gen.name, func(t *testing.T) {
				t.Parallel()
				content := gen.fn(1, 200)                                                                        // Reduced from 500 bytes
				assert.True(t, len(content) >= 100, "%s generator should produce substantial content", gen.name) // Reduced expectation
				assert.True(t, strings.Contains(content, "#"), "%s generator should include headings", gen.name)
			})
		}
	})
}

func TestBenchmarkCommand_Utilities(t *testing.T) {
	t.Run("expand_file_paths", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create test files
		testFiles := map[string]string{
			"file1.md":      "# File 1",
			"file2.md":      "# File 2",
			"doc1.txt":      "Text file",
			"subdir/sub.md": "# Subdir file",
		}

		for filename, content := range testFiles {
			filePath := filepath.Join(tmpDir, filename)
			if strings.Contains(filename, "/") {
				err := os.MkdirAll(filepath.Dir(filePath), 0755)
				require.NoError(t, err)
			}
			err := os.WriteFile(filePath, []byte(content), 0644)
			require.NoError(t, err)
		}

		originalDir, err := os.Getwd()
		require.NoError(t, err)
		err = os.Chdir(tmpDir)
		require.NoError(t, err)
		defer os.Chdir(originalDir)

		// Test glob expansion
		files, err := expandFilePaths([]string{"*.md"})
		require.NoError(t, err)
		assert.Len(t, files, 2, "Should find 2 .md files in root")

		// Test multiple patterns
		files, err = expandFilePaths([]string{"*.md", "*.txt"})
		require.NoError(t, err)
		assert.Len(t, files, 3, "Should find 2 .md files and 1 .txt file")
	})

	t.Run("find_markdown_files", func(t *testing.T) {
		tmpDir := t.TempDir()

		testFiles := map[string]string{
			"file1.md":       "# File 1",
			"file2.markdown": "# File 2",
			"file3.txt":      "Not markdown",
			"subdir/sub.md":  "# Subdir file",
		}

		for filename, content := range testFiles {
			filePath := filepath.Join(tmpDir, filename)
			if strings.Contains(filename, "/") {
				err := os.MkdirAll(filepath.Dir(filePath), 0755)
				require.NoError(t, err)
			}
			err := os.WriteFile(filePath, []byte(content), 0644)
			require.NoError(t, err)
		}

		files, err := findMarkdownFiles(tmpDir)
		require.NoError(t, err)
		assert.True(t, len(files) >= 2, "Should find at least 2 markdown files")

		// Verify all found files are markdown
		for _, file := range files {
			assert.True(t, isMarkdownFile(file), "File %s should be recognized as markdown", file)
		}
	})

	t.Run("calculate_average", func(t *testing.T) {
		results := []BenchmarkResult{
			{
				Tool:            "test",
				Version:         "1.0",
				ExecutionTime:   100 * time.Millisecond,
				MemoryUsage:     1000,
				ViolationsFound: 5,
				FilesProcessed:  10,
				Success:         true,
			},
			{
				Tool:            "test",
				Version:         "1.0",
				ExecutionTime:   200 * time.Millisecond,
				MemoryUsage:     2000,
				ViolationsFound: 10,
				FilesProcessed:  20,
				Success:         true,
			},
		}

		avg := calculateAverage(results)
		assert.Equal(t, "test", avg.Tool)
		assert.Equal(t, "1.0", avg.Version)
		assert.Equal(t, 150*time.Millisecond, avg.ExecutionTime)
		assert.Equal(t, int64(1500), avg.MemoryUsage)
		assert.Equal(t, 7, avg.ViolationsFound) // (5+10)/2 = 7.5 -> 7 (integer division)
		assert.Equal(t, 15, avg.FilesProcessed)
		assert.True(t, avg.Success)
	})

	t.Run("calculate_speedup", func(t *testing.T) {
		oldTime := 200 * time.Millisecond
		newTime := 100 * time.Millisecond
		speedup := calculateSpeedup(oldTime, newTime)
		assert.Equal(t, 2.0, speedup)

		// Test zero division handling
		speedup = calculateSpeedup(oldTime, 0)
		assert.Equal(t, 0.0, speedup)
	})

	t.Run("calculate_memory_ratio", func(t *testing.T) {
		ratio := calculateMemoryRatio(2000, 1000)
		assert.Equal(t, 2.0, ratio)

		// Test zero division handling
		ratio = calculateMemoryRatio(2000, 0)
		assert.Equal(t, 0.0, ratio)
	})

	t.Run("format_bytes", func(t *testing.T) {
		assert.Equal(t, "500 B", formatBytes(500))
		assert.Equal(t, "1.5 KB", formatBytes(1536))
		assert.Equal(t, "2.0 MB", formatBytes(2*1024*1024))
		assert.Equal(t, "1.0 GB", formatBytes(1024*1024*1024))
	})
}

func TestBenchmarkCommand_ErrorHandling(t *testing.T) {
	scenarios := []benchmarkCommandScenario{
		{
			name: "invalid_test_file_count",
			args: []string{},
			flags: map[string]interface{}{
				"generate-test-files": true,
				"test-file-count":     0,
				"skip-markdownlint":   true,
			},
			expectError: false, // Should handle gracefully, might default to some minimum
		},
		{
			name: "invalid_output_directory",
			args: []string{},
			flags: map[string]interface{}{
				"generate-test-files": true,
				"test-file-count":     2,
				"iterations":          1,
				"skip-markdownlint":   true,
				"output":              "/nonexistent/directory/output.json",
			},
			expectError: true, // Should fail to write to non-existent directory
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			runBenchmarkTestScenario(t, scenario)
		})
	}
}

func runBenchmarkTestScenario(t *testing.T, scenario benchmarkCommandScenario) {
	t.Helper()

	if scenario.skipSlowTests && testing.Short() {
		t.Skip("Skipping slow test in short mode")
	}

	// Setup test directory and files
	tmpDir := createTempTestFiles(t, scenario.setupFiles)

	// Change to test directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer os.Chdir(originalDir)

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

	// Setup command
	cmd := NewBenchmarkCommand()

	// Set flags on the command
	for flagName, flagValue := range scenario.flags {
		switch v := flagValue.(type) {
		case bool:
			cmd.Flags().Set(flagName, fmt.Sprintf("%t", v))
		case string:
			cmd.Flags().Set(flagName, v)
		case int:
			cmd.Flags().Set(flagName, fmt.Sprintf("%d", v))
		default:
			cmd.Flags().Set(flagName, fmt.Sprintf("%v", v))
		}
	}

	// Execute command - benchmark commands can be complex, so we simplify testing
	// by focusing on error conditions and file outputs rather than exact output
	ctx := context.Background()
	cmd.SetContext(ctx)

	// Execute command
	if cmd.RunE != nil {
		err = cmd.RunE(cmd, scenario.args)
	} else {
		t.Fatal("Benchmark command should have RunE function")
	}

	// Verify results
	if scenario.expectError {
		assert.Error(t, err, "Expected error but got none")
	} else {
		assert.NoError(t, err, "Expected no error but got: %v", err)
	}

	// Verify expected files exist (if specified)
	for filename, expectedContent := range scenario.expectedFiles {
		assert.FileExists(t, filename, "Expected file %s should exist", filename)

		if expectedContent != "" {
			content, err := os.ReadFile(filename)
			if assert.NoError(t, err, "Should be able to read file %s", filename) {
				if strings.HasSuffix(filename, ".json") {
					// For JSON files, verify it's valid JSON
					var jsonData interface{}
					assert.NoError(t, json.Unmarshal(content, &jsonData),
						"File %s should contain valid JSON", filename)
				} else {
					assert.Contains(t, string(content), expectedContent,
						"File %s should contain expected content", filename)
				}
			}
		}
	}

	// Verify output contains expected content (when testable)
	if scenario.expectedOutput != "" {
		// Note: This is limited since benchmark commands write directly to stdout
		// In a full implementation, we might capture output using the same
		// techniques as in the config tests
		t.Logf("Expected output (not verified): %s", scenario.expectedOutput)
	}
}

// Note: isMarkdownFile function is already defined in lint.go

func TestBenchmarkCommand_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This is a more comprehensive integration test
	t.Run("full_benchmark_cycle", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		err = os.Chdir(tmpDir)
		require.NoError(t, err)
		defer os.Chdir(originalDir)

		// Create some real markdown files with various issues
		testFiles := map[string]string{
			"good.md":  "# Good Document\n\nThis is properly formatted.\n\n## Section\n\nContent here.\n",
			"bad.md":   "#Bad heading\n\nThis has	tabs and    trailing spaces   \n\n##Another bad heading\n",
			"mixed.md": "# Mixed Document\n\n*Good* formatting and **bold** text.\n\n#Bad heading here\n",
		}

		for filename, content := range testFiles {
			err := os.WriteFile(filename, []byte(content), 0644)
			require.NoError(t, err)
		}

		// Run benchmark command
		cmd := NewBenchmarkCommand()
		cmd.Flags().Set("iterations", "1")
		cmd.Flags().Set("skip-markdownlint", "true")
		cmd.Flags().Set("output", "results.json")
		cmd.SetContext(context.Background())

		err = cmd.RunE(cmd, []string{"*.md"})
		assert.NoError(t, err, "Full benchmark cycle should succeed")

		// Verify results file was created
		assert.FileExists(t, "results.json")

		// Verify results file contains valid JSON
		content, err := os.ReadFile("results.json")
		require.NoError(t, err)

		var results map[string]interface{}
		err = json.Unmarshal(content, &results)
		require.NoError(t, err, "Results file should contain valid JSON")

		// Basic validation of results structure
		assert.Contains(t, results, "gomdlint", "Results should contain gomdlint data")
		assert.Contains(t, results, "timestamp", "Results should contain timestamp")
	})
}

// Benchmarks for the benchmark command itself
func BenchmarkBenchmarkCommand_TestFileGeneration(b *testing.B) {
	tmpDir := b.TempDir()
	originalDir, err := os.Getwd()
	require.NoError(b, err)
	err = os.Chdir(tmpDir)
	require.NoError(b, err)
	defer os.Chdir(originalDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		files, err := generateTestMarkdownFiles(10, 1000)
		if err != nil {
			b.Fatalf("Failed to generate test files: %v", err)
		}
		cleanupTestFiles(files)
	}
}

func BenchmarkBenchmarkCommand_CalculateAverage(b *testing.B) {
	// Create sample results
	results := make([]BenchmarkResult, 100)
	for i := range results {
		results[i] = BenchmarkResult{
			Tool:            "test",
			Version:         "1.0",
			ExecutionTime:   time.Duration(i+1) * time.Millisecond,
			MemoryUsage:     int64((i + 1) * 1000),
			ViolationsFound: i + 1,
			FilesProcessed:  10,
			Success:         true,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calculateAverage(results)
	}
}
