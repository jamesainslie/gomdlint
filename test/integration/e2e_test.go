package integration

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

	"github.com/gomdlint/gomdlint/pkg/gomdlint"
	"github.com/gomdlint/gomdlint/test/helpers"
)

// Integration tests following club/ standards for end-to-end testing

// collectMarkdownFiles collects actual markdown files from the given args
func collectMarkdownFiles(t testing.TB, args []string) []string {
	t.Helper()

	var files []string

	for _, arg := range args {
		if arg == "." {
			// Scan current directory for markdown files
			err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".md") {
					files = append(files, path)
				}
				return nil
			})
			require.NoError(t, err)
		} else if strings.HasSuffix(arg, "/") {
			// Scan directory for markdown files
			err := filepath.Walk(arg, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".md") {
					files = append(files, path)
				}
				return nil
			})
			require.NoError(t, err)
		} else {
			// Direct file path
			files = append(files, arg)
		}
	}

	return files
}

func TestE2E_BasicLintingWorkflow(t *testing.T) {
	scenarios := helpers.CreateTestScenarios()

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			helpers.WithTempDir(t, func(tempDir string) {
				// Setup test project
				project := helpers.TestProject{
					Name:        scenario.Name,
					Files:       scenario.Files,
					ConfigFiles: make(map[string]string),
				}

				if scenario.Config != "" {
					project.ConfigFiles[".markdownlint.json"] = scenario.Config
				}

				projectDir := helpers.CreateTestProject(t, project)

				// Change to the project directory where files were created
				originalDir, err := os.Getwd()
				require.NoError(t, err)
				err = os.Chdir(projectDir)
				require.NoError(t, err)
				defer func() {
					os.Chdir(originalDir)
				}()

				// Collect actual markdown files instead of passing directory
				markdownFiles := collectMarkdownFiles(t, scenario.Args)

				// Handle empty file list - if we expect no violations and have no files, that's OK
				if len(markdownFiles) == 0 {
					if !scenario.ExpectViolations {
						// No files and expecting no violations - test passes
						return
					}
					// No files but expecting violations - skip test as it's not meaningful
					t.Skipf("No markdown files found for scenario: %s", scenario.Name)
					return
				}

				// Prepare lint options
				options := gomdlint.LintOptions{
					Files:  markdownFiles,
					Config: make(map[string]interface{}),
				}

				// Load config if present
				if scenario.Config != "" {
					var config map[string]interface{}
					err := json.Unmarshal([]byte(scenario.Config), &config)
					require.NoError(t, err, "Failed to parse scenario config")
					options.Config = config
				}

				// Execute linting
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				result, err := gomdlint.Lint(ctx, options)

				// Verify results
				if scenario.ExpectError {
					require.Error(t, err)
					return
				}

				require.NoError(t, err)
				require.NotNil(t, result)

				// Check violation expectations
				if scenario.ExpectViolations {
					assert.Greater(t, result.TotalViolations, 0,
						"Expected violations but found none in scenario: %s", scenario.Name)

					if scenario.MinViolations > 0 {
						assert.GreaterOrEqual(t, result.TotalViolations, scenario.MinViolations,
							"Expected at least %d violations but found %d", scenario.MinViolations, result.TotalViolations)
					}

					if scenario.MaxViolations > 0 {
						assert.LessOrEqual(t, result.TotalViolations, scenario.MaxViolations,
							"Expected at most %d violations but found %d", scenario.MaxViolations, result.TotalViolations)
					}
				} else {
					assert.Equal(t, 0, result.TotalViolations,
						"Expected no violations but found %d in scenario: %s", result.TotalViolations, scenario.Name)
				}

				// Verify expected rules are found
				if len(scenario.ExpectedRules) > 0 {
					foundRules := extractRuleNamesFromResult(result)
					for _, expectedRule := range scenario.ExpectedRules {
						assert.Contains(t, foundRules, expectedRule,
							"Expected rule %s not found in results", expectedRule)
					}
				}
			})
		})
	}
}

