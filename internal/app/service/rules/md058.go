package rules

import (
	"context"
	"net/url"
	"strings"

	"github.com/gomdlint/gomdlint/internal/domain/entity"
	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// MD058 - Tables should be surrounded by blank lines
func NewMD058Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md058.md")

	return entity.NewRule(
		[]string{"MD058", "blanks-around-tables"},
		"Tables should be surrounded by blank lines",
		[]string{"blank_lines", "table"},
		infoURL,
		"commonmark",
		map[string]interface{}{},
		md058Function,
	)
}

func md058Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Track table state
	inTable := false
	tableStart := -1
	tableEnd := -1

	// Process each line to identify table boundaries
	for i, line := range params.Lines {
		trimmed := strings.TrimSpace(line)

		// Check if this is a table row (contains pipes and isn't empty)
		isTableRow := isTableLine(line)

		if isTableRow {
			if !inTable {
				// Starting a new table
				inTable = true
				tableStart = i
			}
			tableEnd = i // Update end of table
		} else if trimmed == "" {
			// Blank line - continue current state
			continue
		} else {
			// Non-blank, non-table line
			if inTable {
				// End of table - check for blank lines around it
				violations = append(violations, checkTableBlanks(params.Lines, tableStart, tableEnd)...)
				inTable = false
				tableStart = -1
				tableEnd = -1
			}
		}
	}

	// Check final table if file ends with one
	if inTable {
		violations = append(violations, checkTableBlanks(params.Lines, tableStart, tableEnd)...)
	}

	return functional.Ok(violations)
}

// isTableLine checks if a line is part of a table
func isTableLine(line string) bool {
	trimmed := strings.TrimSpace(line)

	// Must contain at least one pipe
	if !strings.Contains(trimmed, "|") {
		return false
	}

	// Skip if it's likely code (indented by 4+ spaces)
	if strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t") {
		return false
	}

	// Simple heuristic: if it has pipes and isn't obviously not a table
	// A table row typically has format: | cell | cell | or starts/ends with |

	// Check for table separator row (contains only |, -, :, and spaces)
	separatorPattern := true
	for _, char := range trimmed {
		if char != '|' && char != '-' && char != ':' && char != ' ' {
			separatorPattern = false
			break
		}
	}

	if separatorPattern && strings.Contains(trimmed, "-") {
		return true // This is a table separator
	}

	// Check for regular table row
	if strings.Contains(trimmed, "|") {
		// Split by pipes and see if it looks table-like
		parts := strings.Split(trimmed, "|")

		// If it has multiple parts separated by pipes, likely a table
		if len(parts) >= 2 {
			return true
		}
	}

	return false
}

// checkTableBlanks checks if a table has blank lines before and after it
func checkTableBlanks(lines []string, tableStart, tableEnd int) []value.Violation {
	var violations []value.Violation

	// Check blank line before table
	if tableStart > 0 {
		prevLine := strings.TrimSpace(lines[tableStart-1])
		if prevLine != "" {
			violation := value.NewViolation(
				[]string{"MD058", "blanks-around-tables"},
				"Tables should be surrounded by blank lines",
				nil,
				tableStart+1, // 1-based line number
			)

			violation = violation.WithErrorDetail("Table should be preceded by blank line")
			violation = violation.WithErrorContext(strings.TrimSpace(lines[tableStart]))

			// Add fix information - insert blank line before table
			fixInfo := value.NewFixInfo().
				WithLineNumber(tableStart + 1).
				WithEditColumn(1).
				WithDeleteLength(0).
				WithReplaceText("\n")

			violation = violation.WithFixInfo(*fixInfo)
			violations = append(violations, *violation)
		}
	}

	// Check blank line after table
	if tableEnd < len(lines)-1 {
		nextLine := strings.TrimSpace(lines[tableEnd+1])
		if nextLine != "" {
			violation := value.NewViolation(
				[]string{"MD058", "blanks-around-tables"},
				"Tables should be surrounded by blank lines",
				nil,
				tableEnd+1, // 1-based line number
			)

			violation = violation.WithErrorDetail("Table should be followed by blank line")
			violation = violation.WithErrorContext(strings.TrimSpace(lines[tableEnd]))

			// Add fix information - insert blank line after table
			fixInfo := value.NewFixInfo().
				WithLineNumber(tableEnd + 2). // After the table row
				WithEditColumn(1).
				WithDeleteLength(0).
				WithReplaceText("\n")

			violation = violation.WithFixInfo(*fixInfo)
			violations = append(violations, *violation)
		}
	}

	return violations
}
