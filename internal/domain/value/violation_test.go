package value

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test scenarios for violation value objects following club/ standards
func TestNewViolation(t *testing.T) {
	ruleNames := []string{"MD001", "heading-increment"}
	description := "Heading levels should only increment by one level at a time"
	lineNumber := 5

	violation := &Violation{
		RuleNames:       ruleNames,
		RuleDescription: description,
		LineNumber:      lineNumber,
	}

	assert.NotNil(t, violation, "Violation should not be nil")
	assert.Equal(t, ruleNames, violation.RuleNames)
	assert.Equal(t, description, violation.RuleDescription)
	assert.Equal(t, lineNumber, violation.LineNumber)
}

func TestViolation_WithMethods(t *testing.T) {
	baseViolation := &Violation{
		RuleNames:       []string{"MD018", "no-missing-space-atx"},
		RuleDescription: "No space after hash on atx style heading",
		LineNumber:      3,
	}

	t.Run("WithErrorDetail", func(t *testing.T) {
		detail := "Missing space after hash"
		violation := baseViolation.WithErrorDetail(detail)

		assert.Equal(t, detail, violation.ErrorDetail.Unwrap())
	})

	t.Run("WithErrorContext", func(t *testing.T) {
		context := "#Heading without space"
		violation := baseViolation.WithErrorContext(context)

		assert.Equal(t, context, violation.ErrorContext.Unwrap())
	})

	t.Run("basic violation structure", func(t *testing.T) {
		violation := baseViolation.
			WithErrorDetail("Missing space").
			WithErrorContext("#Heading")

		assert.Equal(t, "Missing space", violation.ErrorDetail.Unwrap())
		assert.Equal(t, "#Heading", violation.ErrorContext.Unwrap())
		assert.Equal(t, 3, violation.LineNumber)
	})
}

func TestViolation_Scenarios(t *testing.T) {
	scenarios := []struct {
		name        string
		ruleNames   []string
		description string
		lineNumber  int
		expectValid bool
	}{
		{
			name:        "valid basic violation",
			ruleNames:   []string{"MD001"},
			description: "Heading increment",
			lineNumber:  1,
			expectValid: true,
		},
		{
			name:        "violation with multiple rule names",
			ruleNames:   []string{"MD018", "no-missing-space-atx"},
			description: "No space after hash",
			lineNumber:  5,
			expectValid: true,
		},
		{
			name:        "violation with empty description",
			ruleNames:   []string{"MD999"},
			description: "",
			lineNumber:  10,
			expectValid: true, // Should handle gracefully
		},
		{
			name:        "violation with zero line number",
			ruleNames:   []string{"MD001"},
			description: "Test violation",
			lineNumber:  0,
			expectValid: true, // Line 0 might be valid in some contexts
		},
		{
			name:        "violation with negative line number",
			ruleNames:   []string{"MD001"},
			description: "Test violation",
			lineNumber:  -1,
			expectValid: true, // Should handle gracefully
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			violation := NewViolation(scenario.ruleNames, scenario.description, nil, scenario.lineNumber)

			if scenario.expectValid {
				assert.NotNil(t, violation)
				assert.Equal(t, scenario.ruleNames, violation.RuleNames)
				assert.Equal(t, scenario.description, violation.RuleDescription)
				assert.Equal(t, scenario.lineNumber, violation.LineNumber)
			}
		})
	}
}

func TestViolation_EdgeCases(t *testing.T) {
	t.Run("nil rule names", func(t *testing.T) {
		violation := NewViolation(nil, "Test", nil, 1)

		assert.NotNil(t, violation)
		assert.NotNil(t, violation.RuleNames, "Should initialize empty slice for safety")
		assert.Empty(t, violation.RuleNames, "Should be empty when given nil")
	})

	t.Run("empty rule names slice", func(t *testing.T) {
		violation := NewViolation([]string{}, "Test", nil, 1)

		assert.NotNil(t, violation)
		assert.Equal(t, []string{}, violation.RuleNames)
	})

	t.Run("very long rule names", func(t *testing.T) {
		longRuleName := make([]string, 100)
		for i := 0; i < 100; i++ {
			longRuleName[i] = fmt.Sprintf("VERY_LONG_RULE_NAME_%d", i)
		}

		violation := NewViolation(longRuleName, "Test", nil, 1)

		assert.NotNil(t, violation)
		assert.Len(t, violation.RuleNames, 100)
	})

	t.Run("very long description", func(t *testing.T) {
		longDescription := strings.Repeat("Very long description text. ", 1000)

		violation := NewViolation([]string{"MD001"}, longDescription, nil, 1)

		assert.NotNil(t, violation)
		assert.Equal(t, longDescription, violation.RuleDescription)
	})
}

