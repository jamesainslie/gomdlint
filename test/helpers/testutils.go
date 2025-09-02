package helpers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestFile represents a test file with its content
type TestFile struct {
	Path    string
	Content string
	Mode    os.FileMode
}

// TestProject represents a complete test project structure
type TestProject struct {
	Name        string
	BaseDir     string
	Files       []TestFile
	ConfigFiles map[string]string // config filename -> content
}

// CreateTestProject creates a temporary test project with the specified structure
func CreateTestProject(t testing.TB, project TestProject) string {
	t.Helper()

	baseDir := t.TempDir()
	if project.BaseDir != "" {
		baseDir = filepath.Join(baseDir, project.BaseDir)
		err := os.MkdirAll(baseDir, 0755)
		require.NoError(t, err)
	}

	// Create regular files
	for _, file := range project.Files {
		fullPath := filepath.Join(baseDir, file.Path)

		// Create directory if needed
		dir := filepath.Dir(fullPath)
		if dir != baseDir {
			err := os.MkdirAll(dir, 0755)
			require.NoError(t, err)
		}

		// Set default mode if not specified
		mode := file.Mode
		if mode == 0 {
			mode = 0644
		}

		err := os.WriteFile(fullPath, []byte(file.Content), mode)
		require.NoError(t, err)
	}

	// Create config files
	for filename, content := range project.ConfigFiles {
		configPath := filepath.Join(baseDir, filename)
		err := os.WriteFile(configPath, []byte(content), 0644)
		require.NoError(t, err)
	}

	return baseDir
}

// MarkdownFiles provides common markdown file templates
var MarkdownFiles = struct {
	Valid           string
	InvalidHeading  string
	WithViolations  string
	Complex         string
	Empty           string
	WithFrontMatter string
}{
	Valid: `# Project Title

## Introduction

This is a well-formatted markdown file with proper structure.

### Features

- Feature 1: Great functionality
- Feature 2: Amazing performance
- Feature 3: Excellent usability

#### Code Example

` + "```go\nfunc main() {\n    fmt.Println(\"Hello, World!\")\n}\n```" + `

## Conclusion

This document follows all markdown best practices.
`,

	InvalidHeading: `#Title Without Space

This file has a heading without space after the hash, violating MD018.

##Another Bad Heading

Content here.
`,

	WithViolations: `#Bad Title

This file has multiple violations:


Too many blank lines above (MD012).

This line is way too long and exceeds the default line length limit which should trigger MD013 violation in most configurations.

- List item
-List item without space (MD004)

	Tab character above (MD010)

Line with trailing spaces   

> Block quote
>Missing space (MD027)
`,

	Complex: `---
title: "Complex Document"
author: "Test Author"
date: "2024-01-01"
---

# Complex Markdown Document

This document contains various markdown elements to test comprehensive linting.

## Table of Contents

- [Introduction](#introduction)
- [Code Examples](#code-examples)
- [Lists](#lists)
- [Tables](#tables)

## Introduction

This section introduces the document with **bold text**, *italic text*, and ` + "`inline code`" + `.

### Subsection

Here's a [link to example](https://example.com) and an image:

![Alt text](image.png "Image title")

## Code Examples

### JavaScript

` + "```javascript\nfunction hello(name) {\n    console.log(`Hello, ${name}!`);\n}\n```" + `

### Python

` + "```python\ndef greet(name):\n    print(f\"Hello, {name}!\")\n```" + `

## Lists

### Unordered List

- First item
  - Nested item
  - Another nested item
- Second item
- Third item

### Ordered List

1. First step
2. Second step
   1. Sub-step
   2. Another sub-step
3. Third step

## Tables

| Column 1 | Column 2 | Column 3 |
|----------|----------|----------|
| Row 1    | Data     | More     |
| Row 2    | Info     | Data     |

## Blockquotes

> This is a blockquote with **bold** and *italic* text.
>
> It can span multiple lines and contain other elements.

---

## Final Section

This document demonstrates various markdown features and potential violations.
`,

	Empty: ``,

	WithFrontMatter: `---
title: "Document with Front Matter"
layout: page
published: true
tags:
  - markdown
  - test
---

# Document with Front Matter

This document has YAML front matter that should be handled properly.

The front matter should be ignored during linting.
`,
}

