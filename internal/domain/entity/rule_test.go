package entity

import (
	"context"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// Mock rule function for testing
func mockRuleFunction(ctx context.Context, params RuleParams) functional.Result[[]value.Violation] {
	violations := []value.Violation{}
	if len(params.Lines) > 0 && params.Lines[0] == "TRIGGER_VIOLATION" {
		violation := value.NewViolation(
			[]string{"MOCK", "mock-rule"},
			"Mock violation",
			nil,
			1,
		)
		violations = append(violations, *violation)
	}
	return functional.Ok(violations)
}

func mockErrorRuleFunction(ctx context.Context, params RuleParams) functional.Result[[]value.Violation] {
	return functional.Err[[]value.Violation](assert.AnError)
}

func TestNewRule_ValidInput(t *testing.T) {
	infoURL, _ := url.Parse("https://example.com/rule")

	result := NewRule(
		[]string{"TEST001", "test-rule"},
		"Test rule description",
		[]string{"test", "example"},
		infoURL,
		"commonmark",
		map[string]interface{}{"enabled": true},
		mockRuleFunction,
	)

	require.True(t, result.IsOk())
	rule := result.Unwrap()

	assert.Equal(t, []string{"TEST001", "test-rule"}, rule.Names())
	assert.Equal(t, "Test rule description", rule.Description())
	assert.Equal(t, []string{"test", "example"}, rule.Tags())
	assert.Equal(t, infoURL, rule.Information())
	assert.Equal(t, "commonmark", rule.Parser())
	assert.Equal(t, map[string]interface{}{"enabled": true}, rule.Config())
}

func TestNewRule_EmptyNames(t *testing.T) {
	infoURL, _ := url.Parse("https://example.com/rule")

	result := NewRule(
		[]string{},
		"Test rule description",
		[]string{"test"},
		infoURL,
		"commonmark",
		map[string]interface{}{},
		mockRuleFunction,
	)

	require.True(t, result.IsErr())
	assert.Contains(t, result.Error().Error(), "must have at least one name")
}

func TestNewRule_EmptyDescription(t *testing.T) {
	infoURL, _ := url.Parse("https://example.com/rule")

	result := NewRule(
		[]string{"TEST001"},
		"",
		[]string{"test"},
		infoURL,
		"commonmark",
		map[string]interface{}{},
		mockRuleFunction,
	)

	require.True(t, result.IsErr())
	assert.Contains(t, result.Error().Error(), "must have a description")
}

func TestNewRule_NilFunction(t *testing.T) {
	infoURL, _ := url.Parse("https://example.com/rule")

	result := NewRule(
		[]string{"TEST001"},
		"Test rule description",
		[]string{"test"},
		infoURL,
		"commonmark",
		map[string]interface{}{},
		nil,
	)

	require.True(t, result.IsErr())
	assert.Contains(t, result.Error().Error(), "must have a function")
}

func TestNewRule_EmptyParser(t *testing.T) {
	infoURL, _ := url.Parse("https://example.com/rule")

	result := NewRule(
		[]string{"TEST001"},
		"Test rule description",
		[]string{"test"},
		infoURL,
		"", // Empty parser should be allowed
		map[string]interface{}{},
		mockRuleFunction,
	)

	require.True(t, result.IsOk())
	rule := result.Unwrap()
	assert.Equal(t, "", rule.Parser())
}

func TestNewRule_InvalidNames(t *testing.T) {
	infoURL, _ := url.Parse("https://example.com/rule")

	result := NewRule(
		[]string{"", "valid-name"},
		"Test rule description",
		[]string{"test"},
		infoURL,
		"commonmark",
		map[string]interface{}{},
		mockRuleFunction,
	)

	require.True(t, result.IsErr())
	assert.Contains(t, result.Error().Error(), "names cannot be empty")
}

func TestNewRule_InvalidTags(t *testing.T) {
	infoURL, _ := url.Parse("https://example.com/rule")

	result := NewRule(
		[]string{"TEST001"},
		"Test rule description",
		[]string{"valid-tag", ""}, // Empty tags should be allowed
		infoURL,
		"commonmark",
		map[string]interface{}{},
		mockRuleFunction,
	)

	require.True(t, result.IsOk()) // Should succeed
	rule := result.Unwrap()
	assert.Equal(t, []string{"valid-tag", ""}, rule.Tags())
}

func TestNewRule_NilConfig(t *testing.T) {
	infoURL, _ := url.Parse("https://example.com/rule")

	result := NewRule(
		[]string{"TEST001"},
		"Test rule description",
		[]string{"test"},
		infoURL,
		"commonmark",
		nil, // nil config should be allowed
		mockRuleFunction,
	)

	require.True(t, result.IsOk())
	rule := result.Unwrap()
	assert.NotNil(t, rule.Config())
	assert.Empty(t, rule.Config())
}

func TestNewRule_NilInformation(t *testing.T) {
	result := NewRule(
		[]string{"TEST001"},
		"Test rule description",
		[]string{"test"},
		nil, // nil information should be allowed
		"commonmark",
		map[string]interface{}{},
		mockRuleFunction,
	)

	require.True(t, result.IsOk())
	rule := result.Unwrap()
	assert.Nil(t, rule.Information())
}

func TestRule_Execute(t *testing.T) {
	infoURL, _ := url.Parse("https://example.com/rule")

	rule := NewRule(
		[]string{"TEST001"},
		"Test rule",
		[]string{"test"},
		infoURL,
		"commonmark",
		map[string]interface{}{},
		mockRuleFunction,
	).Unwrap()

	// Test with normal input
	params := RuleParams{
		Lines:    []string{"Normal line"},
		Config:   map[string]interface{}{},
		Filename: "test.md",
		Tokens:   []value.Token{},
	}

	result := rule.Execute(context.Background(), params)
	require.True(t, result.IsOk())
	violations := result.Unwrap()
	assert.Empty(t, violations)

	// Test with trigger input
	params.Lines[0] = "TRIGGER_VIOLATION"
	result = rule.Execute(context.Background(), params)
	require.True(t, result.IsOk())
	violations = result.Unwrap()
	assert.Len(t, violations, 1)
	assert.Contains(t, violations[0].RuleNames, "MOCK")
}

func TestRule_ExecuteWithError(t *testing.T) {
	rule := NewRule(
		[]string{"ERROR_TEST"},
		"Error test rule",
		[]string{"test"},
		nil,
		"commonmark",
		map[string]interface{}{},
		mockErrorRuleFunction,
	).Unwrap()

	params := RuleParams{
		Lines:    []string{"test"},
		Config:   map[string]interface{}{},
		Filename: "test.md",
		Tokens:   []value.Token{},
	}

	result := rule.Execute(context.Background(), params)
	require.True(t, result.IsErr())
	assert.Equal(t, assert.AnError, result.Error())
}

func TestRule_MatchesName(t *testing.T) {
	rule := NewRule(
		[]string{"MD001", "heading-increment", "h1-increment"},
		"Test rule",
		[]string{"test"},
		nil,
		"commonmark",
		map[string]interface{}{},
		mockRuleFunction,
	).Unwrap()

	// Test primary name
	assert.True(t, rule.HasName("MD001"))

	// Test aliases
	assert.True(t, rule.HasName("heading-increment"))
	assert.True(t, rule.HasName("h1-increment"))

	// Test case sensitivity - the HasName method should handle case insensitive matching
	assert.True(t, rule.HasName("md001"))
	assert.True(t, rule.HasName("HEADING-INCREMENT"))

	// Test non-matching name
	assert.False(t, rule.HasName("MD002"))
	assert.False(t, rule.HasName("other-rule"))
}

func TestRule_HasTag(t *testing.T) {
	rule := NewRule(
		[]string{"TEST001"},
		"Test rule",
		[]string{"headings", "formatting", "style"},
		nil,
		"commonmark",
		map[string]interface{}{},
		mockRuleFunction,
	).Unwrap()

	// Test existing tags
	assert.True(t, rule.HasTag("headings"))
	assert.True(t, rule.HasTag("formatting"))
	assert.True(t, rule.HasTag("style"))

	// Test case sensitivity - HasTag is case insensitive
	assert.True(t, rule.HasTag("Headings"))
	assert.True(t, rule.HasTag("FORMATTING"))

	// Test non-existing tag
	assert.False(t, rule.HasTag("lists"))
	assert.False(t, rule.HasTag("code"))
}

func TestRule_String(t *testing.T) {
	rule := NewRule(
		[]string{"MD001", "heading-increment"},
		"Heading levels should only increment by one level at a time",
		[]string{"headings"},
		nil,
		"commonmark",
		map[string]interface{}{},
		mockRuleFunction,
	).Unwrap()

	str := rule.String()
	// The String method just returns the primary name
	assert.Equal(t, "MD001", str)
}

func TestRule_PrimaryName(t *testing.T) {
	rule := NewRule(
		[]string{"MD001", "heading-increment", "h1-increment"},
		"Test rule",
		[]string{"test"},
		nil,
		"commonmark",
		map[string]interface{}{},
		mockRuleFunction,
	).Unwrap()

	assert.Equal(t, "MD001", rule.PrimaryName())
}

func TestRule_PrimaryName_EmptyNames(t *testing.T) {
	// This should not be possible to create via NewRule, but test defensive programming
	rule := &Rule{
		names: []string{},
	}

	assert.Equal(t, "", rule.PrimaryName())
}

func TestRuleParams_WithFrontMatter(t *testing.T) {
	frontMatter := map[string]interface{}{
		"title":  "Test Document",
		"author": "Test Author",
	}

	params := RuleParams{
		Lines:       []string{"# Test"},
		Config:      map[string]interface{}{},
		Filename:    "test.md",
		Tokens:      []value.Token{},
		FrontMatter: functional.Some(frontMatter),
	}

	assert.True(t, params.FrontMatter.IsSome())
	fm := params.FrontMatter.Unwrap()
	assert.Equal(t, "Test Document", fm["title"])
	assert.Equal(t, "Test Author", fm["author"])
}

func TestRuleParams_WithoutFrontMatter(t *testing.T) {
	params := RuleParams{
		Lines:       []string{"# Test"},
		Config:      map[string]interface{}{},
		Filename:    "test.md",
		Tokens:      []value.Token{},
		FrontMatter: functional.None[map[string]interface{}](),
	}

	assert.True(t, params.FrontMatter.IsNone())
}

// Test rule creation with various edge cases
func TestRule_EdgeCases(t *testing.T) {
	t.Run("very_long_description", func(t *testing.T) {
		longDesc := "This is a very long description that goes on and on and on to test how the rule handles extremely long descriptions that might be used in some edge cases where developers write verbose descriptions for their rules."

		result := NewRule(
			[]string{"LONG001"},
			longDesc,
			[]string{"test"},
			nil,
			"commonmark",
			map[string]interface{}{},
			mockRuleFunction,
		)

		require.True(t, result.IsOk())
		rule := result.Unwrap()
		assert.Equal(t, longDesc, rule.Description())
	})

	t.Run("many_names", func(t *testing.T) {
		manyNames := []string{
			"MD001", "heading-increment", "h1-increment",
			"heading-level-increment", "heading-levels",
			"atx-heading-increment", "setext-heading-increment",
		}

		result := NewRule(
			manyNames,
			"Test rule with many names",
			[]string{"test"},
			nil,
			"commonmark",
			map[string]interface{}{},
			mockRuleFunction,
		)

		require.True(t, result.IsOk())
		rule := result.Unwrap()
		assert.Equal(t, manyNames, rule.Names())

		// Test all names match
		for _, name := range manyNames {
			assert.True(t, rule.HasName(name))
		}
	})

	t.Run("many_tags", func(t *testing.T) {
		manyTags := []string{
			"headings", "formatting", "style", "atx", "setext",
			"structure", "organization", "accessibility", "seo",
		}

		result := NewRule(
			[]string{"TAGS001"},
			"Test rule with many tags",
			manyTags,
			nil,
			"commonmark",
			map[string]interface{}{},
			mockRuleFunction,
		)

		require.True(t, result.IsOk())
		rule := result.Unwrap()
		assert.Equal(t, manyTags, rule.Tags())

		// Test all tags match
		for _, tag := range manyTags {
			assert.True(t, rule.HasTag(tag))
		}
	})

	t.Run("complex_config", func(t *testing.T) {
		complexConfig := map[string]interface{}{
			"enabled":     true,
			"line_length": 80,
			"exceptions":  []string{"code", "tables"},
			"nested": map[string]interface{}{
				"strict_mode":     false,
				"ignore_patterns": []string{"^<!--", "^```"},
			},
		}

		result := NewRule(
			[]string{"COMPLEX001"},
			"Test rule with complex config",
			[]string{"test"},
			nil,
			"commonmark",
			complexConfig,
			mockRuleFunction,
		)

		require.True(t, result.IsOk())
		rule := result.Unwrap()

		config := rule.Config()
		assert.Equal(t, true, config["enabled"])
		assert.Equal(t, 80, config["line_length"])
		assert.Contains(t, config["exceptions"], "code")

		nested := config["nested"].(map[string]interface{})
		assert.Equal(t, false, nested["strict_mode"])
	})
}

// Benchmark tests for rule creation and execution
func BenchmarkNewRule(b *testing.B) {
	infoURL, _ := url.Parse("https://example.com/rule")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewRule(
			[]string{"BENCH001", "bench-rule"},
			"Benchmark test rule",
			[]string{"benchmark", "performance"},
			infoURL,
			"commonmark",
			map[string]interface{}{"enabled": true},
			mockRuleFunction,
		)
	}
}

func BenchmarkRule_Execute(b *testing.B) {
	rule := NewRule(
		[]string{"BENCH001"},
		"Benchmark test rule",
		[]string{"benchmark"},
		nil,
		"commonmark",
		map[string]interface{}{},
		mockRuleFunction,
	).Unwrap()

	params := RuleParams{
		Lines:    []string{"Test line 1", "Test line 2", "Test line 3"},
		Config:   map[string]interface{}{},
		Filename: "benchmark.md",
		Tokens:   []value.Token{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rule.Execute(context.Background(), params)
	}
}

func BenchmarkRule_MatchesName(b *testing.B) {
	rule := NewRule(
		[]string{"MD001", "heading-increment", "h1-increment"},
		"Benchmark test rule",
		[]string{"benchmark"},
		nil,
		"commonmark",
		map[string]interface{}{},
		mockRuleFunction,
	).Unwrap()

	names := []string{"MD001", "heading-increment", "h1-increment", "other-rule"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name := names[i%len(names)]
		rule.HasName(name)
	}
}
