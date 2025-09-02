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

// Test scenarios for rule engine following club/ standards
type ruleEngineTestScenario struct {
	name              string
	tokens            []value.Token
	lines             []string
	identifier        string
	expectError       bool
	expectViolations  bool
	minViolations     int
	maxViolations     int
	expectedRuleNames []string
}

// Helper functions
func createTestRuleEngine(t testing.TB) *RuleEngine {
	t.Helper()

	engine, err := NewRuleEngine()
	require.NoError(t, err)
	require.NotNil(t, engine)

	return engine
}

func createTestTokens(content string) []value.Token {
	// This is a simplified token creation for testing
	// In reality, tokens would come from the parser
	lines := strings.Split(content, "\n")
	tokens := make([]value.Token, 0)

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var tokenType value.TokenType
		var tokenContent = line

		if strings.HasPrefix(line, "#") {
			tokenType = value.TokenTypeHeading
		} else if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") || strings.HasPrefix(line, "+") {
			tokenType = value.TokenTypeListItem
		} else if strings.HasPrefix(line, ">") {
			tokenType = value.TokenTypeBlockquote
		} else if strings.HasPrefix(line, "```") {
			tokenType = value.TokenTypeCodeBlock
		} else if line == "---" || line == "***" {
			tokenType = value.TokenTypeHorizontalRule
		} else {
			tokenType = value.TokenTypeParagraph
		}

		token := value.NewToken(
			tokenType,
			tokenContent,
			value.NewPosition(i+1, 1),
			value.NewPosition(i+1, len(line)+1),
		)
		tokens = append(tokens, token)
	}

	return tokens
}

func assertRuleEngineResult(t *testing.T, result functional.Result[[]value.Violation], scenario ruleEngineTestScenario) {
	t.Helper()

	if scenario.expectError {
		require.True(t, result.IsErr(), "Expected error but got success")
		return
	}

	if !result.IsOk() {
		t.Fatalf("Expected success but got error: %v", result.Error())
	}

	violations := result.Unwrap()
	// Note: violations can be nil slice, which is valid (equivalent to empty slice)

	if scenario.expectViolations {
		assert.Greater(t, len(violations), 0, "Expected violations but got none")

		if scenario.minViolations > 0 {
			assert.GreaterOrEqual(t, len(violations), scenario.minViolations)
		}

		if scenario.maxViolations > 0 {
			assert.LessOrEqual(t, len(violations), scenario.maxViolations)
		}

		// Check for specific rule violations
		if len(scenario.expectedRuleNames) > 0 {
			foundRules := make(map[string]bool)
			for _, violation := range violations {
				for _, ruleName := range violation.RuleNames {
					foundRules[ruleName] = true
				}
			}

			for _, expectedRule := range scenario.expectedRuleNames {
				assert.True(t, foundRules[expectedRule], "Expected rule %s not found in violations", expectedRule)
			}
		}
	} else {
		if len(violations) > 0 {
			for i, violation := range violations {
				t.Logf("Unexpected violation %d: Rule=%v, Description=%s, Line=%d, Detail=%s",
					i+1, violation.RuleNames, violation.RuleDescription, violation.LineNumber, violation.ErrorDetail)
			}
		}
		assert.Equal(t, 0, len(violations), "Expected no violations but got %d", len(violations))
	}
}

func TestNewRuleEngine(t *testing.T) {
	engine, err := NewRuleEngine()

	require.NoError(t, err, "Creating rule engine should not error")
	require.NotNil(t, engine, "Rule engine should not be nil")

	// Verify that rules are loaded
	stats := engine.Stats()
	require.Contains(t, stats, "total_rules")
	assert.Greater(t, stats["total_rules"], 0, "Should have loaded rules")
}

