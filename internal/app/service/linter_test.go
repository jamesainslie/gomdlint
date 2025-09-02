package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// Test scenario structure following club/ standards
type linterTestScenario struct {
	name             string
	content          string
	config           map[string]interface{}
	expectError      bool
	expectViolations bool
	minViolations    int
	expectedRules    []string
}

// Test helper functions
func createTempMarkdownFile(t *testing.T, content string) string {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "linter-test-*.md")
	require.NoError(t, err)

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)

	err = tmpFile.Close()
	require.NoError(t, err)

	t.Cleanup(func() {
		os.Remove(tmpFile.Name())
	})

	return tmpFile.Name()
}

func createTempMarkdownFiles(t *testing.T, contents map[string]string) []string {
	t.Helper()

	var files []string
	for filename, content := range contents {
		tmpFile := filepath.Join(t.TempDir(), filename)
		err := os.WriteFile(tmpFile, []byte(content), 0644)
		require.NoError(t, err)
		files = append(files, tmpFile)
	}

	return files
}

func assertLinterResult(t *testing.T, result functional.Result[*value.LintResult], scenario linterTestScenario) {
	t.Helper()

	if scenario.expectError {
		require.True(t, result.IsErr(), "Expected error but got success")
		return
	}

	if result.IsErr() {
		require.Fail(t, "Expected success but got error", "%v", result.Error())
	}
	require.True(t, result.IsOk())

	lintResult := result.Unwrap()
	require.NotNil(t, lintResult)

	if scenario.expectViolations {
		assert.Greater(t, lintResult.TotalViolations, 0, "Expected violations but got none")

		if scenario.minViolations > 0 {
			assert.GreaterOrEqual(t, lintResult.TotalViolations, scenario.minViolations)
		}

		// Check for specific rule violations
		if len(scenario.expectedRules) > 0 {
			foundRules := make(map[string]bool)
			for _, violations := range lintResult.Results {
				for _, violation := range violations {
					for _, ruleName := range violation.RuleNames {
						foundRules[ruleName] = true
					}
				}
			}

			for _, expectedRule := range scenario.expectedRules {
				assert.True(t, foundRules[expectedRule], "Expected rule %s not found in violations", expectedRule)
			}
		}
	} else {
		assert.Equal(t, 0, lintResult.TotalViolations, "Expected no violations but got %d", lintResult.TotalViolations)
	}
}

func TestNewLinterService_Scenarios(t *testing.T) {
	scenarios := []struct {
		name        string
		options     *value.LintOptions
		expectError bool
		expectNil   bool
	}{
		{
			name:        "default options",
			options:     value.NewLintOptions(),
			expectError: false,
			expectNil:   false,
		},
		{
			name:        "nil options",
			options:     nil,
			expectError: true,
			expectNil:   true,
		},
		{
			name: "options with configuration",
			options: value.NewLintOptions().WithConfig(map[string]interface{}{
				"MD013": map[string]interface{}{"line_length": 120},
				"MD041": false,
			}),
			expectError: false,
			expectNil:   false,
		},
		{
			name: "options with files and strings",
			options: value.NewLintOptions().
				WithFiles([]string{"test1.md", "test2.md"}).
				WithStrings(map[string]string{"string1": "# Title\n"}),
			expectError: false,
			expectNil:   false,
		},
		{
			name: "options with invalid config",
			options: value.NewLintOptions().WithConfig(map[string]interface{}{
				"MD013": "invalid_config_value", // Should be map, not string
			}),
			expectError: true, // Service should reject invalid config
			expectNil:   true,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			service, err := NewLinterService(scenario.options)

			if scenario.expectError {
				require.Error(t, err)
				if scenario.expectNil {
					assert.Nil(t, service)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, service)

			// Verify service components are initialized
			assert.NotNil(t, service.parser)
			assert.NotNil(t, service.ruleEngine)
			assert.NotNil(t, service.options)
			assert.NotNil(t, service.resultCache)
			assert.Greater(t, service.concurrency, 0)
		})
	}
}

func TestLinterService_LintStrings_Scenarios(t *testing.T) {
	scenarios := []linterTestScenario{
		{
			name:             "valid markdown",
			content:          "# Title\n\n## Subtitle\n\nParagraph content.\n",
			expectError:      false,
			expectViolations: false,
		},
		{
			name:             "heading without space",
			content:          "#Title\n\nContent.\n",
			expectError:      false,
			expectViolations: true,
			minViolations:    1,
			expectedRules:    []string{"MD018"},
		},
		{
			name:             "multiple violations",
			content:          "#Title\n\n\n\nToo many blank lines.\ttab\n",
			expectError:      false,
			expectViolations: true,
			minViolations:    2,
			expectedRules:    []string{"MD018", "MD012"},
		},
		{
			name:             "empty content",
			content:          "",
			expectError:      false,
			expectViolations: false, // Empty files are not violations
		},
		{
			name:             "only whitespace",
			content:          "   \n\t\n   ",
			expectError:      false,
			expectViolations: true, // Linter may flag whitespace-only files
		},
		{
			name:        "very large content",
			content:     "# Title\n\n" + strings.Repeat("This is line content.\n", 10000),
			expectError: false,
		},
	}

	ctx := context.Background()

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			service := createTestLinterService(t)

			content := map[string]string{
				"test": scenario.content,
			}

			result := service.LintStrings(ctx, content)
			assertLinterResult(t, result, scenario)
		})
	}
}

