package rules

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/gomdlint/gomdlint/internal/domain/entity"
	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// MD012 - Multiple consecutive blank lines
func NewMD012Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md012.md")

	return entity.NewRule(
		[]string{"MD012", "no-multiple-blanks"},
		"Multiple consecutive blank lines",
		[]string{"blank_lines", "whitespace"},
		infoURL,
		"commonmark",
		map[string]interface{}{
			"maximum": 1, // Maximum number of consecutive blank lines
		},
		md012Function,
	)
}

func md012Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Get configuration
	maximum := getIntConfig(params.Config, "maximum", 1)

	consecutiveBlankLines := 0
	var blankLineStart int
	inCodeBlock := false

	// Process each line
	for i, line := range params.Lines {
		lineNumber := i + 1

		// Check if we're in a code block (skip consecutive blank line checks in code blocks)
		if !inCodeBlock && (strings.HasPrefix(strings.TrimSpace(line), "```") || strings.HasPrefix(strings.TrimSpace(line), "~~~")) {
			inCodeBlock = true
			consecutiveBlankLines = 0 // Reset counter when entering code block
			continue
		} else if inCodeBlock && (strings.HasPrefix(strings.TrimSpace(line), "```") || strings.HasPrefix(strings.TrimSpace(line), "~~~")) {
			inCodeBlock = false
			consecutiveBlankLines = 0 // Reset counter when exiting code block
			continue
		}

		// Skip blank line checks inside code blocks
		if inCodeBlock {
			continue
		}

		// Check if line is blank
		isBlank := strings.TrimSpace(line) == ""

		if isBlank {
			if consecutiveBlankLines == 0 {
				blankLineStart = lineNumber
			}
			consecutiveBlankLines++
		} else {
			// Check if we exceeded the maximum
			if consecutiveBlankLines > maximum {
				violation := value.NewViolation(
					[]string{"MD012", "no-multiple-blanks"},
					"Multiple consecutive blank lines",
					nil,
					blankLineStart+maximum, // Report on the first excess blank line
				)

				excessLines := consecutiveBlankLines - maximum
				detail := fmt.Sprintf("Expected: %d, Actual: %d", maximum, consecutiveBlankLines)
				violation = violation.WithErrorDetail(detail)

				// Add fix information - remove excess blank lines
				fixInfo := value.NewFixInfo().
					WithLineNumber(blankLineStart + maximum). // Start at first excess line
					WithDeleteCount(excessLines)              // Delete excess lines

				violation = violation.WithFixInfo(*fixInfo)

				violations = append(violations, *violation)
			}
			consecutiveBlankLines = 0
		}
	}

	// Check if file ends with too many blank lines
	if consecutiveBlankLines > maximum {
		violation := value.NewViolation(
			[]string{"MD012", "no-multiple-blanks"},
			"Multiple consecutive blank lines",
			nil,
			blankLineStart+maximum, // Report on the first excess blank line
		)

		excessLines := consecutiveBlankLines - maximum
		detail := fmt.Sprintf("Expected: %d, Actual: %d", maximum, consecutiveBlankLines)
		violation = violation.WithErrorDetail(detail)

		// Add fix information - remove excess blank lines at end
		fixInfo := value.NewFixInfo().
			WithLineNumber(blankLineStart + maximum).
			WithDeleteCount(excessLines)

		violation = violation.WithFixInfo(*fixInfo)

		violations = append(violations, *violation)
	}

	return functional.Ok(violations)
}
