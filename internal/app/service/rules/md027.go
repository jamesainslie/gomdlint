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

// MD027 - Multiple spaces after blockquote symbol
func NewMD027Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md027.md")

	return entity.NewRule(
		[]string{"MD027", "no-multiple-space-blockquote"},
		"Multiple spaces after blockquote symbol",
		[]string{"blockquote", "indentation", "whitespace"},
		infoURL,
		"commonmark",
		map[string]interface{}{
			"list_items": true, // Include list items in blockquotes
		},
		md027Function,
	)
}

func md027Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Get configuration
	checkListItems := getBoolConfig(params.Config, "list_items", true)

	// Regex for blockquote with multiple spaces
	blockquoteRegex := regexp.MustCompile(`^(\s*)(>\s{2,})(.*)$`)

	// Process each line
	for i, line := range params.Lines {
		lineNumber := i + 1

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check for blockquote with multiple spaces
		matches := blockquoteRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		indent := matches[1]
		blockquotePrefix := matches[2] // Contains > and the extra spaces
		content := matches[3]

		// Count spaces after >
		spacesAfterSymbol := len(blockquotePrefix) - 1 // Subtract 1 for the > symbol

		// Skip if this is a list item in blockquote and list_items is false
		if !checkListItems {
			listItemRegex := regexp.MustCompile(`^\s*([-*+]|\d+\.)\s`)
			if listItemRegex.MatchString(content) {
				continue
			}
		}

		if spacesAfterSymbol > 1 {
			violation := value.NewViolation(
				[]string{"MD027", "no-multiple-space-blockquote"},
				"Multiple spaces after blockquote symbol",
				nil,
				lineNumber,
			)

			detail := fmt.Sprintf("Expected: 1 space, Actual: %d spaces", spacesAfterSymbol)
			violation = violation.WithErrorDetail(detail)
			violation = violation.WithErrorContext(strings.TrimSpace(line))

			// Position after the > symbol
			symbolPos := len(indent) + 1
			violation = violation.WithColumn(symbolPos + 1)
			violation = violation.WithLength(spacesAfterSymbol - 1) // Excess spaces

			// Add fix information - normalize to single space
			fixedPrefix := "> "
			fixedLine := indent + fixedPrefix + content

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