func TestLinterService_LintFiles_Scenarios(t *testing.T) {
	scenarios := []struct {
		name          string
		files         map[string]string // filename -> content
		expectError   bool
		expectSuccess bool
		minFiles      int
		minViolations int
	}{
		{
			name: "single valid file",
			files: map[string]string{
				"test.md": "# Title\n\nValid content.\n",
			},
			expectError:   false,
			expectSuccess: true,
			minFiles:      1,
			minViolations: 0,
		},
		{
			name: "multiple files mixed results",
			files: map[string]string{
				"good.md":    "# Good File\n\nProper content.\n",
				"bad.md":     "#Bad heading\n\nContent.\n",
				"minimal.md": "# Minimal\n",
			},
			expectError:   false,
			expectSuccess: true,
			minFiles:      3,
			minViolations: 1, // bad.md should have violations
		},
		{
			name: "empty files",
			files: map[string]string{
				"empty1.md": "",
				"empty2.md": "",
			},
			expectError:   false,
			expectSuccess: true,
			minFiles:      2,
			minViolations: 0,
		},
		{
			name:          "no files",
			files:         map[string]string{},
			expectError:   false,
			expectSuccess: true,
			minFiles:      0,
			minViolations: 0,
		},
	}

	ctx := context.Background()

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			service := createTestLinterService(t)

			// Create test files
			files := createTempMarkdownFiles(t, scenario.files)

			result := service.LintFiles(ctx, files)

			if scenario.expectError {
				require.True(t, result.IsErr(), "Expected error but got success")
				return
			}

			if result.IsErr() {
				require.True(t, result.IsOk(), "Expected success but got error: %v", result.Error())
			} else {
				require.True(t, result.IsOk(), "Expected success but result was not OK")
			}

			lintResult := result.Unwrap()
			require.NotNil(t, lintResult)

			assert.GreaterOrEqual(t, lintResult.TotalFiles, scenario.minFiles)
			assert.GreaterOrEqual(t, lintResult.TotalViolations, scenario.minViolations)
		})
	}
}

func TestLinterService_Lint_CombinedInput(t *testing.T) {
	ctx := context.Background()

	// Create test files
	testFiles := createTempMarkdownFiles(t, map[string]string{
		"file1.md": "# File One\n\nContent from file.\n",
		"file2.md": "#Bad file\n\nContent.\n", // Should have violation
	})

	// Create service with combined input
	options := value.NewLintOptions().
		WithFiles(testFiles).
		WithStrings(map[string]string{
			"string1": "# String One\n\nContent from string.\n",
			"string2": "#Bad string\n\nContent.\n", // Should have violation
		})

	service := createTestLinterService(t, options)

	result := service.Lint(ctx)
	if result.IsErr() {
		require.True(t, result.IsOk(), "Expected success but got error: %v", result.Error())
	} else {
		require.True(t, result.IsOk(), "Expected success but result was not OK")
	}

	lintResult := result.Unwrap()
	require.NotNil(t, lintResult)

	// Should process both files and strings
	assert.Equal(t, 4, lintResult.TotalFiles)        // 2 files + 2 strings
	assert.Greater(t, lintResult.TotalViolations, 0) // Should have violations from bad file and string

	// Verify all inputs are represented
	assert.Len(t, lintResult.Results, 4)
}

func TestLinterService_ContextCancellation(t *testing.T) {
	// Test that linting respects context cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	// Create large content that would take time to process
	largeContent := map[string]string{
		"large": "# Title\n\n" + strings.Repeat("Content line.\n", 100000),
	}

	service := createTestLinterService(t)

	result := service.LintStrings(ctx, largeContent)

	// Should either complete quickly or be cancelled
	if result.IsErr() {
		// If cancelled, error should be context-related
		err := result.Error()
		assert.Contains(t, strings.ToLower(err.Error()), "context")
	} else {
		// If completed, result should be valid
		lintResult := result.Unwrap()
		assert.NotNil(t, lintResult)
	}
}

func TestLinterService_Concurrency(t *testing.T) {
	ctx := context.Background()

	// Create multiple files to test concurrent processing
	files := make(map[string]string)
	for i := 0; i < 10; i++ {
		files[fmt.Sprintf("file%d.md", i)] = fmt.Sprintf("# File %d\n\nContent for file %d.\n", i, i)
	}

	testFiles := createTempMarkdownFiles(t, files)

	service := createTestLinterService(t)

	// Measure processing time
	start := time.Now()
	result := service.LintFiles(ctx, testFiles)
	duration := time.Since(start)

	if result.IsErr() {
		t.Fatalf("Expected success but got error: %v", result.Error())
	}
	require.True(t, result.IsOk(), "Expected success")

	lintResult := result.Unwrap()
	require.NotNil(t, lintResult)

	assert.Equal(t, 10, lintResult.TotalFiles)

	// Concurrent processing should be reasonably fast
	assert.Less(t, duration, 5*time.Second, "Processing took too long, concurrency may not be working")
}