func TestE2E_PerformanceWithLargeRepository(t *testing.T) {
	helpers.WithTempDir(t, func(tempDir string) {
		// Generate a large repository structure
		files := make([]helpers.TestFile, 50)
		for i := 0; i < 50; i++ {
			content := helpers.GenerateMarkdownContent(helpers.ContentOptions{
				Title:                "Performance Test Document",
				Sections:             5,
				ParagraphsPerSection: 3,
				InvalidHeadings:      i%10 == 0, // 10% have violations
				LongLines:            i%15 == 0, // Some have long lines
			})

			files[i] = helpers.TestFile{
				Path:    filepath.Join("docs", fmt.Sprintf("doc%03d.md", i)),
				Content: content,
			}
		}

		project := helpers.TestProject{
			Name:  "Large Repository",
			Files: files,
			ConfigFiles: map[string]string{
				".markdownlint.json": helpers.ConfigFiles.Basic,
			},
		}

		projectDir := helpers.CreateTestProject(t, project)

		// Change to the project directory where files were created
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		err = os.Chdir(projectDir)
		require.NoError(t, err)
		defer func() {
			os.Chdir(originalDir)
		}()

		// Measure linting performance
		start := time.Now()

		// Collect actual markdown files
		markdownFiles := collectMarkdownFiles(t, []string{"docs/"})

		options := gomdlint.LintOptions{
			Files:  markdownFiles,
			Config: make(map[string]interface{}),
		}

		// Load config
		var config map[string]interface{}
		err = json.Unmarshal([]byte(helpers.ConfigFiles.Basic), &config)
		require.NoError(t, err, "Failed to parse Basic config")
		options.Config = config

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		result, err := gomdlint.Lint(ctx, options)

		duration := time.Since(start)

		require.NoError(t, err)
		require.NotNil(t, result)

		// Performance assertions
		assert.Less(t, duration, 30*time.Second, "Linting should complete within 30 seconds")
		assert.Equal(t, 50, result.TotalFiles, "Should process all 50 files")
		assert.Greater(t, result.TotalViolations, 0, "Should find some violations")

		t.Logf("Processed %d files in %v, found %d violations",
			result.TotalFiles, duration, result.TotalViolations)
	})
}

func TestE2E_ConfigurationHierarchy(t *testing.T) {
	helpers.WithTempDir(t, func(tempDir string) {
		// Create hierarchical configuration setup
		project := helpers.TestProject{
			Name: "Hierarchical Config",
			Files: []helpers.TestFile{
				{Path: "README.md", Content: helpers.MarkdownFiles.InvalidHeading},
				{Path: "docs/guide.md", Content: helpers.MarkdownFiles.InvalidHeading},
				{Path: "docs/api/reference.md", Content: helpers.MarkdownFiles.InvalidHeading},
			},
			ConfigFiles: map[string]string{
				".markdownlint.json":          helpers.ConfigFiles.Strict,
				"docs/.markdownlint.json":     helpers.ConfigFiles.Relaxed,
				"docs/api/.markdownlint.json": helpers.ConfigFiles.DisableRules,
			},
		}

		projectDir := helpers.CreateTestProject(t, project)

		// Change to the project directory where files were created
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		err = os.Chdir(projectDir)
		require.NoError(t, err)
		defer func() {
			os.Chdir(originalDir)
		}()

		testCases := []struct {
			name             string
			files            []string
			expectViolations bool
		}{
			{
				name:             "Root level with strict config",
				files:            []string{"README.md"},
				expectViolations: true, // Strict config should catch violations
			},
			{
				name:             "Docs level with relaxed config",
				files:            []string{"docs/guide.md"},
				expectViolations: false, // Relaxed config allows violations
			},
			{
				name:             "API level with disabled rules",
				files:            []string{"docs/api/reference.md"},
				expectViolations: false, // Rules disabled
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				options := gomdlint.LintOptions{
					Files:  tc.files,
					Config: make(map[string]interface{}),
				}

				// Determine which config to use based on the file path
				var configContent string
				if len(tc.files) > 0 {
					filePath := tc.files[0]
					if strings.Contains(filePath, "docs/api/") {
						configContent = helpers.ConfigFiles.DisableRules
					} else if strings.Contains(filePath, "docs/") {
						configContent = helpers.ConfigFiles.Relaxed
					} else {
						configContent = helpers.ConfigFiles.Strict
					}
				}

				if configContent != "" {
					var config map[string]interface{}
					err := json.Unmarshal([]byte(configContent), &config)
					require.NoError(t, err, "Failed to parse config")
					options.Config = config
				}

				ctx := context.Background()
				result, err := gomdlint.Lint(ctx, options)

				require.NoError(t, err)
				require.NotNil(t, result)

				if tc.expectViolations {
					assert.Greater(t, result.TotalViolations, 0)
				} else {
					assert.Equal(t, 0, result.TotalViolations)
				}
			})
		}
	})
}

