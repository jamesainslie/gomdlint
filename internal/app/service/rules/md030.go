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

// MD030 - Spaces after list markers
func NewMD030Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md030.md")

	return entity.NewRule(
		[]string{"MD030", "list-marker-space"},
		"Spaces after list markers",
		[]string{"ol", "ul", "whitespace"},
		infoURL,
		"commonmark",
		map[string]interface{}{
			"ul_single": 1, // Spaces for single-line unordered list items
			"ol_single": 1, // Spaces for single-line ordered list items
			"ul_multi":  1, // Spaces for multi-line unordered list items
			"ol_multi":  1, // Spaces for multi-line ordered list items
		},
		md030Function,
	)
}

func md030Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Get configuration
	ulSingle := getIntConfig(params.Config, "ul_single", 1)
	olSingle := getIntConfig(params.Config, "ol_single", 1)
	ulMulti := getIntConfig(params.Config, "ul_multi", 1)
	olMulti := getIntConfig(params.Config, "ol_multi", 1)

	// Regex for list items
	ulRegex := regexp.MustCompile(`^(\s*)([-*+])(\s+)(.*)$`)
	olRegex := regexp.MustCompile(`^(\s*)(\d{1,9}[.)])(\s+)(.*)$`)

	// Process each line
	for i, line := range params.Lines {
		lineNumber := i + 1

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check unordered list items
		if matches := ulRegex.FindStringSubmatch(line); matches != nil {
			indent := matches[1]
			marker := matches[2]
			spaces := matches[3]
			content := matches[4]

			// Determine if this is a multi-line item
			isMultiLine := isMultiLineListItem(params.Lines, i)
			expectedSpaces := ulSingle
			if isMultiLine {
				expectedSpaces = ulMulti
			}

			actualSpaces := len(spaces)
			if actualSpaces != expectedSpaces {
				violation := value.NewViolation(
					[]string{"MD030", "list-marker-space"},
					"Spaces after list markers",
					nil,
					lineNumber,
				)

				itemType := "single-line"
				if isMultiLine {
					itemType = "multi-line"
				}

				detail := fmt.Sprintf("Expected: %d spaces after unordered list marker (%s), Actual: %d spaces", expectedSpaces, itemType, actualSpaces)
				violation = violation.WithErrorDetail(detail)
				violation = violation.WithErrorContext(strings.TrimSpace(line))
				violation = violation.WithColumn(len(indent) + len(marker) + 1)
				violation = violation.WithLength(actualSpaces)

				// Add fix information
				fixedSpaces := strings.Repeat(" ", expectedSpaces)
				fixedLine := indent + marker + fixedSpaces + content

				fixInfo := value.NewFixInfo().
					WithLineNumber(lineNumber).
					WithEditColumn(1).
					WithDeleteLength(len(line)).
					WithReplaceText(fixedLine)

				violation = violation.WithFixInfo(*fixInfo)
				violations = append(violations, *violation)
			}
		}

		// Check ordered list items
		if matches := olRegex.FindStringSubmatch(line); matches != nil {
			indent := matches[1]
			marker := matches[2]
			spaces := matches[3]
			content := matches[4]

			// Determine if this is a multi-line item
			isMultiLine := isMultiLineListItem(params.Lines, i)
			expectedSpaces := olSingle
			if isMultiLine {
				expectedSpaces = olMulti
			}

			actualSpaces := len(spaces)
			if actualSpaces != expectedSpaces {
				violation := value.NewViolation(
					[]string{"MD030", "list-marker-space"},
					"Spaces after list markers",
					nil,
					lineNumber,
				)

				itemType := "single-line"
				if isMultiLine {
					itemType = "multi-line"
				}

				detail := fmt.Sprintf("Expected: %d spaces after ordered list marker (%s), Actual: %d spaces", expectedSpaces, itemType, actualSpaces)
				violation = violation.WithErrorDetail(detail)
				violation = violation.WithErrorContext(strings.TrimSpace(line))
				violation = violation.WithColumn(len(indent) + len(marker) + 1)
				violation = violation.WithLength(actualSpaces)

				// Add fix information
				fixedSpaces := strings.Repeat(" ", expectedSpaces)
				fixedLine := indent + marker + fixedSpaces + content

				fixInfo := value.NewFixInfo().
					WithLineNumber(lineNumber).
					WithEditColumn(1).
					WithDeleteLength(len(line)).
					WithReplaceText(fixedLine)

				violation = violation.WithFixInfo(*fixInfo)
				violations = append(violations, *violation)
			}
		}
	}

	return functional.Ok(violations)
}

// isMultiLineListItem determines if a list item spans multiple lines or contains sub-blocks
func isMultiLineListItem(lines []string, currentIndex int) bool {
	if currentIndex >= len(lines)-1 {
		return false // No more lines to check
	}

	// Get current item's indentation
	currentLine := lines[currentIndex]
	ulRegex := regexp.MustCompile(`^(\s*)([-*+])(\s+)(.*)$`)
	olRegex := regexp.MustCompile(`^(\s*)(\d{1,9}[.)])(\s+)(.*)$`)

	var currentIndent int
	if matches := ulRegex.FindStringSubmatch(currentLine); matches != nil {
		currentIndent = len(matches[1])
	} else if matches := olRegex.FindStringSubmatch(currentLine); matches != nil {
		currentIndent = len(matches[1])
	} else {
		return false
	}

	// Check subsequent lines
	for i := currentIndex + 1; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Empty line
		if trimmed == "" {
			continue
		}

		// Check if this line belongs to the current list item (indented more than the marker)
		leadingSpaces := len(line) - len(strings.TrimLeft(line, " \t"))

		if leadingSpaces > currentIndent {
			// This line is part of the current list item
			// Check if it's a continuation paragraph or sub-block
			if !strings.HasPrefix(strings.TrimLeft(line, " \t"), "- ") &&
				!strings.HasPrefix(strings.TrimLeft(line, " \t"), "* ") &&
				!strings.HasPrefix(strings.TrimLeft(line, " \t"), "+ ") &&
				!regexp.MustCompile(`^\d{1,9}[.)]`).MatchString(strings.TrimLeft(line, " \t")) {
				return true // Found continuation content (paragraph, code block, etc.)
			}
		} else {
			// Line at same or lower indentation - end of current item
			break
		}
	}

	return false
}