func TestViolation_String(t *testing.T) {
	t.Run("basic string representation", func(t *testing.T) {
		violation := NewViolation(
			[]string{"MD018", "no-missing-space-atx"},
			"No space after hash on atx style heading",
			nil,
			5,
		).WithErrorDetail("Missing space after #")

		str := violation.String()

		assert.Contains(t, str, "MD018")
		assert.Contains(t, str, "No space after hash")
		assert.Contains(t, str, "5")
		assert.Contains(t, str, "Missing space after #")
	})

	t.Run("minimal violation string", func(t *testing.T) {
		violation := NewViolation(
			[]string{"MD001"},
			"Heading increment",
			nil,
			1,
		)

		str := violation.String()

		assert.Contains(t, str, "MD001")
		assert.Contains(t, str, "Heading increment")
		assert.Contains(t, str, "1")
	})

	t.Run("violation with context", func(t *testing.T) {
		violation := NewViolation(
			[]string{"MD018"},
			"No space after hash",
			nil,
			3,
		).WithErrorContext("#Heading")

		str := violation.String()

		assert.Contains(t, str, "#Heading")
	})
}

func TestViolation_Equality(t *testing.T) {
	violation1 := NewViolation(
		[]string{"MD018"},
		"No space after hash",
		nil,
		5,
	).WithErrorDetail("Missing space")

	violation2 := NewViolation(
		[]string{"MD018"},
		"No space after hash",
		nil,
		5,
	).WithErrorDetail("Missing space")

	violation3 := NewViolation(
		[]string{"MD019"},
		"Different rule",
		nil,
		5,
	)

	// Test equality (this would require implementing comparison methods)
	assert.Equal(t, violation1.RuleNames, violation2.RuleNames)
	assert.Equal(t, violation1.RuleDescription, violation2.RuleDescription)
	assert.Equal(t, violation1.LineNumber, violation2.LineNumber)

	assert.NotEqual(t, violation1.RuleNames, violation3.RuleNames)
	assert.NotEqual(t, violation1.RuleDescription, violation3.RuleDescription)
}

func TestViolation_ComplexScenarios(t *testing.T) {
	t.Run("violation with all optional fields", func(t *testing.T) {
		fixInfo := NewFixInfo().
			WithInsertText("# ").
			WithEditColumn(1)

		violation := NewViolation(
			[]string{"MD018", "no-missing-space-atx"},
			"No space after hash on atx style heading",
			nil,
			5,
		).WithErrorDetail("Missing space after '#' in ATX heading").
			WithErrorContext("#Heading without space").
			WithErrorRange(*NewRange(NewPosition(5, 1), NewPosition(5, 2))).
			WithFixInfo(*fixInfo)

		// Verify all fields are set
		assert.True(t, violation.ErrorDetail.IsSome())
		assert.True(t, violation.ErrorContext.IsSome())
		assert.True(t, violation.ErrorRange.IsSome())
		assert.True(t, violation.FixInfo.IsSome())

		// Verify values
		assert.Equal(t, "Missing space after '#' in ATX heading", violation.ErrorDetail.Unwrap())
		assert.Equal(t, "#Heading without space", violation.ErrorContext.Unwrap())

		retrievedFixInfo := violation.FixInfo.Unwrap()
		assert.Equal(t, "# ", retrievedFixInfo.InsertText.Unwrap())
		assert.Equal(t, 1, retrievedFixInfo.EditColumn.Unwrap())
	})

	t.Run("violation with Unicode content", func(t *testing.T) {
		violation := NewViolation(
			[]string{"MD018"},
			"Unicode: 中文标题没有空格",
			nil,
			3,
		).WithErrorContext("#中文标题").
			WithErrorDetail("缺少空格")

		assert.Contains(t, violation.RuleDescription, "中文")
		assert.Equal(t, "缺少空格", violation.ErrorDetail.Unwrap())
		assert.Equal(t, "#中文标题", violation.ErrorContext.Unwrap())
	})
}

// Benchmark tests
func BenchmarkNewViolation(b *testing.B) {
	ruleNames := []string{"MD018", "no-missing-space-atx"}
	description := "No space after hash on atx style heading"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		violation := NewViolation(ruleNames, description, nil, 5)
		_ = violation
	}
}

func BenchmarkViolation_WithMethods(b *testing.B) {
	baseViolation := NewViolation(
		[]string{"MD018"},
		"No space after hash",
		nil,
		5,
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		violation := baseViolation.
			WithErrorDetail("Missing space").
			WithErrorContext("#Heading").
			WithErrorRange(*NewRange(NewPosition(5, 1), NewPosition(5, 2)))
		_ = violation
	}
}

func BenchmarkViolation_String(b *testing.B) {
	violation := NewViolation(
		[]string{"MD018", "no-missing-space-atx"},
		"No space after hash on atx style heading",
		nil,
		5,
	).WithErrorDetail("Missing space after '#' in ATX heading").
		WithErrorContext("#Heading without space")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		str := violation.String()
		_ = str
	}
}
