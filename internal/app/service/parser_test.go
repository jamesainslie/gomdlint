package service

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// Test scenarios for parser service following club/ standards
type parserTestScenario struct {
	name           string
	content        string
	identifier     string
	expectError    bool
	expectedTokens int
	minTokens      int
	expectedTypes  []string // Token types we expect to find
}

// Helper functions
func createTestParserService(t testing.TB) *ParserService {
	t.Helper()

	parser := NewParserService()
	require.NotNil(t, parser)

	return parser
}

func assertParseResult(t *testing.T, result functional.Result[[]value.Token], scenario parserTestScenario) {
	t.Helper()

	if scenario.expectError {
		require.True(t, result.IsErr(), "Expected error but got success")
		return
	}

	if result.IsErr() {
		t.Fatalf("Expected success but got error: %v", result.Error())
	}
	require.True(t, result.IsOk(), "Expected success")

	tokens := result.Unwrap()
	require.NotNil(t, tokens)

	if scenario.expectedTokens > 0 {
		assert.Equal(t, scenario.expectedTokens, len(tokens))
	}

	if scenario.minTokens > 0 {
		assert.GreaterOrEqual(t, len(tokens), scenario.minTokens)
	}

	// Check for expected token types
	if len(scenario.expectedTypes) > 0 {
		foundTypes := make(map[string]bool)
		for _, token := range tokens {
			foundTypes[token.Type.String()] = true
		}

		for _, expectedType := range scenario.expectedTypes {
			assert.True(t, foundTypes[expectedType], "Expected token type %s not found", expectedType)
		}
	}
}

func TestNewParserService(t *testing.T) {
	parser := NewParserService()

	assert.NotNil(t, parser, "Parser service should not be nil")

	// Verify internal state is properly initialized
	assert.NotNil(t, parser, "Parser should be initialized")
}

func TestParserService_ParseDocument_Scenarios(t *testing.T) {
	scenarios := []parserTestScenario{
		{
			name:          "simple heading",
			content:       "# Main Title\n",
			identifier:    "simple.md",
			expectError:   false,
			minTokens:     1,
			expectedTypes: []string{"atxHeading"},
		},
		{
			name:          "heading with paragraph",
			content:       "# Title\n\nThis is a paragraph.\n",
			identifier:    "with-para.md",
			expectError:   false,
			minTokens:     2,
			expectedTypes: []string{"atxHeading", "paragraph"},
		},
		{
			name:          "multiple headings",
			content:       "# H1\n\n## H2\n\n### H3\n",
			identifier:    "multi-headings.md",
			expectError:   false,
			minTokens:     3,
			expectedTypes: []string{"atxHeading"},
		},
		{
			name:          "list items",
			content:       "# Title\n\n- Item 1\n- Item 2\n- Item 3\n",
			identifier:    "list.md",
			expectError:   false,
			minTokens:     2,
			expectedTypes: []string{"atxHeading", "list"},
		},
		{
			name:          "code block",
			content:       "# Title\n\n```go\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```\n",
			identifier:    "code.md",
			expectError:   false,
			minTokens:     2,
			expectedTypes: []string{"atxHeading", "codeFenced"},
		},
		{
			name:          "blockquote",
			content:       "# Title\n\n> This is a quote\n> Continued quote\n",
			identifier:    "quote.md",
			expectError:   false,
			minTokens:     2,
			expectedTypes: []string{"atxHeading", "blockQuote"},
		},
		{
			name:          "horizontal rule",
			content:       "# Title\n\n---\n\nAfter rule.\n",
			identifier:    "hr.md",
			expectError:   false,
			minTokens:     3,
			expectedTypes: []string{"atxHeading", "thematicBreak", "paragraph"},
		},
		{
			name:          "table",
			content:       "# Title\n\n| Col1 | Col2 |\n|------|------|\n| A    | B    |\n",
			identifier:    "table.md",
			expectError:   false,
			minTokens:     2,
			expectedTypes: []string{"atxHeading", "table"},
		},
		{
			name:          "mixed inline formatting",
			content:       "# Title\n\nThis has **bold**, *italic*, and `code` formatting.\n",
			identifier:    "inline.md",
			expectError:   false,
			minTokens:     2,
			expectedTypes: []string{"atxHeading", "paragraph"},
		},
		{
			name:        "empty content",
			content:     "",
			identifier:  "empty.md",
			expectError: false,
			minTokens:   0,
		},
		{
			name:        "only whitespace",
			content:     "   \n\t\n   \n",
			identifier:  "whitespace.md",
			expectError: false,
			minTokens:   0,
		},
		{
			name: "complex document",
			content: `# Main Title

## Introduction

This is the introduction paragraph with some **bold text** and *italic text*.

### Code Example

Here's a code example:

` + "```python\ndef hello():\n    print(\"Hello, world!\")\n```" + `

### List of Features

- Feature 1 with [link](http://example.com)
- Feature 2 with ` + "`inline code`" + `
- Feature 3

> This is an important note in a blockquote.

---

## Conclusion

Final thoughts here.
`,
			identifier:    "complex.md",
			expectError:   false,
			minTokens:     10,
			expectedTypes: []string{"atxHeading", "paragraph", "codeFenced", "list", "blockQuote", "thematicBreak"},
		},
		{
			name:          "malformed markdown",
			content:       "# Title\n\n[Incomplete link",
			identifier:    "malformed.md",
			expectError:   false, // Parser should handle gracefully
			minTokens:     1,
			expectedTypes: []string{"atxHeading"},
		},
		{
			name:          "very large document",
			content:       "# Title\n\n" + strings.Repeat("Paragraph content line.\n\n", 1000),
			identifier:    "large.md",
			expectError:   false,
			minTokens:     100,
			expectedTypes: []string{"atxHeading", "paragraph"},
		},
	}

	ctx := context.Background()

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			parser := createTestParserService(t)

			result := parser.ParseDocument(ctx, scenario.content, scenario.identifier)
			assertParseResult(t, result, scenario)
		})
	}
}

