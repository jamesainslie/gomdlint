package rules

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/gomdlint/gomdlint/internal/domain/entity"
	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// MD029 - Ordered list item prefix
func NewMD029Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md029.md")

	return entity.NewRule(
		[]string{"MD029", "ol-prefix"},
		"Ordered list item prefix",
		[]string{"ol"},
		infoURL,
		"commonmark",
		map[string]interface{}{
			"style": "one_or_ordered", // one|ordered|zero|one_or_ordered
		},
		md029Function,
	)
}

func md029Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Get configuration
	styleConfig := getStringConfig(params.Config, "style", "one_or_ordered")

	// Regex for ordered list items
	olRegex := regexp.MustCompile(`^(\s*)(\d{1,9})([.)])(\s+)(.*)$`)

	// Track list contexts
	var listStack []listContext

	// Process each line
	for i, line := range params.Lines {
		lineNumber := i + 1

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check if this is an ordered list item
		matches := olRegex.FindStringSubmatch(line)
		if matches == nil {
			// Not a list item - reset list context if line is not indented or is a different block element
			if !strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "\t") {
				listStack = []listContext{}
			}
			continue
		}

		indent := len(matches[1])
		numberStr := matches[2]
		marker := matches[3]
		content := matches[5]

		number, _ := strconv.Atoi(numberStr)

		// Find or create appropriate list context
		contextIndex := findListContext(listStack, indent)
		if contextIndex == -1 {
			// New list level
			expectedStart := getExpectedStart(styleConfig, number)
			listStack = append(listStack, listContext{
				indent:        indent,
				expectedNext:  expectedStart + 1,
				actualStart:   number,
				expectedStart: expectedStart,
				style:         styleConfig,
			})
		} else {
			// Update existing context
			context := &listStack[contextIndex]

			// Check if number is correct
			expectedNumber := context.expectedNext

			// Handle different styles
			switch styleConfig {
			case "one":
				expectedNumber = 1
			case "zero":
				expectedNumber = 0
			case "ordered":
				expectedNumber = context.expectedNext
			case "one_or_ordered":
				// Accept either "one" style or "ordered" style
				if context.actualStart == 1 {
					// Following "ordered" pattern from 1
					expectedNumber = context.expectedNext
				} else {
					// Following "one" pattern or zero-based
					if context.actualStart == 0 {
						expectedNumber = 0
					} else {
						expectedNumber = 1
					}
				}
			}

			if number != expectedNumber {
				violation := value.NewViolation(
					[]string{"MD029", "ol-prefix"},
					"Ordered list item prefix",
					nil,
					lineNumber,
				)

				detail := fmt.Sprintf("Expected: %d, Actual: %d", expectedNumber, number)
				violation = violation.WithErrorDetail(detail)
				violation = violation.WithErrorContext(strings.TrimSpace(line))

				// Add fix information
				fixedNumberStr := fmt.Sprintf("%d", expectedNumber)
				fixedLine := matches[1] + fixedNumberStr + marker + matches[4] + content

				fixInfo := value.NewFixInfo().
					WithLineNumber(lineNumber).
					WithEditColumn(1).
					WithDeleteLength(len(line)).
					WithReplaceText(fixedLine)

				violation = violation.WithFixInfo(*fixInfo)
				violations = append(violations, *violation)
			}

			// Update expected next number
			if styleConfig == "ordered" || (styleConfig == "one_or_ordered" && context.actualStart == 1) {
				context.expectedNext++
			}
		}

		// Remove deeper list levels
		for len(listStack) > 0 && listStack[len(listStack)-1].indent > indent {
			listStack = listStack[:len(listStack)-1]
		}
	}

	return functional.Ok(violations)
}

type listContext struct {
	indent        int
	expectedNext  int
	actualStart   int
	expectedStart int
	style         string
}

// findListContext finds the appropriate list context for the given indentation
func findListContext(stack []listContext, indent int) int {
	for i := len(stack) - 1; i >= 0; i-- {
		if stack[i].indent == indent {
			return i
		}
	}
	return -1
}

// getExpectedStart returns the expected starting number based on style
func getExpectedStart(style string, actualFirst int) int {
	switch style {
	case "one":
		return 1
	case "zero":
		return 0
	case "ordered":
		return actualFirst // Start with whatever the first item uses
	case "one_or_ordered":
		return actualFirst // Start with whatever the first item uses
	default:
		return 1
	}
}
