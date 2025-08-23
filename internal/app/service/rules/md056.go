package rules

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/gomdlint/gomdlint/internal/domain/entity"
	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// MD056 - Table column count
func NewMD056Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md056.md")

	return entity.NewRule(
		[]string{"MD056", "table-column-count"},
		"Table column count",
		[]string{"table"},
		infoURL,
		"commonmark",
		map[string]interface{}{},
		md056Function,
	)
}

type tableRow struct {
	line    string
	lineNum int
	columns int
}

func md056Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Track table state
	var currentTable []tableRow

	// Process each line
	for i, line := range params.Lines {
		lineNumber := i + 1

		if isTableLine(line) {
			// Count columns in this row
			columns := countTableColumns(line)

			currentTable = append(currentTable, tableRow{
				line:    line,
				lineNum: lineNumber,
				columns: columns,
			})
		} else {
			// End of table or non-table line
			if len(currentTable) > 0 {
				// Check the table for column count consistency
				violations = append(violations, checkTableColumnCount(currentTable)...)
				currentTable = []tableRow{}
			}
		}
	}

	// Check final table if file ends with one
	if len(currentTable) > 0 {
		violations = append(violations, checkTableColumnCount(currentTable)...)
	}

	return functional.Ok(violations)
}

// countTableColumns counts the number of columns in a table row
func countTableColumns(line string) int {
	trimmed := strings.TrimSpace(line)

	// Remove leading and trailing pipes if present
	if strings.HasPrefix(trimmed, "|") {
		trimmed = trimmed[1:]
	}
	if strings.HasSuffix(trimmed, "|") {
		trimmed = trimmed[:len(trimmed)-1]
	}

	// Count pipes + 1 to get column count
	if strings.TrimSpace(trimmed) == "" {
		return 0
	}

	return strings.Count(trimmed, "|") + 1
}

// checkTableColumnCount checks if all rows in a table have the same number of columns
func checkTableColumnCount(table []tableRow) []value.Violation {
	var violations []value.Violation

	if len(table) == 0 {
		return violations
	}

	// Find expected column count (from first row or most common count)
	expectedColumns := table[0].columns

	// Check if there's a separator row to determine correct column count
	for _, row := range table {
		trimmed := strings.TrimSpace(row.line)
		isSeparator := true
		for _, char := range trimmed {
			if char != '|' && char != '-' && char != ':' && char != ' ' {
				isSeparator = false
				break
			}
		}

		if isSeparator && strings.Contains(trimmed, "-") {
			// This is the separator row - use its column count as the reference
			expectedColumns = row.columns
			break
		}
	}

	// Check each row
	for _, row := range table {
		if row.columns != expectedColumns {
			violation := value.NewViolation(
				[]string{"MD056", "table-column-count"},
				"Table column count",
				nil,
				row.lineNum,
			)

			detail := fmt.Sprintf("Expected: %d columns, Actual: %d columns", expectedColumns, row.columns)
			violation = violation.WithErrorDetail(detail)
			violation = violation.WithErrorContext(strings.TrimSpace(row.line))

			violations = append(violations, *violation)
		}
	}

	return violations
}