func TestParserService_Performance(t *testing.T) {
	ctx := context.Background()
	parser := createTestParserService(t)

	// Generate large content for performance testing
	largeContent := generateLargeMarkdownContent(5000)

	result := parser.ParseDocument(ctx, largeContent, "performance.md")
	require.True(t, result.IsOk(), "Parsing large content should succeed")

	tokens := result.Unwrap()
	assert.Greater(t, len(tokens), 100, "Large content should generate many tokens")
}

func TestParserService_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	parser := createTestParserService(t)

	// Cancel context immediately
	cancel()

	// Try to parse content (should either complete quickly or be cancelled)
	content := "# Title\n\n" + strings.Repeat("Content.\n", 10000)
	result := parser.ParseDocument(ctx, content, "cancelled.md")

	// Either succeeds (fast parsing) or fails with context error
	if result.IsErr() {
		err := result.Error()
		assert.Contains(t, strings.ToLower(err.Error()), "context")
	} else {
		// If succeeded, tokens should be valid
		tokens := result.Unwrap()
		assert.NotNil(t, tokens)
	}
}

func TestParserService_ClearCaches(t *testing.T) {
	parser := createTestParserService(t)

	// The ClearCaches method should exist and not panic
	require.NotPanics(t, func() {
		parser.ClearCaches()
	}, "ClearCaches should not panic")
}

func TestParserService_EdgeCases(t *testing.T) {
	ctx := context.Background()
	parser := createTestParserService(t)

	edgeCases := []struct {
		name       string
		content    string
		shouldWork bool
	}{
		{
			name:       "null bytes",
			content:    "# Title\n\nContent with \x00 null byte",
			shouldWork: true, // Should handle gracefully
		},
		{
			name:       "unicode content",
			content:    "# 标题\n\n这是中文内容。\n\n- 项目 1\n- 项目 2\n",
			shouldWork: true,
		},
		{
			name:       "emoji content",
			content:    "# Title 🚀\n\nContent with emojis 😊 and symbols ⭐\n",
			shouldWork: true,
		},
		{
			name:       "mixed line endings",
			content:    "# Title\r\n\r\nWindows line endings\r\nMixed with\nUnix endings",
			shouldWork: true,
		},
		{
			name:       "very long line",
			content:    "# Title\n\n" + strings.Repeat("a", 10000) + "\n",
			shouldWork: true,
		},
		{
			name:       "deeply nested lists",
			content:    "# Title\n\n- Level 1\n  - Level 2\n    - Level 3\n      - Level 4\n        - Level 5\n",
			shouldWork: true,
		},
		{
			name:       "html mixed with markdown",
			content:    "# Title\n\n<div>HTML content</div>\n\n**Bold** markdown\n",
			shouldWork: true,
		},
		{
			name:       "malformed tables",
			content:    "# Title\n\n| Col1 | Col2\n|------|------\n| A    | B\n| C\n",
			shouldWork: true, // Should handle malformed tables gracefully
		},
	}

	for _, tc := range edgeCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parser.ParseDocument(ctx, tc.content, tc.name+".md")

			if tc.shouldWork {
				require.True(t, result.IsOk(), "Parsing should succeed for %s", tc.name)
				tokens := result.Unwrap()
				assert.NotNil(t, tokens)
			} else {
				// Currently all cases should work - adjust if needed
				require.True(t, result.IsOk(), "All edge cases should be handled gracefully")
			}
		})
	}
}

// Benchmark tests
func BenchmarkParserService_ParseDocument(b *testing.B) {
	ctx := context.Background()
	parser := createTestParserService(b)

	content := generateLargeMarkdownContent(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := parser.ParseDocument(ctx, content, "benchmark.md")
		require.True(b, result.IsOk())
	}
}

func BenchmarkParserService_ParseSimple(b *testing.B) {
	ctx := context.Background()
	parser := createTestParserService(b)

	content := "# Title\n\nSimple paragraph content.\n\n- List item\n"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := parser.ParseDocument(ctx, content, "simple.md")
		require.True(b, result.IsOk())
	}
}

func BenchmarkParserService_ParseComplex(b *testing.B) {
	ctx := context.Background()
	parser := createTestParserService(b)

	content := `# Complex Document

## Section with Code

` + "```go\nfunc example() {\n    return \"hello\"\n}\n```" + `

## Section with Table

| Column 1 | Column 2 | Column 3 |
|----------|----------|----------|
| Data 1   | Data 2   | Data 3   |

## Section with List

- **Bold item** with [link](http://example.com)
- *Italic item* with ` + "`inline code`" + `
- Regular item

> Important blockquote here.

---

Final section.
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := parser.ParseDocument(ctx, content, "complex.md")
		require.True(b, result.IsOk())
	}
}

// Helper function to generate large markdown content
func generateLargeMarkdownContent(sections int) string {
	var content strings.Builder
	content.WriteString("# Large Document\n\n")

	for i := 0; i < sections; i++ {
		content.WriteString(fmt.Sprintf("## Section %d\n\n", i+1))
		content.WriteString("This is the content of section with some **bold** and *italic* text.\n\n")

		if i%3 == 0 {
			content.WriteString("```go\nfunc example() {\n    return \"code\"\n}\n```\n\n")
		}

		if i%5 == 0 {
			content.WriteString("- List item 1\n- List item 2\n- List item 3\n\n")
		}

		if i%7 == 0 {
			content.WriteString("> This is a blockquote section.\n\n")
		}
	}

	return content.String()
}