// ConfigFiles provides common configuration file templates
var ConfigFiles = struct {
	Basic            string
	Strict           string
	Relaxed          string
	CustomLineLength string
	DisableRules     string
}{
	Basic: `{
	"MD009": false,
	"MD013": {
		"line_length": 100
	},
	"MD031": false,
	"MD037": false,
	"MD040": false,
	"MD041": false
}`,

	Strict: `{
	"MD001": true,
	"MD003": {"style": "atx"},
	"MD004": {"style": "dash"},
	"MD009": false,
	"MD013": {
		"line_length": 100,
		"tables": false
	},
	"MD018": true,
	"MD022": true,
	"MD025": true,
	"MD031": false,
	"MD036": false,
	"MD037": false,
	"MD040": false
}`,

	Relaxed: `{
	"MD001": false,
	"MD009": false,
	"MD013": {
		"line_length": 120
	},
	"MD018": false,
	"MD031": false,
	"MD036": false,
	"MD037": false,
	"MD040": false,
	"MD041": false
}`,

	CustomLineLength: `{
	"MD009": false,
	"MD012": false,
	"MD013": {
		"line_length": 600,
		"tables": true,
		"code_blocks": false
	},
	"MD031": false,
	"MD036": false,
	"MD037": false,
	"MD040": false,
	"MD041": false,
	"MD047": false
}`,

	DisableRules: `{
	"MD001": false,
	"MD012": false,
	"MD013": false,
	"MD018": false,
	"MD022": false,
	"MD041": false
}`,
}

// TestScenario represents a complete test scenario with setup and expectations
type TestScenario struct {
	Name             string
	Description      string
	Files            []TestFile
	Config           string
	Args             []string
	Flags            map[string]interface{}
	ExpectError      bool
	ExpectViolations bool
	MinViolations    int
	MaxViolations    int
	ExpectedRules    []string
	ExpectedOutput   []string
	ExpectedExitCode int
}

