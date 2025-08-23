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

// MD026 - Trailing punctuation in heading
func NewMD026Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md026.md")

	return entity.NewRule(
		[]string{"MD026", "no-trailing-punctuation"},
		"Trailing punctuation in heading",
		[]string{"headings"},
		infoURL,
		"commonmark",
		map[string]interface{}{
			"punctuation": ".,;:!。，；：！", // Punctuation characters to check for
		},
		md026Function,
	)
}

func md026Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Get configuration
	punctuation := getStringConfig(params.Config, "punctuation", ".,;:!。，；：！")

	if punctuation == "" {
		// If punctuation is empty, rule is disabled
		return functional.Ok(violations)
	}

	// Create set of punctuation characters for fast lookup
	punctuationSet := make(map[rune]bool)
	for _, char := range punctuation {
		punctuationSet[char] = true
	}

	// Regexes for different heading types
	atxRegex := regexp.MustCompile(`^(\s*)(#{1,6})(\s+)(.*)$`)
	setextRegex := regexp.MustCompile(`^(=+|-+)\s*$`)

	// HTML entity regex to exclude HTML entity references like &copy;
	htmlEntityRegex := regexp.MustCompile(`&[a-zA-Z0-9]+;$|&#[0-9]+;$|&#x[0-9a-fA-F]+;$`)

	// Process each line
	for i, line := range params.Lines {
		lineNumber := i + 1
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		var headingText string
		var isHeading bool

		// Check for ATX headings
		if matches := atxRegex.FindStringSubmatch(line); matches != nil {
			headingText = strings.TrimSpace(matches[4])
			isHeading = true
		}

		// Check for Setext headings
		if !isHeading && setextRegex.MatchString(line) && i > 0 {
			prevLine := strings.TrimSpace(params.Lines[i-1])
			if prevLine != "" {
				headingText = prevLine
				isHeading = true
				lineNumber = i // Use the text line number (i is 0-based)
			}
		}

		if isHeading && headingText != "" {
			// Check if heading ends with trailing punctuation
			lastChar := rune(headingText[len(headingText)-1])

			if punctuationSet[lastChar] {
				// Check if this is part of an HTML entity reference
				if htmlEntityRegex.MatchString(headingText) {
					continue // Skip HTML entities
				}

				violation := value.NewViolation(
					[]string{"MD026", "no-trailing-punctuation"},
					"Trailing punctuation in heading",
					nil,
					lineNumber,
				)

				violation = violation.WithErrorDetail("Heading ends with punctuation: '" + string(lastChar) + "'")
				violation = violation.WithErrorContext(headingText)

				// Find the position of the trailing punctuation
				headingEndPos := strings.LastIndex(line, headingText) + len(headingText)
				violation = violation.WithColumn(headingEndPos)

				// Add fix information - remove trailing punctuation
				fixedText := strings.TrimRightFunc(headingText, func(r rune) bool {
					return punctuationSet[r]
				})

				// For ATX headings, we need to replace the entire heading
				if strings.Contains(line, "#") {
					// Find and replace the heading text part
					fixedLine := strings.Replace(line, headingText, fixedText, 1)

					fixInfo := value.NewFixInfo().
						WithLineNumber(lineNumber).
						WithEditColumn(1).
						WithDeleteLength(len(line)).
						WithReplaceText(fixedLine)

					violation = violation.WithFixInfo(*fixInfo)
				} else {
					// For setext headings, just fix the text line
					fixInfo := value.NewFixInfo().
						WithLineNumber(lineNumber).
						WithEditColumn(1).
						WithDeleteLength(len(line)).
						WithReplaceText(fixedText)

					violation = violation.WithFixInfo(*fixInfo)
				}

				violations = append(violations, *violation)
			}
		}
	}

	return functional.Ok(violations)
}
