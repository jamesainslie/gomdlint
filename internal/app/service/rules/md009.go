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

// MD009 - Trailing spaces
func NewMD009Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md009.md")

	return entity.NewRule(
		[]string{"MD009", "no-trailing-spaces"},
		"Trailing spaces",
		[]string{"whitespace"},
		infoURL,
		"commonmark",
		map[string]interface{}{
			"br_spaces":             2,     // Spaces for line breaks (2+ spaces = <br>)
			"list_item_empty_lines": false, // Allow trailing spaces in empty list item lines
			"strict":                false, // Include unnecessary breaks (even when using br_spaces)
		},
		md009Function,
	)
}

func md009Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Get configuration
	brSpaces := getIntConfig(params.Config, "br_spaces", 2)
	listItemEmptyLines := getBoolConfig(params.Config, "list_item_empty_lines", false)
	strict := getBoolConfig(params.Config, "strict", false)

	// Ensure br_spaces is at least 2 to be effective (per markdownlint spec)
	if brSpaces < 2 {
		brSpaces = 0 // Disable if less than 2
	}

	// Process each line
	for i, line := range params.Lines {
		lineNumber := i + 1

		// Find trailing whitespace
		trailingSpaces := getTrailingSpaceCount(line)
		if trailingSpaces == 0 {
			continue
		}

		// Check if this line should be ignored
		if shouldIgnoreTrailingSpaces(params.Tokens, params.Lines, lineNumber, listItemEmptyLines) {
			continue
		}

		// Determine if this is a valid line break (br_spaces or more trailing spaces)
		isValidLineBreak := brSpaces > 0 && trailingSpaces >= brSpaces

		// Determine if we should flag this violation
		shouldFlag := false
		if !isValidLineBreak {
			// Always flag invalid trailing spaces (less than br_spaces)
			shouldFlag = true
		} else {
			// It's a valid line break (enough spaces), but check if it's actually creating a break
			if !isActualLineBreak(params.Lines, lineNumber) {
				// Not creating an actual line break, so flag it
				shouldFlag = true
			} else if strict {
				// In strict mode, flag even valid line breaks
				shouldFlag = true
			}
			// If it's a valid line break and not in strict mode, don't flag
		}

		if shouldFlag {
			violation := value.NewViolation(
				[]string{"MD009", "no-trailing-spaces"},
				"Trailing spaces",
				nil,
				lineNumber,
			)

			// Calculate the column where trailing spaces start
			contentLength := len(strings.TrimRightFunc(line, func(r rune) bool {
				return r == ' ' || r == '\t'
			}))

			detail := fmt.Sprintf("Expected: 0 or %d, Actual: %d", brSpaces, trailingSpaces)
			if brSpaces == 0 {
				detail = fmt.Sprintf("Expected: 0, Actual: %d", trailingSpaces)
			}

			violation = violation.WithColumn(contentLength + 1) // 1-based column
			violation = violation.WithLength(trailingSpaces)
			violation = violation.WithErrorDetail(detail)

			// Add fix information - remove trailing spaces
			fixInfo := value.NewFixInfo().
				WithLineNumber(lineNumber).
				WithEditColumn(contentLength + 1).
				WithDeleteLength(trailingSpaces).
				WithReplaceText("")

			violation = violation.WithFixInfo(*fixInfo)

			violations = append(violations, *violation)
		}
	}

	return functional.Ok(violations)
}

// getTrailingSpaceCount returns the number of trailing spaces in a line
func getTrailingSpaceCount(line string) int {
	count := 0
	for i := len(line) - 1; i >= 0; i-- {
		if line[i] == ' ' {
			count++
		} else {
			break
		}
	}
	return count
}

// shouldIgnoreTrailingSpaces determines if trailing spaces should be ignored for this line
func shouldIgnoreTrailingSpaces(tokens []value.Token, lines []string, lineNumber int, listItemEmptyLines bool) bool {
	// Find the token containing this line
	containingToken := findTokenContainingLine(tokens, lineNumber)
	if containingToken == nil {
		return false
	}

	// Always allow trailing spaces in code blocks
	if containingToken.IsCodeBlock() {
		return true
	}

	// Check for list item empty lines if configured
	if listItemEmptyLines && containingToken.IsType(value.TokenTypeListItem) {
		line := lines[lineNumber-1] // Convert to 0-based
		if strings.TrimSpace(line) == "" {
			return true
		}
	}

	return false
}

// isActualLineBreak determines if trailing spaces actually create a line break
func isActualLineBreak(lines []string, lineNumber int) bool {
	// Check if this is the last line or followed by an empty line
	if lineNumber >= len(lines) {
		return false // Last line, no break possible
	}

	nextLine := lines[lineNumber] // Convert to 0-based (lineNumber is 1-based)

	// If next line is empty or starts a new block element, it's not a line break
	trimmedNext := strings.TrimSpace(nextLine)
	if trimmedNext == "" {
		return false // Empty line follows, so not creating a <br>
	}

	// Check if next line starts a new block element (simplified check)
	blockStarters := []string{"#", ">", "-", "*", "+", "1.", "2.", "3.", "4.", "5.", "6.", "7.", "8.", "9."}
	for _, starter := range blockStarters {
		if strings.HasPrefix(trimmedNext, starter) {
			return false // Block element follows
		}
	}

	// Check for horizontal rules
	hrRegex := regexp.MustCompile(`^(\*\s*){3,}$|^(-\s*){3,}$|^(_\s*){3,}$`)
	if hrRegex.MatchString(trimmedNext) {
		return false
	}

	// Check for fenced code blocks
	if strings.HasPrefix(trimmedNext, "```") || strings.HasPrefix(trimmedNext, "~~~") {
		return false
	}

	// If we get here, it's likely a line break within a paragraph
	return true
}
