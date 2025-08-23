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

// MD032 - Lists should be surrounded by blank lines
func NewMD032Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md032.md")

	return entity.NewRule(
		[]string{"MD032", "blanks-around-lists"},
		"Lists should be surrounded by blank lines",
		[]string{"bullet", "ol", "ul", "blank_lines"},
		infoURL,
		"commonmark",
		map[string]interface{}{},
		md032Function,
	)
}

func md032Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Regex for list items
	ulRegex := regexp.MustCompile(`^(\s*)([-*+])(\s+)(.*)$`)
	olRegex := regexp.MustCompile(`^(\s*)(\d{1,9}[.)])(\s+)(.*)$`)

	// Track list state
	inList := false
	listStart := -1
	listEnd := -1

	// Process each line to identify list boundaries
	for i, line := range params.Lines {
		trimmed := strings.TrimSpace(line)

		// Check if this is a list item
		isListItem := ulRegex.MatchString(line) || olRegex.MatchString(line)

		if isListItem {
			if !inList {
				// Starting a new list
				inList = true
				listStart = i
			}
			listEnd = i // Update end of list
		} else if trimmed == "" {
			// Blank line - continue current state
			continue
		} else {
			// Non-blank, non-list line
			if inList {
				// End of list - check for blank lines around it
				violations = append(violations, checkListBlanks(params.Lines, listStart, listEnd)...)
				inList = false
				listStart = -1
				listEnd = -1
			}
		}
	}

	// Check final list if file ends with one
	if inList {
		violations = append(violations, checkListBlanks(params.Lines, listStart, listEnd)...)
	}

	return functional.Ok(violations)
}

// checkListBlanks checks if a list has blank lines before and after it
func checkListBlanks(lines []string, listStart, listEnd int) []value.Violation {
	var violations []value.Violation

	// Check blank line before list
	if listStart > 0 {
		prevLine := strings.TrimSpace(lines[listStart-1])
		if prevLine != "" {
			// Check if previous line is also a list item (in which case, no blank needed)
			ulRegex := regexp.MustCompile(`^(\s*)([-*+])(\s+)(.*)$`)
			olRegex := regexp.MustCompile(`^(\s*)(\d{1,9}[.)])(\s+)(.*)$`)

			isPrevListItem := ulRegex.MatchString(lines[listStart-1]) || olRegex.MatchString(lines[listStart-1])

			if !isPrevListItem {
				violation := value.NewViolation(
					[]string{"MD032", "blanks-around-lists"},
					"Lists should be surrounded by blank lines",
					nil,
					listStart+1, // 1-based line number
				)

				violation = violation.WithErrorDetail("List should be preceded by blank line")
				violation = violation.WithErrorContext(strings.TrimSpace(lines[listStart]))

				// Add fix information - insert blank line before list
				fixInfo := value.NewFixInfo().
					WithLineNumber(listStart + 1).
					WithEditColumn(1).
					WithDeleteLength(0).
					WithReplaceText("\n")

				violation = violation.WithFixInfo(*fixInfo)
				violations = append(violations, *violation)
			}
		}
	}

	// Check blank line after list
	if listEnd < len(lines)-1 {
		nextLine := strings.TrimSpace(lines[listEnd+1])
		if nextLine != "" {
			// Check if next line is also a list item (in which case, no blank needed)
			ulRegex := regexp.MustCompile(`^(\s*)([-*+])(\s+)(.*)$`)
			olRegex := regexp.MustCompile(`^(\s*)(\d{1,9}[.)])(\s+)(.*)$`)

			isNextListItem := ulRegex.MatchString(lines[listEnd+1]) || olRegex.MatchString(lines[listEnd+1])

			if !isNextListItem {
				violation := value.NewViolation(
					[]string{"MD032", "blanks-around-lists"},
					"Lists should be surrounded by blank lines",
					nil,
					listEnd+1, // 1-based line number
				)

				violation = violation.WithErrorDetail("List should be followed by blank line")
				violation = violation.WithErrorContext(strings.TrimSpace(lines[listEnd]))

				// Add fix information - insert blank line after list
				fixInfo := value.NewFixInfo().
					WithLineNumber(listEnd + 2). // After the list item
					WithEditColumn(1).
					WithDeleteLength(0).
					WithReplaceText("\n")

				violation = violation.WithFixInfo(*fixInfo)
				violations = append(violations, *violation)
			}
		}
	}

	return violations
}