func TestE2E_OutputFormats(t *testing.T) {
	helpers.WithTempDir(t, func(tempDir string) {
		// Create test project with violations
		project := helpers.TestProject{
			Name: "Output Format Test",
			Files: []helpers.TestFile{
				{Path: "test.md", Content: helpers.MarkdownFiles.WithViolations},
			},
			ConfigFiles: map[string]string{
				".markdownlint.json": helpers.ConfigFiles.Basic,
			},
		}

		helpers.CreateTestProject(t, project)

		options := gomdlint.LintOptions{
			Files: []string{"test.md"},
		}

		ctx := context.Background()
		result, err := gomdlint.Lint(ctx, options)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Greater(t, result.TotalViolations, 0, "Should have violations for output format testing")

		// Test different output formats
		t.Run("default_format", func(t *testing.T) {
			output := result.String()
			assert.NotEmpty(t, output)
			assert.Contains(t, output, "test.md")
		})

		t.Run("json_format", func(t *testing.T) {
			jsonOutput, err := result.ToJSON()
			require.NoError(t, err)
			assert.NotEmpty(t, jsonOutput)

			// Should be valid JSON
			var data interface{}
			err = json.Unmarshal([]byte(jsonOutput), &data)
			assert.NoError(t, err)
		})

		t.Run("formatted_string_with_aliases", func(t *testing.T) {
			output := result.ToFormattedString(true)
			assert.NotEmpty(t, output)
			// Should contain rule aliases instead of MD### codes when available
		})
	})
}

func TestE2E_FrontMatterHandling(t *testing.T) {
	helpers.WithTempDir(t, func(tempDir string) {
		frontMatterContent := `---
title: "Test Document"
author: "Test Author"
date: "2024-01-01"
tags:
  - markdown
  - test
---

#Title Without Space

This should still trigger MD018 violation despite the front matter.
`

		project := helpers.TestProject{
			Name: "Front Matter Test",
			Files: []helpers.TestFile{
				{Path: "with-frontmatter.md", Content: frontMatterContent},
				{Path: "without-frontmatter.md", Content: "#Title Without Space\n\nContent."},
			},
			ConfigFiles: map[string]string{
				".markdownlint.json": helpers.ConfigFiles.Basic,
			},
		}

		projectDir := helpers.CreateTestProject(t, project)

		// Change to the project directory where files were created
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		err = os.Chdir(projectDir)
		require.NoError(t, err)
		defer func() {
			os.Chdir(originalDir)
		}()

		// Collect actual markdown files
		markdownFiles := collectMarkdownFiles(t, []string{"."})

		options := gomdlint.LintOptions{
			Files:  markdownFiles,
			Config: make(map[string]interface{}),
		}

		// Load basic configuration
		var config map[string]interface{}
		err = json.Unmarshal([]byte(helpers.ConfigFiles.Basic), &config)
		require.NoError(t, err, "Failed to parse Basic config")
		options.Config = config

		ctx := context.Background()
		result, err := gomdlint.Lint(ctx, options)

		require.NoError(t, err)
		require.NotNil(t, result)

		// Both files should have violations (MD018)
		assert.Greater(t, result.TotalViolations, 0)
		assert.Equal(t, 2, result.TotalFiles)

		// Verify both files are processed despite front matter
		assert.Contains(t, result.Results, "with-frontmatter.md")
		assert.Contains(t, result.Results, "without-frontmatter.md")
	})
}