// CreateTestScenarios returns a comprehensive set of test scenarios
func CreateTestScenarios() []TestScenario {
	return []TestScenario{
		{
			Name:        "Valid Markdown Files",
			Description: "Test with properly formatted markdown files",
			Files: []TestFile{
				{Path: "README.md", Content: MarkdownFiles.Valid},
				{Path: "docs/guide.md", Content: MarkdownFiles.Valid},
			},
			Config:           ConfigFiles.Basic,
			Args:             []string{"."},
			ExpectError:      false,
			ExpectViolations: false,
			ExpectedExitCode: 0,
		},
		{
			Name:        "Files with Violations",
			Description: "Test with markdown files containing various violations",
			Files: []TestFile{
				{Path: "bad1.md", Content: MarkdownFiles.InvalidHeading},
				{Path: "bad2.md", Content: MarkdownFiles.WithViolations},
			},
			Config:           ConfigFiles.Basic,
			Args:             []string{"."},
			ExpectError:      false,
			ExpectViolations: true,
			MinViolations:    2,
			ExpectedRules:    []string{"MD018", "MD012"},
			ExpectedExitCode: 1,
		},
		{
			Name:        "Mixed Valid and Invalid",
			Description: "Test with a mix of valid and invalid files",
			Files: []TestFile{
				{Path: "good.md", Content: MarkdownFiles.Valid},
				{Path: "bad.md", Content: MarkdownFiles.InvalidHeading},
				{Path: "empty.md", Content: MarkdownFiles.Empty},
			},
			Config:           ConfigFiles.Basic,
			Args:             []string{"."},
			ExpectError:      false,
			ExpectViolations: true,
			MinViolations:    1,
			ExpectedRules:    []string{"MD018"},
			ExpectedExitCode: 1,
		},
		{
			Name:        "Strict Configuration",
			Description: "Test with strict linting configuration",
			Files: []TestFile{
				{Path: "test.md", Content: MarkdownFiles.Complex},
			},
			Config:           ConfigFiles.Strict,
			Args:             []string{"test.md"},
			ExpectError:      false,
			ExpectViolations: false, // Complex file should pass strict rules
			ExpectedExitCode: 0,
		},
		{
			Name:        "Relaxed Configuration",
			Description: "Test with relaxed linting configuration",
			Files: []TestFile{
				{Path: "test.md", Content: MarkdownFiles.InvalidHeading},
			},
			Config:           ConfigFiles.Relaxed,
			Args:             []string{"test.md"},
			ExpectError:      false,
			ExpectViolations: false, // Should pass with relaxed config
			ExpectedExitCode: 0,
		},
		{
			Name:        "No Configuration",
			Description: "Test without any configuration file (use defaults)",
			Files: []TestFile{
				{Path: "test.md", Content: MarkdownFiles.WithViolations},
			},
			Config:           "", // No config
			Args:             []string{"test.md"},
			Flags:            map[string]interface{}{"no-config": true},
			ExpectError:      false,
			ExpectViolations: true,
			MinViolations:    1,
			ExpectedExitCode: 1,
		},
		{
			Name:        "Custom Line Length",
			Description: "Test custom line length configuration",
			Files: []TestFile{
				{Path: "long-lines.md", Content: "# Title\n\n" + strings.Repeat("Very long line content that exceeds normal limits. ", 10)},
			},
			Config:           ConfigFiles.CustomLineLength,
			Args:             []string{"long-lines.md"},
			ExpectError:      false,
			ExpectViolations: false, // Should pass with longer line length
			ExpectedExitCode: 0,
		},
		{
			Name:        "Front Matter Handling",
			Description: "Test handling of front matter in markdown files",
			Files: []TestFile{
				{Path: "with-frontmatter.md", Content: MarkdownFiles.WithFrontMatter},
			},
			Config:           ConfigFiles.Basic,
			Args:             []string{"with-frontmatter.md"},
			ExpectError:      false,
			ExpectViolations: false,
			ExpectedExitCode: 0,
		},
		{
			Name:        "Nested Directory Structure",
			Description: "Test with nested directory structure",
			Files: []TestFile{
				{Path: "docs/api/README.md", Content: MarkdownFiles.Valid},
				{Path: "docs/guides/setup.md", Content: MarkdownFiles.Valid},
				{Path: "docs/guides/advanced.md", Content: MarkdownFiles.InvalidHeading},
				{Path: "examples/basic.md", Content: MarkdownFiles.Complex},
			},
			Config:           ConfigFiles.Basic,
			Args:             []string{"docs/", "examples/"},
			ExpectError:      false,
			ExpectViolations: true,
			MinViolations:    1,
			ExpectedRules:    []string{"MD018"},
			ExpectedExitCode: 1,
		},
		{
			Name:        "Large File Set",
			Description: "Test performance with many files",
			Files: func() []TestFile {
				files := make([]TestFile, 20)
				for i := 0; i < 20; i++ {
					content := MarkdownFiles.Valid
					if i%5 == 0 {
						content = MarkdownFiles.InvalidHeading // Some files have violations
					}
					files[i] = TestFile{
						Path:    fmt.Sprintf("file%02d.md", i),
						Content: content,
					}
				}
				return files
			}(),
			Config:           ConfigFiles.Basic,
			Args:             []string{"."},
			ExpectError:      false,
			ExpectViolations: true,
			MinViolations:    1,
			ExpectedExitCode: 1,
		},
	}
}

// AssertViolations is a helper function to assert violation expectations
func AssertViolations(t *testing.T, violations []interface{}, scenario TestScenario) {
	t.Helper()

	if scenario.ExpectViolations {
		if len(violations) == 0 {
			t.Errorf("Expected violations but found none in scenario: %s", scenario.Name)
			return
		}

		if scenario.MinViolations > 0 && len(violations) < scenario.MinViolations {
			t.Errorf("Expected at least %d violations but found %d in scenario: %s",
				scenario.MinViolations, len(violations), scenario.Name)
		}

		if scenario.MaxViolations > 0 && len(violations) > scenario.MaxViolations {
			t.Errorf("Expected at most %d violations but found %d in scenario: %s",
				scenario.MaxViolations, len(violations), scenario.Name)
		}
	} else {
		if len(violations) > 0 {
			t.Errorf("Expected no violations but found %d in scenario: %s",
				len(violations), scenario.Name)
		}
	}
}

