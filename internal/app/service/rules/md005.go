package rules

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/gomdlint/gomdlint/internal/domain/entity"
	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// MD005 - Inconsistent indentation for list items at the same level
func NewMD005Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md005.md")

	return entity.NewRule(
		[]string{"MD005", "list-indent"},
		"Inconsistent indentation for list items at the same level",
		[]string{"bullet", "indentation", "ul"},
		infoURL,
		"commonmark",
		map[string]interface{}{},
		md005Function,
	)
}

func md005Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Regex for list items (both ordered and unordered)
	listRegex := regexp.MustCompile(`^(\s*)([-*+]|\d{1,9}[.)])(\s+)(.*)$`)

	// Track indentation by level (depth)
	levelIndents := make(map[int]int) // level -> expected indent

	// Process each line
	for i, line := range params.Lines {
		lineNumber := i + 1

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check if this is a list item
		matches := listRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		indent := len(matches[1])

		// Determine the list level by counting how many existing levels this indent fits into
		level := 0
		for existingLevel, existingIndent := range levelIndents {
			if indent > existingIndent {
				if existingLevel >= level {
					level = existingLevel + 1
				}
			}
		}

		// Check if we've seen this level before
		if expectedIndent, exists := levelIndents[level]; exists {
			// This level has been seen before - check consistency
			if indent != expectedIndent {
				violation := value.NewViolation(
					[]string{"MD005", "list-indent"},
					"Inconsistent indentation for list items at the same level",
					nil,
					lineNumber,
				)

				detail := fmt.Sprintf("Expected: %d spaces, Actual: %d spaces", expectedIndent, indent)
				violation = violation.WithErrorDetail(detail)
				violation = violation.WithErrorContext(strings.TrimSpace(line))
				violation = violation.WithColumn(1)
				violation = violation.WithLength(indent)

				// Add fix information - adjust indentation
				spaceDiff := expectedIndent - indent
				var fixInfo *value.FixInfo

				if spaceDiff > 0 {
					// Need to add spaces
					fixInfo = value.NewFixInfo().
						WithLineNumber(lineNumber).
						WithEditColumn(1).
						WithDeleteLength(0).
						WithReplaceText(strings.Repeat(" ", spaceDiff))
				} else {
					// Need to remove spaces
					fixInfo = value.NewFixInfo().
						WithLineNumber(lineNumber).
						WithEditColumn(1).
						WithDeleteLength(-spaceDiff).
						WithReplaceText("")
				}

				violation = violation.WithFixInfo(*fixInfo)
				violations = append(violations, *violation)
			}
		} else {
			// New level - record the indentation
			levelIndents[level] = indent
		}
	}

	return functional.Ok(violations)
}
