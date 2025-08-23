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

// MD022 - Headings should be surrounded by blank lines
func NewMD022Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md022.md")

	return entity.NewRule(
		[]string{"MD022", "blanks-around-headings"},
		"Headings should be surrounded by blank lines",
		[]string{"blank_lines", "headings"},
		infoURL,
		"commonmark",
		map[string]interface{}{
			"lines_above": 1, // Blank lines above heading (int or []int for per-level)
			"lines_below": 1, // Blank lines below heading (int or []int for per-level)
		},
		md022Function,
	)
}

func md022Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Get configuration
	linesAbove := getIntConfig(params.Config, "lines_above", 1)
	linesBelow := getIntConfig(params.Config, "lines_below", 1)

	// Regexes for different heading types
	atxRegex := regexp.MustCompile(`^(\s*)(#{1,6})(\s+)(.*)$`)
	setextRegex := regexp.MustCompile(`^(=+|-+)\s*$`)

	// Process each line to find headings
	for i, line := range params.Lines {
		lineNumber := i + 1
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		var isHeading bool
		var headingLevel int
		var headingLineNumber int

		// Check for ATX headings
		if matches := atxRegex.FindStringSubmatch(line); matches != nil {
			isHeading = true
			headingLevel = len(matches[2])
			headingLineNumber = lineNumber
		}

		// Check for Setext headings (underlined)
		if !isHeading && setextRegex.MatchString(line) && i > 0 {
			prevLine := strings.TrimSpace(params.Lines[i-1])
			if prevLine != "" {
				isHeading = true
				headingLevel = 1
				if strings.HasPrefix(trimmed, "-") {
					headingLevel = 2
				}
				headingLineNumber = i // Line number of underline (0-based) + 1

				// For setext headings, we check the text line above
				violations = append(violations, checkHeadingBlanks(params.Lines, i, linesAbove, linesBelow, headingLevel, true)...)
				continue
			}
		}

		if isHeading {
			violations = append(violations, checkHeadingBlanks(params.Lines, headingLineNumber-1, linesAbove, linesBelow, headingLevel, false)...)
		}
	}

	return functional.Ok(violations)
}

// checkHeadingBlanks checks if a heading has the required blank lines around it
func checkHeadingBlanks(lines []string, headingIndex int, linesAbove, linesBelow, headingLevel int, isSetext bool) []value.Violation {
	var violations []value.Violation
	lineNumber := headingIndex + 1

	// Check lines above
	if headingIndex > 0 { // Not the first line
		blankLinesAbove := 0
		for i := headingIndex - 1; i >= 0; i-- {
			if strings.TrimSpace(lines[i]) == "" {
				blankLinesAbove++
			} else {
				break
			}
		}

		if blankLinesAbove < linesAbove {
			violation := value.NewViolation(
				[]string{"MD022", "blanks-around-headings"},
				"Headings should be surrounded by blank lines",
				nil,
				lineNumber,
			)

			detail := fmt.Sprintf("Expected: %d blank lines above, Actual: %d", linesAbove, blankLinesAbove)
			violation = violation.WithErrorDetail(detail)
			violation = violation.WithErrorContext(strings.TrimSpace(lines[headingIndex]))

			// Add fix information - insert blank lines above
			neededLines := linesAbove - blankLinesAbove
			insertText := strings.Repeat("\n", neededLines)

			fixInfo := value.NewFixInfo().
				WithLineNumber(lineNumber).
				WithEditColumn(1).
				WithDeleteLength(0).
				WithReplaceText(insertText)

			violation = violation.WithFixInfo(*fixInfo)
			violations = append(violations, *violation)
		}
	}

	// Check lines below (skip for setext underlines, check the text line instead)
	belowIndex := headingIndex
	if isSetext {
		belowIndex = headingIndex + 1 // Skip the underline for setext
	}

	if belowIndex < len(lines)-1 { // Not the last line
		blankLinesBelow := 0
		for i := belowIndex + 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "" {
				blankLinesBelow++
			} else {
				break
			}
		}

		if blankLinesBelow < linesBelow {
			violation := value.NewViolation(
				[]string{"MD022", "blanks-around-headings"},
				"Headings should be surrounded by blank lines",
				nil,
				lineNumber,
			)

			detail := fmt.Sprintf("Expected: %d blank lines below, Actual: %d", linesBelow, blankLinesBelow)
			violation = violation.WithErrorDetail(detail)
			violation = violation.WithErrorContext(strings.TrimSpace(lines[headingIndex]))

			// Add fix information - insert blank lines below
			neededLines := linesBelow - blankLinesBelow
			insertText := strings.Repeat("\n", neededLines)
			insertLineNumber := belowIndex + 2 // After the heading

			if isSetext {
				insertLineNumber = belowIndex + 1 // After the underline
			}

			fixInfo := value.NewFixInfo().
				WithLineNumber(insertLineNumber).
				WithEditColumn(1).
				WithDeleteLength(0).
				WithReplaceText(insertText)

			violation = violation.WithFixInfo(*fixInfo)
			violations = append(violations, *violation)
		}
	}

	return violations
}
