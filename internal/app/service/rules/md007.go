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

// MD007 - Unordered list indentation
func NewMD007Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md007.md")

	return entity.NewRule(
		[]string{"MD007", "ul-indent"},
		"Unordered list indentation",
		[]string{"bullet", "indentation", "ul"},
		infoURL,
		"commonmark",
		map[string]interface{}{
			"indent":         2,     // Spaces for indent
			"start_indented": false, // Whether to indent the first level
			"start_indent":   2,     // Spaces for first level indent (when start_indented is true)
		},
		md007Function,
	)
}

func md007Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Get configuration
	indent := getIntConfig(params.Config, "indent", 2)
	startIndented := getBoolConfig(params.Config, "start_indented", false)
	startIndent := getIntConfig(params.Config, "start_indent", 2)

	// Regex for unordered list items only
	ulRegex := regexp.MustCompile(`^(\s*)([-*+])(\s+)(.*)$`)

	// Track nesting levels and their expected indentation
	var listStack []int // Stack of indentation levels

	// Process each line
	for i, line := range params.Lines {
		lineNumber := i + 1

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check if this is an unordered list item
		matches := ulRegex.FindStringSubmatch(line)
		if matches == nil {
			// Not a list item - reset stack
			listStack = []int{}
			continue
		}

		actualIndent := len(matches[1])

		// Determine expected indentation based on nesting level
		var expectedIndent int
		level := 0

		// Find the appropriate nesting level
		for len(listStack) > 0 && actualIndent <= listStack[len(listStack)-1] {
			// Pop levels that are at the same or greater indentation
			listStack = listStack[:len(listStack)-1]
		}

		level = len(listStack)

		// Calculate expected indentation
		if level == 0 {
			// First level
			if startIndented {
				expectedIndent = startIndent
			} else {
				expectedIndent = 0
			}
		} else {
			// Nested level
			if startIndented {
				expectedIndent = startIndent + (level * indent)
			} else {
				expectedIndent = level * indent
			}
		}

		// Check if indentation is correct
		if actualIndent != expectedIndent {
			// Check if this item is part of an ordered list context (skip if so)
			if !isInOrderedListContext(params.Lines, lineNumber) {
				violation := value.NewViolation(
					[]string{"MD007", "ul-indent"},
					"Unordered list indentation",
					nil,
					lineNumber,
				)

				detail := fmt.Sprintf("Expected: %d spaces, Actual: %d spaces", expectedIndent, actualIndent)
				violation = violation.WithErrorDetail(detail)
				violation = violation.WithErrorContext(strings.TrimSpace(line))
				violation = violation.WithColumn(1)
				violation = violation.WithLength(actualIndent)

				// Add fix information
				spaceDiff := expectedIndent - actualIndent
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
		}

		// Add this level to the stack
		listStack = append(listStack, actualIndent)
	}

	return functional.Ok(violations)
}

// isInOrderedListContext checks if the unordered list is nested within an ordered list
func isInOrderedListContext(lines []string, currentLine int) bool {
	olRegex := regexp.MustCompile(`^(\s*)(\d{1,9}[.)])(\s+)(.*)$`)
	ulRegex := regexp.MustCompile(`^(\s*)([-*+])(\s+)(.*)$`)

	// Get current item's indentation
	currentMatches := ulRegex.FindStringSubmatch(lines[currentLine-1])
	if currentMatches == nil {
		return false
	}
	currentIndent := len(currentMatches[1])

	// Look backwards for ordered list items at a lower indentation level
	for i := currentLine - 2; i >= 0; i-- {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check for ordered list item
		if matches := olRegex.FindStringSubmatch(line); matches != nil {
			olIndent := len(matches[1])
			if olIndent < currentIndent {
				return true // Found parent ordered list
			}
		}

		// If we hit a line with equal or less indentation that's not a list item, stop
		if len(strings.TrimLeft(line, " \t")) > 0 && len(line)-len(strings.TrimLeft(line, " \t")) <= currentIndent {
			if !ulRegex.MatchString(line) && !olRegex.MatchString(line) {
				break
			}
		}
	}

	return false
}
