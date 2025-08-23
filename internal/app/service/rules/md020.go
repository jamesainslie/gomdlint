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

// MD020 - No space inside hashes on closed ATX style heading
func NewMD020Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md020.md")

	return entity.NewRule(
		[]string{"MD020", "no-missing-space-closed-atx"},
		"No space inside hashes on closed ATX style heading",
		[]string{"atx_closed", "headings", "spaces"},
		infoURL,
		"commonmark",
		map[string]interface{}{},
		md020Function,
	)
}

func md020Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Regex to match closed ATX headings with missing spaces
	// Matches: #text# or #text ## or ## text# etc.
	closedATXRegex := regexp.MustCompile(`^(\s*)(#{1,6})([^#]*?)(#{1,6})\s*$`)

	// Process each line
	for i, line := range params.Lines {
		lineNumber := i + 1

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check for closed ATX heading
		matches := closedATXRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		indent := matches[1]
		openHashes := matches[2]
		content := matches[3]
		closeHashes := matches[4]

		// Check if this is actually a closed ATX heading (has closing hashes)
		if len(closeHashes) == 0 {
			continue
		}

		// Check for missing space after opening hashes
		missingSpaceAfterOpen := !strings.HasPrefix(content, " ")

		// Check for missing space before closing hashes
		missingSpaceBeforeClose := !strings.HasSuffix(content, " ")

		if missingSpaceAfterOpen || missingSpaceBeforeClose {
			violation := value.NewViolation(
				[]string{"MD020", "no-missing-space-closed-atx"},
				"No space inside hashes on closed ATX style heading",
				nil,
				lineNumber,
			)

			var detail string
			if missingSpaceAfterOpen && missingSpaceBeforeClose {
				detail = "Missing space after opening and before closing hashes"
			} else if missingSpaceAfterOpen {
				detail = "Missing space after opening hashes"
			} else {
				detail = "Missing space before closing hashes"
			}

			violation = violation.WithErrorDetail(detail)
			violation = violation.WithErrorContext(strings.TrimSpace(line))

			// Add fix information
			fixedContent := content
			if missingSpaceAfterOpen && !strings.HasPrefix(fixedContent, " ") {
				fixedContent = " " + fixedContent
			}
			if missingSpaceBeforeClose && !strings.HasSuffix(fixedContent, " ") {
				fixedContent = fixedContent + " "
			}

			fixedLine := indent + openHashes + fixedContent + closeHashes

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