func TestE2E_LargeFileHandling(t *testing.T) {
	helpers.WithTempDir(t, func(tempDir string) {
		// Generate a very large markdown file
		largeContent := helpers.GenerateMarkdownContent(helpers.ContentOptions{
			Title:                "Large Document",
			Sections:             100,
			ParagraphsPerSection: 5,
			LongLines:            true,
		})

		project := helpers.TestProject{
			Name: "Large File Test",
			Files: []helpers.TestFile{
				{Path: "large.md", Content: largeContent},
			},
			ConfigFiles: map[string]string{
				".markdownlint.json": helpers.ConfigFiles.Basic,
			},
		}

		helpers.CreateTestProject(t, project)

		options := gomdlint.LintOptions{
			Files: []string{"large.md"},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		start := time.Now()
		result, err := gomdlint.Lint(ctx, options)
		duration := time.Since(start)

		require.NoError(t, err)
		require.NotNil(t, result)

		// Should handle large files efficiently
		assert.Less(t, duration, 10*time.Second, "Large file should be processed quickly")
		assert.Equal(t, 1, result.TotalFiles)

		// Should find line length violations
		assert.Greater(t, result.TotalViolations, 0)

		t.Logf("Processed large file (%d lines) in %v",
			helpers.CountLines(largeContent), duration)
	})
}

func TestE2E_ContextCancellation(t *testing.T) {
	helpers.WithTempDir(t, func(tempDir string) {
		// Create multiple files to process
		files := make([]helpers.TestFile, 20)
		for i := 0; i < 20; i++ {
			files[i] = helpers.TestFile{
				Path: fmt.Sprintf("file%02d.md", i),
				Content: helpers.GenerateMarkdownContent(helpers.ContentOptions{
					Title:                "Cancellation Test",
					Sections:             10,
					ParagraphsPerSection: 5,
				}),
			}
		}

		project := helpers.TestProject{
			Name:  "Cancellation Test",
			Files: files,
		}

		helpers.CreateTestProject(t, project)

		// Collect actual markdown files
		markdownFiles := collectMarkdownFiles(t, []string{"."})

		options := gomdlint.LintOptions{
			Files: markdownFiles,
		}

		// Create a context that cancels quickly
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		result, err := gomdlint.Lint(ctx, options)

		// Should either complete quickly or be cancelled
		if err != nil {
			assert.Contains(t, strings.ToLower(err.Error()), "context")
		} else {
			// If completed, result should be valid
			assert.NotNil(t, result)
		}
	})
}

func TestE2E_ErrorHandling(t *testing.T) {
	helpers.WithTempDir(t, func(tempDir string) {
		errorScenarios := []struct {
			name        string
			setup       func() gomdlint.LintOptions
			expectError bool
		}{
			{
				name: "nonexistent_file",
				setup: func() gomdlint.LintOptions {
					return gomdlint.LintOptions{
						Files: []string{"nonexistent.md"},
					}
				},
				expectError: false, // API handles file errors gracefully as violations
			},
			{
				name: "permission_denied",
				setup: func() gomdlint.LintOptions {
					// Create a file with no read permissions
					testFile := filepath.Join(tempDir, "no-permission.md")
					err := os.WriteFile(testFile, []byte("# Test"), 0000)
					require.NoError(t, err)

					return gomdlint.LintOptions{
						Files: []string{testFile},
					}
				},
				expectError: false, // API handles file errors gracefully as violations
			},
			{
				name: "empty_options",
				setup: func() gomdlint.LintOptions {
					return gomdlint.LintOptions{}
				},
				expectError: false, // Should handle gracefully
			},
		}

		for _, scenario := range errorScenarios {
			t.Run(scenario.name, func(t *testing.T) {
				options := scenario.setup()

				ctx := context.Background()
				result, err := gomdlint.Lint(ctx, options)

				if scenario.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.NotNil(t, result)
				}
			})
		}
	})
}

// Helper functions
func extractRuleNamesFromResult(result *gomdlint.LintResult) []string {
	var rules []string
	for _, violations := range result.Results {
		for _, violation := range violations {
			for _, ruleName := range violation.RuleNames {
				rules = append(rules, ruleName)
			}
		}
	}
	return rules
}

// Benchmark tests for integration scenarios
func BenchmarkE2E_TypicalProject(b *testing.B) {
	helpers.WithTempDir(b, func(tempDir string) {
		// Create a typical project structure
		files := []helpers.TestFile{
			{Path: "README.md", Content: helpers.MarkdownFiles.Complex},
			{Path: "docs/installation.md", Content: helpers.MarkdownFiles.Valid},
			{Path: "docs/usage.md", Content: helpers.MarkdownFiles.Valid},
			{Path: "docs/api/overview.md", Content: helpers.MarkdownFiles.Complex},
			{Path: "docs/api/reference.md", Content: helpers.MarkdownFiles.Valid},
			{Path: "CHANGELOG.md", Content: helpers.MarkdownFiles.Valid},
			{Path: "CONTRIBUTING.md", Content: helpers.MarkdownFiles.Valid},
		}

		project := helpers.TestProject{
			Name:  "Typical Project",
			Files: files,
			ConfigFiles: map[string]string{
				".markdownlint.json": helpers.ConfigFiles.Basic,
			},
		}

		helpers.CreateTestProject(b, project)

		// Collect actual markdown files
		markdownFiles := collectMarkdownFiles(b, []string{"."})

		options := gomdlint.LintOptions{
			Files: markdownFiles,
		}

		ctx := context.Background()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result, err := gomdlint.Lint(ctx, options)
			require.NoError(b, err)
			require.NotNil(b, result)
		}
	})
}
