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

// MD055 - Table pipe style
func NewMD055Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md055.md")

	return entity.NewRule(
		[]string{"MD055", "table-pipe-style"},
		"Table pipe style",
		[]string{"table"},
		infoURL,
		"commonmark",
		map[string]interface{}{
			"style": "consistent", // consistent|leading_and_trailing|leading_only|trailing_only|no_leading_or_trailing
		},
		md055Function,
	)
}

type TablePipeStyle int

const (
	TablePipeUnknown             TablePipeStyle = iota
	TablePipeLeadingAndTrailing                 // | cell | cell |
	TablePipeLeadingOnly                        // | cell | cell
	TablePipeTrailingOnly                       //   cell | cell |
	TablePipeNoLeadingOrTrailing                //   cell | cell
)

func md055Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Get configuration
	styleConfig := getStringConfig(params.Config, "style", "consistent")

	var expectedStyle TablePipeStyle
	var firstTableLine int

	// Parse expected style if not consistent
	if styleConfig != "consistent" {
		switch styleConfig {
		case "leading_and_trailing":
			expectedStyle = TablePipeLeadingAndTrailing
		case "leading_only":
			expectedStyle = TablePipeLeadingOnly
		case "trailing_only":
			expectedStyle = TablePipeTrailingOnly
		case "no_leading_or_trailing":
			expectedStyle = TablePipeNoLeadingOrTrailing
		}
	}

	// Process each line
	for i, line := range params.Lines {
		lineNumber := i + 1

		// Check if this is a table row
		if !isTableLine(line) {
			continue
		}

		// Skip table separator rows (contain only |, -, :, spaces)
		trimmed := strings.TrimSpace(line)
		isSeparator := true
		for _, char := range trimmed {
			if char != '|' && char != '-' && char != ':' && char != ' ' {
				isSeparator = false
				break
			}
		}
		if isSeparator && strings.Contains(trimmed, "-") {
			continue
		}

		// Determine current table pipe style
		currentStyle := determineTablePipeStyle(line)

		// For consistent style, establish expected style from first table row
		if styleConfig == "consistent" && expectedStyle == TablePipeUnknown {
			expectedStyle = currentStyle
			firstTableLine = lineNumber
		}

		// Check for style violations
		if currentStyle != expectedStyle {
			violation := value.NewViolation(
				[]string{"MD055", "table-pipe-style"},
				"Table pipe style",
				nil,
				lineNumber,
			)

			expectedStyleName := getTablePipeStyleName(expectedStyle)
			actualStyleName := getTablePipeStyleName(currentStyle)

			detail := fmt.Sprintf("Expected: %s, Actual: %s", expectedStyleName, actualStyleName)
			if styleConfig == "consistent" && firstTableLine > 0 {
				detail += fmt.Sprintf(" [Expected: %s (based on line %d)]", expectedStyleName, firstTableLine)
			}

			violation = violation.WithErrorDetail(detail)
			violation = violation.WithErrorContext(strings.TrimSpace(line))

			// Add fix information - convert to expected style
			fixedLine := convertTablePipeStyle(line, expectedStyle)

			fixInfo := value.NewFixInfo().
				WithLineNumber(lineNumber).
				WithEditColumn(1).
				WithDeleteLength(len(line)).
				WithReplaceText(fixedLine)

			violation = violation.WithFixInfo(*fixInfo)
			violations = append(violations, *violation)
		}
	}

	return functional.Ok(violations)
}

// determineTablePipeStyle determines the pipe style of a table row
func determineTablePipeStyle(line string) TablePipeStyle {
	trimmed := strings.TrimSpace(line)

	hasLeading := strings.HasPrefix(trimmed, "|")
	hasTrailing := strings.HasSuffix(trimmed, "|")

	if hasLeading && hasTrailing {
		return TablePipeLeadingAndTrailing
	} else if hasLeading && !hasTrailing {
		return TablePipeLeadingOnly
	} else if !hasLeading && hasTrailing {
		return TablePipeTrailingOnly
	} else {
		return TablePipeNoLeadingOrTrailing
	}
}

// getTablePipeStyleName returns a human-readable name for table pipe style
func getTablePipeStyleName(style TablePipeStyle) string {
	switch style {
	case TablePipeLeadingAndTrailing:
		return "leading_and_trailing"
	case TablePipeLeadingOnly:
		return "leading_only"
	case TablePipeTrailingOnly:
		return "trailing_only"
	case TablePipeNoLeadingOrTrailing:
		return "no_leading_or_trailing"
	default:
		return "unknown"
	}
}

// convertTablePipeStyle converts a table row to the expected pipe style
func convertTablePipeStyle(line string, expectedStyle TablePipeStyle) string {
	// Extract the leading whitespace and content
	leadingWhitespace := ""
	content := line

	for i, char := range line {
		if char != ' ' && char != '\t' {
			leadingWhitespace = line[:i]
			content = line[i:]
			break
		}
	}

	// Remove existing leading/trailing pipes from content
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "|") {
		content = content[1:]
	}
	if strings.HasSuffix(content, "|") {
		content = content[:len(content)-1]
	}
	content = strings.TrimSpace(content)

	// Apply the expected style
	switch expectedStyle {
	case TablePipeLeadingAndTrailing:
		return leadingWhitespace + "| " + content + " |"
	case TablePipeLeadingOnly:
		return leadingWhitespace + "| " + content
	case TablePipeTrailingOnly:
		return leadingWhitespace + content + " |"
	case TablePipeNoLeadingOrTrailing:
		return leadingWhitespace + content
	default:
		return line // Fallback
	}
}