func TestRuleEngine_ConfigureRules_Scenarios(t *testing.T) {
	scenarios := []struct {
		name        string
		config      map[string]interface{}
		expectError bool
	}{
		{
			name:        "empty config",
			config:      map[string]interface{}{},
			expectError: false,
		},
		{
			name: "disable specific rule",
			config: map[string]interface{}{
				"MD001": false,
			},
			expectError: false,
		},
		{
			name: "enable specific rule",
			config: map[string]interface{}{
				"MD001": true,
			},
			expectError: false,
		},
		{
			name: "configure rule options",
			config: map[string]interface{}{
				"MD013": map[string]interface{}{
					"line_length": 120,
					"tables":      false,
				},
			},
			expectError: false,
		},
		{
			name: "mixed configuration",
			config: map[string]interface{}{
				"MD001": true,
				"MD002": false,
				"MD013": map[string]interface{}{
					"line_length": 80,
				},
				"MD041": false,
			},
			expectError: false,
		},
		{
			name: "invalid rule name",
			config: map[string]interface{}{
				"MD999": false, // Non-existent rule
			},
			expectError: false, // Should handle gracefully
		},
		{
			name: "invalid config value",
			config: map[string]interface{}{
				"MD013": "invalid", // Should be map or bool
			},
			expectError: true, // Should reject invalid config
		},
		{
			name:        "nil config",
			config:      nil,
			expectError: false,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			engine := createTestRuleEngine(t)

			err := engine.ConfigureRules(scenario.config)

			if scenario.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRuleEngine_LintDocument_Scenarios(t *testing.T) {
	scenarios := []ruleEngineTestScenario{
		{
			name:             "valid document",
			tokens:           createTestTokens("# Title\n\n## Subtitle\n\nParagraph content.\n"),
			lines:            []string{"# Title", "", "## Subtitle", "", "Paragraph content.", ""},
			identifier:       "valid.md",
			expectError:      false,
			expectViolations: false,
		},
		{
			name:              "heading without space",
			tokens:            createTestTokens("#Title\n\nContent."),
			lines:             []string{"#Title", "", "Content."},
			identifier:        "no-space.md",
			expectError:       false,
			expectViolations:  true,
			minViolations:     1,
			expectedRuleNames: []string{"MD018"},
		},
		{
			name:              "multiple violations",
			tokens:            createTestTokens("#Title\n\n\n\nContent with tab\tcharacter."),
			lines:             []string{"#Title", "", "", "", "Content with tab\tcharacter."},
			identifier:        "multiple.md",
			expectError:       false,
			expectViolations:  true,
			minViolations:     2,
			expectedRuleNames: []string{"MD018", "MD012"},
		},
		{
			name:             "empty document",
			tokens:           []value.Token{},
			lines:            []string{},
			identifier:       "empty.md",
			expectError:      false,
			expectViolations: false,
		},
		{
			name:             "only whitespace",
			tokens:           []value.Token{},
			lines:            []string{""},
			identifier:       "whitespace.md",
			expectError:      false,
			expectViolations: false,
		},
		{
			name:              "trailing whitespace",
			tokens:            createTestTokens("# Title\n\nContent with trailing spaces.\n"),
			lines:             []string{"# Title", "", "Content with trailing spaces.   ", ""}, // Add trailing spaces in lines
			identifier:        "trailing.md",
			expectError:       false,
			expectViolations:  true,
			expectedRuleNames: []string{"MD009"},
		},
	}

	ctx := context.Background()

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			engine := createTestRuleEngine(t)

			result := engine.LintDocument(ctx, scenario.tokens, scenario.lines, scenario.identifier)
			assertRuleEngineResult(t, result, scenario)
		})
	}
}

func TestRuleEngine_RuleSpecificConfigurations(t *testing.T) {
	ctx := context.Background()

	t.Run("MD013 line length configuration", func(t *testing.T) {
		engine := createTestRuleEngine(t)

		// Configure MD013 with short line length
		err := engine.ConfigureRules(map[string]interface{}{
			"MD013": map[string]interface{}{
				"line_length": 20,
			},
		})
		require.NoError(t, err)

		longLine := "This is a very long line that exceeds the configured limit"
		tokens := createTestTokens("# Title\n\n" + longLine)
		lines := []string{"# Title", "", longLine}

		result := engine.LintDocument(ctx, tokens, lines, "long-line.md")
		require.True(t, result.IsOk())

		violations := result.Unwrap()
		assert.Greater(t, len(violations), 0, "Should have line length violations")

		// Check that MD013 violation exists
		foundMD013 := false
		for _, violation := range violations {
			for _, ruleName := range violation.RuleNames {
				if ruleName == "MD013" {
					foundMD013 = true
					break
				}
			}
		}
		assert.True(t, foundMD013, "Should have MD013 violation")
	})

	t.Run("disabled rule should not trigger", func(t *testing.T) {
		engine := createTestRuleEngine(t)

		// Disable MD018 rule
		err := engine.ConfigureRules(map[string]interface{}{
			"MD018": false,
		})
		require.NoError(t, err)

		tokens := createTestTokens("#Title without space")
		lines := []string{"#Title without space"}

		result := engine.LintDocument(ctx, tokens, lines, "disabled-rule.md")
		require.True(t, result.IsOk())

		violations := result.Unwrap()

		// Should not have MD018 violations
		foundMD018 := false
		for _, violation := range violations {
			for _, ruleName := range violation.RuleNames {
				if ruleName == "MD018" {
					foundMD018 = true
					break
				}
			}
		}
		assert.False(t, foundMD018, "Should not have MD018 violation when disabled")
	})
}

func TestRuleEngine_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	engine := createTestRuleEngine(t)

	// Cancel context immediately
	cancel()

	// Create large content that might take time to process
	largeContent := strings.Repeat("# Title\n\nContent line.\n\n", 1000)
	tokens := createTestTokens(largeContent)
	lines := strings.Split(largeContent, "\n")

	result := engine.LintDocument(ctx, tokens, lines, "cancelled.md")

	// Should either complete quickly or be cancelled
	if result.IsErr() {
		err := result.Error()
		assert.Contains(t, strings.ToLower(err.Error()), "context")
	} else {
		// If completed, result should be valid
		violations := result.Unwrap()
		assert.NotNil(t, violations)
	}
}

