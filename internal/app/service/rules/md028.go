package rules

import (
	"context"
	"net/url"
	"regexp"
	"strings"

	"github.com/gomdlint/gomdlint/internal/domain/entity"
	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// MD028 - Blank line inside blockquote
func NewMD028Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md028.md")

	return entity.NewRule(
		[]string{"MD028", "no-blanks-blockquote"},
		"Blank line inside blockquote",
		[]string{"blockquote", "whitespace"},
		infoURL,
		"commonmark",
		map[string]interface{}{},
		md028Function,
	)
}

func md028Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Regex for blockquote lines
	blockquoteRegex := regexp.MustCompile(`^(\s*)>\s*(.*)$`)

	// Track blockquote state
	inBlockquote := false
	var blockquoteLines []int

	// Process each line
	for i, line := range params.Lines {
		lineNumber := i + 1
		trimmed := strings.TrimSpace(line)

		// Check if this line is a blockquote
		isBlockquoteLine := blockquoteRegex.MatchString(line)

		if isBlockquoteLine {
			if !inBlockquote {
				// Starting a blockquote
				inBlockquote = true
				blockquoteLines = []int{lineNumber}
			} else {
				// Continuing blockquote
				blockquoteLines = append(blockquoteLines, lineNumber)
			}
		} else if trimmed == "" {
			if inBlockquote {
				// Blank line in blockquote context - check next non-empty line
				nextBlockquoteLine := findNextBlockquoteLine(params.Lines, i)
				if nextBlockquoteLine != -1 {
					// Found another blockquote line after blank - this is the violation
					violation := value.NewViolation(
						[]string{"MD028", "no-blanks-blockquote"},
						"Blank line inside blockquote",
						nil,
						lineNumber,
					)

					violation = violation.WithErrorDetail("Blank line separates blockquote blocks")
					violation = violation.WithErrorContext("")

					violations = append(violations, *violation)
				}
				// Continue in blockquote context for now
			}
		} else {
			// Non-empty, non-blockquote line
			if inBlockquote {
				// End of blockquote
				inBlockquote = false
				blockquoteLines = []int{}
			}
		}
	}

	return functional.Ok(violations)
}

// findNextBlockquoteLine finds the next blockquote line after the given index
func findNextBlockquoteLine(lines []string, startIndex int) int {
	blockquoteRegex := regexp.MustCompile(`^(\s*)>\s*(.*)$`)

	for i := startIndex + 1; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue // Skip blank lines
		}

		if blockquoteRegex.MatchString(line) {
			return i + 1 // Return 1-based line number
		}

		// Found non-blockquote, non-blank line
		break
	}

	return -1 // No blockquote line found
}