// AssertOutput is a helper function to assert output expectations
func AssertOutput(t *testing.T, output string, scenario TestScenario) {
	t.Helper()

	for _, expected := range scenario.ExpectedOutput {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q but it didn't in scenario: %s",
				expected, scenario.Name)
		}
	}
}

// WithTempDir executes a test function within a temporary directory
func WithTempDir(t testing.TB, fn func(tempDir string)) {
	t.Helper()

	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	require.NoError(t, err)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	defer func() {
		os.Chdir(originalDir)
	}()

	fn(tempDir)
}

// GenerateMarkdownContent generates markdown content with specified characteristics
func GenerateMarkdownContent(options ContentOptions) string {
	var content strings.Builder

	if options.Title != "" {
		if options.InvalidTitle {
			content.WriteString("#" + options.Title + "\n\n")
		} else {
			content.WriteString("# " + options.Title + "\n\n")
		}
	}

	for i := 0; i < options.Sections; i++ {
		if options.InvalidHeadings && i%2 == 0 {
			content.WriteString(fmt.Sprintf("##Section %d\n\n", i+1))
		} else {
			content.WriteString(fmt.Sprintf("## Section %d\n\n", i+1))
		}

		// Add paragraph content
		for j := 0; j < options.ParagraphsPerSection; j++ {
			paragraph := fmt.Sprintf("This is paragraph %d in section %d.", j+1, i+1)

			if options.LongLines {
				paragraph = strings.Repeat(paragraph+" ", 10)
			}

			if options.TrailingSpaces && j%2 == 0 {
				paragraph += "   "
			}

			if options.TabCharacters && j%3 == 0 {
				paragraph = strings.Replace(paragraph, " ", "\t", 1)
			}

			content.WriteString(paragraph + "\n\n")
		}

		// Add excessive blank lines if requested
		if options.ExcessiveBlankLines && i%3 == 0 {
			content.WriteString("\n\n\n")
		}
	}

	return content.String()
}

// ContentOptions defines options for generating markdown content
type ContentOptions struct {
	Title                string
	Sections             int
	ParagraphsPerSection int
	InvalidTitle         bool
	InvalidHeadings      bool
	LongLines            bool
	TrailingSpaces       bool
	TabCharacters        bool
	ExcessiveBlankLines  bool
}

// DefaultContentOptions returns default content generation options
func DefaultContentOptions() ContentOptions {
	return ContentOptions{
		Title:                "Test Document",
		Sections:             3,
		ParagraphsPerSection: 2,
		InvalidTitle:         false,
		InvalidHeadings:      false,
		LongLines:            false,
		TrailingSpaces:       false,
		TabCharacters:        false,
		ExcessiveBlankLines:  false,
	}
}

// FileExists checks if a file exists at the given path
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// CreateTempConfig creates a temporary configuration file with the given content
func CreateTempConfig(t *testing.T, content, filename string) string {
	t.Helper()

	if filename == "" {
		filename = ".markdownlint.json"
	}

	tempFile := filepath.Join(t.TempDir(), filename)
	err := os.WriteFile(tempFile, []byte(content), 0644)
	require.NoError(t, err)

	return tempFile
}

// NormalizeLineEndings normalizes line endings for cross-platform compatibility
func NormalizeLineEndings(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	return content
}

// CountLines counts the number of lines in a string
func CountLines(content string) int {
	if content == "" {
		return 0
	}
	return strings.Count(content, "\n") + 1
}

// ExtractRuleNames extracts rule names from violation messages
func ExtractRuleNames(violationMessages []string) []string {
	var rules []string
	for _, msg := range violationMessages {
		// Simple extraction - in real implementation would be more sophisticated
		if strings.Contains(msg, "MD") {
			parts := strings.Fields(msg)
			for _, part := range parts {
				if strings.HasPrefix(part, "MD") && len(part) <= 6 {
					rules = append(rules, part)
					break
				}
			}
		}
	}
	return rules
}