func TestRuleEngine_Stats(t *testing.T) {
	engine := createTestRuleEngine(t)

	stats := engine.Stats()
	require.NotNil(t, stats)

	// Should contain expected statistical information
	assert.Contains(t, stats, "total_rules")
	assert.Contains(t, stats, "enabled_rules")
	assert.IsType(t, 0, stats["total_rules"])
	assert.IsType(t, 0, stats["enabled_rules"])

	// Values should be reasonable
	assert.Greater(t, stats["total_rules"], 0)
	assert.GreaterOrEqual(t, stats["enabled_rules"], 0)
	assert.LessOrEqual(t, stats["enabled_rules"], stats["total_rules"])
}

func TestRuleEngine_EdgeCases(t *testing.T) {
	ctx := context.Background()
	engine := createTestRuleEngine(t)

	t.Run("nil tokens", func(t *testing.T) {
		result := engine.LintDocument(ctx, nil, []string{"# Title"}, "nil-tokens.md")

		// Should handle gracefully
		require.True(t, result.IsOk())
		violations := result.Unwrap()
		assert.NotNil(t, violations)
	})

	t.Run("nil lines", func(t *testing.T) {
		tokens := createTestTokens("# Title")
		result := engine.LintDocument(ctx, tokens, nil, "nil-lines.md")

		// Should handle gracefully
		require.True(t, result.IsOk())
		violations := result.Unwrap()
		assert.NotNil(t, violations)
	})

	t.Run("empty identifier", func(t *testing.T) {
		tokens := createTestTokens("# Title")
		lines := []string{"# Title"}
		result := engine.LintDocument(ctx, tokens, lines, "")

		// Should handle gracefully
		require.True(t, result.IsOk())
		violations := result.Unwrap()
		assert.NotNil(t, violations)
	})

	t.Run("mismatched tokens and lines", func(t *testing.T) {
		tokens := createTestTokens("# Title\n\nContent")
		lines := []string{"# Title"} // Fewer lines than expected

		result := engine.LintDocument(ctx, tokens, lines, "mismatched.md")

		// Should handle gracefully
		require.True(t, result.IsOk())
		violations := result.Unwrap()
		assert.NotNil(t, violations)
	})
}

func TestRuleEngine_PerformanceWithLargeDocument(t *testing.T) {
	ctx := context.Background()
	engine := createTestRuleEngine(t)

	// Generate large document
	var contentBuilder strings.Builder
	for i := 0; i < 1000; i++ {
		contentBuilder.WriteString(fmt.Sprintf("# Heading %d\n\n", i))
		contentBuilder.WriteString("Paragraph content with some violations.\n\n")
		if i%10 == 0 {
			contentBuilder.WriteString("#Bad heading without space\n\n")
		}
	}

	content := contentBuilder.String()
	tokens := createTestTokens(content)
	lines := strings.Split(content, "\n")

	result := engine.LintDocument(ctx, tokens, lines, "large.md")
	require.True(t, result.IsOk())

	violations := result.Unwrap()
	assert.NotNil(t, violations)

	// Should find violations in large document
	assert.Greater(t, len(violations), 0)
}

// Benchmark tests
func BenchmarkRuleEngine_LintDocument(b *testing.B) {
	ctx := context.Background()
	engine := createTestRuleEngine(b)

	content := `# Title

## Subtitle

This is paragraph content with some **bold** and *italic* text.

- List item 1
- List item 2
- List item 3

` + "```go\nfunc example() {\n    return \"hello\"\n}\n```" + `

> This is a blockquote.

Final paragraph.
`

	tokens := createTestTokens(content)
	lines := strings.Split(content, "\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := engine.LintDocument(ctx, tokens, lines, "benchmark.md")
		require.True(b, result.IsOk())
	}
}

func BenchmarkRuleEngine_ConfigureRules(b *testing.B) {
	engine := createTestRuleEngine(b)

	config := map[string]interface{}{
		"MD001": true,
		"MD002": false,
		"MD013": map[string]interface{}{
			"line_length": 120,
			"tables":      false,
		},
		"MD018": true,
		"MD041": false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := engine.ConfigureRules(config)
		require.NoError(b, err)
	}
}

func BenchmarkRuleEngine_LintLargeDocument(b *testing.B) {
	ctx := context.Background()
	engine := createTestRuleEngine(b)

	// Generate large content
	var contentBuilder strings.Builder
	for i := 0; i < 500; i++ {
		contentBuilder.WriteString(fmt.Sprintf("# Section %d\n\n", i))
		contentBuilder.WriteString("Content paragraph here.\n\n")
		contentBuilder.WriteString("- List item\n")
		contentBuilder.WriteString("- Another item\n\n")
	}

	content := contentBuilder.String()
	tokens := createTestTokens(content)
	lines := strings.Split(content, "\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := engine.LintDocument(ctx, tokens, lines, "large-benchmark.md")
		require.True(b, result.IsOk())
	}
}