func TestLinterService_CacheOperations(t *testing.T) {
	ctx := context.Background()
	service := createTestLinterService(t)

	t.Run("ClearCache", func(t *testing.T) {
		// First, populate cache by linting some content
		content := map[string]string{
			"test": "# Title\n\nContent.\n",
		}

		result := service.LintStrings(ctx, content)
		require.True(t, result.IsOk())

		// Cache should have entries
		stats := service.Stats()
		cacheSize, exists := stats["cache_size"]
		require.True(t, exists)
		assert.Greater(t, cacheSize, 0)

		// Clear cache
		service.ClearCache()

		// Cache should be empty
		stats = service.Stats()
		cacheSize, exists = stats["cache_size"]
		require.True(t, exists)
		assert.Equal(t, 0, cacheSize)
	})

	t.Run("Stats", func(t *testing.T) {
		stats := service.Stats()

		// Should contain expected keys
		assert.Contains(t, stats, "cache_size")
		assert.Contains(t, stats, "concurrency")

		// Values should be reasonable
		assert.IsType(t, 0, stats["cache_size"])
		assert.IsType(t, 0, stats["concurrency"])
		assert.Greater(t, stats["concurrency"], 0)
	})
}

func TestLinterService_UpdateOptions(t *testing.T) {
	service := createTestLinterService(t)

	// Initial options
	initialOptions := service.GetOptions()
	require.NotNil(t, initialOptions)

	// Update with new options
	newOptions := value.NewLintOptions().WithConfig(map[string]interface{}{
		"MD013": map[string]interface{}{"line_length": 200},
	})

	err := service.UpdateOptions(newOptions)
	require.NoError(t, err)

	// Options should be updated
	updatedOptions := service.GetOptions()
	require.NotNil(t, updatedOptions)
	assert.NotEqual(t, initialOptions, updatedOptions)

	// Cache should be cleared after update
	stats := service.Stats()
	cacheSize, exists := stats["cache_size"]
	require.True(t, exists)
	assert.Equal(t, 0, cacheSize)
}

func TestLinterService_GetterMethods(t *testing.T) {
	service := createTestLinterService(t)

	t.Run("GetParserService", func(t *testing.T) {
		parser := service.GetParserService()
		assert.NotNil(t, parser)
		assert.IsType(t, &ParserService{}, parser)
	})

	t.Run("GetRuleEngine", func(t *testing.T) {
		engine := service.GetRuleEngine()
		assert.NotNil(t, engine)
		assert.IsType(t, &RuleEngine{}, engine)
	})

	t.Run("GetOptions", func(t *testing.T) {
		options := service.GetOptions()
		assert.NotNil(t, options)
		assert.IsType(t, &value.LintOptions{}, options)
	})
}

// Benchmark tests for performance
func BenchmarkLinterService_LintStrings(b *testing.B) {
	ctx := context.Background()
	service := createTestLinterService(b)

	content := map[string]string{
		"benchmark": generateBenchmarkContent(1000),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := service.LintStrings(ctx, content)
		require.True(b, result.IsOk())
	}
}

func BenchmarkLinterService_LintFiles(b *testing.B) {
	ctx := context.Background()
	service := createTestLinterService(b)

	// Create test files
	files := make([]string, 5)
	for i := 0; i < 5; i++ {
		tmpFile, err := os.CreateTemp("", "benchmark-*.md")
		require.NoError(b, err)

		content := generateBenchmarkContent(200)
		_, err = tmpFile.WriteString(content)
		require.NoError(b, err)
		tmpFile.Close()

		files[i] = tmpFile.Name()
		b.Cleanup(func() { os.Remove(tmpFile.Name()) })
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := service.LintFiles(ctx, files)
		require.True(b, result.IsOk())
	}
}

// Helper function to generate benchmark content
func generateBenchmarkContent(lines int) string {
	var content strings.Builder
	content.WriteString("# Benchmark Document\n\n")

	for i := 0; i < lines; i++ {
		switch i % 4 {
		case 0:
			content.WriteString(fmt.Sprintf("## Section %d\n\n", i/4+1))
		case 1:
			content.WriteString("This is paragraph content for testing.\n\n")
		case 2:
			content.WriteString("- List item for variety\n")
		case 3:
			content.WriteString("`inline code` and **bold text**.\n\n")
		}
	}

	return content.String()
}

// Helper function to convert *testing.B to have similar interface as *testing.T for createTestLinterService
func createTestLinterService(tb testing.TB, options ...*value.LintOptions) *LinterService {
	tb.Helper()

	var opts *value.LintOptions
	if len(options) > 0 && options[0] != nil {
		opts = options[0]
	} else {
		opts = value.NewLintOptions()
	}

	service, err := NewLinterService(opts)
	require.NoError(tb, err)
	require.NotNil(tb, service)

	return service
}
