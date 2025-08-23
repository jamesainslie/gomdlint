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

// MD021 - Multiple spaces inside hashes on closed ATX style heading
func NewMD021Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md021.md")

	return entity.NewRule(
		[]string{"MD021", "no-multiple-space-closed-atx"},
		"Multiple spaces inside hashes on closed ATX style heading",
		[]string{"atx_closed", "headings", "spaces"},
		infoURL,
		"commonmark",
		map[string]interface{}{},
		md021Function,
	)
}

func md021Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Regex to match closed ATX headings with multiple spaces
	// Matches: #  text  # or ##   text   ## etc.
	closedATXRegex := regexp.MustCompile(`^(\s*)(#{1,6})(\s{2,}|\s*)(.*?)(\s{2,}|\s*)(#{1,6})\s*$`)

	// Process each line
	for i, line := range params.Lines {
		lineNumber := i + 1

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check for closed ATX heading with multiple spaces
		matches := closedATXRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		indent := matches[1]
		openHashes := matches[2]
		openSpaces := matches[3]
		content := matches[4]
		closeSpaces := matches[5]
		closeHashes := matches[6]

		// Check if this is actually a closed ATX heading (has closing hashes)
		if len(closeHashes) == 0 {
			continue
		}

		// Check for multiple spaces after opening hashes
		multipleSpacesAfterOpen := len(openSpaces) > 1

		// Check for multiple spaces before closing hashes
		multipleSpacesBeforeClose := len(closeSpaces) > 1

		if multipleSpacesAfterOpen || multipleSpacesBeforeClose {
			violation := value.NewViolation(
				[]string{"MD021", "no-multiple-space-closed-atx"},
				"Multiple spaces inside hashes on closed ATX style heading",
				nil,
				lineNumber,
			)

			var detail string
			if multipleSpacesAfterOpen && multipleSpacesBeforeClose {
				detail = fmt.Sprintf("Expected: 1 space after opening and before closing hashes, Actual: %d and %d spaces", len(openSpaces), len(closeSpaces))
			} else if multipleSpacesAfterOpen {
				detail = fmt.Sprintf("Expected: 1 space after opening hashes, Actual: %d spaces", len(openSpaces))
			} else {
				detail = fmt.Sprintf("Expected: 1 space before closing hashes, Actual: %d spaces", len(closeSpaces))
			}

			violation = violation.WithErrorDetail(detail)
			violation = violation.WithErrorContext(strings.TrimSpace(line))

			// Add fix information - normalize to single spaces
			fixedOpenSpaces := " "
			if len(openSpaces) == 0 {
				fixedOpenSpaces = " " // Ensure at least one space
			}

			fixedCloseSpaces := " "
			if len(closeSpaces) == 0 {
				fixedCloseSpaces = " " // Ensure at least one space
			}

			fixedLine := indent + openHashes + fixedOpenSpaces + content + fixedCloseSpaces + closeHashes

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
