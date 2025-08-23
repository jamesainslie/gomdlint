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

// MD023 - Headings must start at the beginning of the line
func NewMD023Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md023.md")

	return entity.NewRule(
		[]string{"MD023", "heading-start-left"},
		"Headings must start at the beginning of the line",
		[]string{"headings", "spaces"},
		infoURL,
		"commonmark",
		map[string]interface{}{},
		md023Function,
	)
}

func md023Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Regex for ATX headings with leading spaces
	indentedATXRegex := regexp.MustCompile(`^(\s+)(#{1,6})(\s+)(.*)$`)

	// Regex for setext underlines with leading spaces
	indentedSetextRegex := regexp.MustCompile(`^(\s+)(=+|-+)\s*$`)

	// Process each line
	for i, line := range params.Lines {
		lineNumber := i + 1

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check for indented ATX headings
		if matches := indentedATXRegex.FindStringSubmatch(line); matches != nil {
			indent := matches[1]

			// Exception: headings in blockquotes are allowed to be indented
			if !isInBlockquote(line) {
				violation := value.NewViolation(
					[]string{"MD023", "heading-start-left"},
					"Headings must start at the beginning of the line",
					nil,
					lineNumber,
				)

				violation = violation.WithErrorDetail("Heading is indented")
				violation = violation.WithErrorContext(strings.TrimSpace(line))
				violation = violation.WithColumn(1)
				violation = violation.WithLength(len(indent))

				// Add fix information - remove indentation
				fixInfo := value.NewFixInfo().
					WithLineNumber(lineNumber).
					WithEditColumn(1).
					WithDeleteLength(len(indent)).
					WithReplaceText("")

				violation = violation.WithFixInfo(*fixInfo)
				violations = append(violations, *violation)
			}
		}

		// Check for indented setext headings
		if matches := indentedSetextRegex.FindStringSubmatch(line); matches != nil && i > 0 {
			prevLine := strings.TrimSpace(params.Lines[i-1])
			if prevLine != "" {
				indent := matches[1]

				// Exception: headings in blockquotes are allowed to be indented
				if !isInBlockquote(line) {
					violation := value.NewViolation(
						[]string{"MD023", "heading-start-left"},
						"Headings must start at the beginning of the line",
						nil,
						i, // Line number of the text (0-based i is the underline)
					)

					violation = violation.WithErrorDetail("Setext heading underline is indented")
					violation = violation.WithErrorContext(strings.TrimSpace(line))
					violation = violation.WithColumn(1)
					violation = violation.WithLength(len(indent))

					// Add fix information - remove indentation from underline
					fixInfo := value.NewFixInfo().
						WithLineNumber(lineNumber).
						WithEditColumn(1).
						WithDeleteLength(len(indent)).
						WithReplaceText("")

					violation = violation.WithFixInfo(*fixInfo)
					violations = append(violations, *violation)
				}
			}
		}
	}

	return functional.Ok(violations)
}

// isInBlockquote checks if a line is inside a blockquote context
func isInBlockquote(line string) bool {
	// Simple check: if the line starts with > after optional whitespace
	// More sophisticated parsing would track blockquote context across lines
	blockquoteRegex := regexp.MustCompile(`^\s*>\s*#`)
	return blockquoteRegex.MatchString(line)
}
