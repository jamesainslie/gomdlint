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

// MD019 - Multiple spaces after hash on ATX style heading
func NewMD019Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md019.md")

	return entity.NewRule(
		[]string{"MD019", "no-multiple-space-atx"},
		"Multiple spaces after hash on ATX style heading",
		[]string{"atx", "headings", "spaces"},
		infoURL,
		"commonmark",
		map[string]interface{}{},
		md019Function,
	)
}

func md019Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Regex to match ATX headings with multiple spaces after hash
	// Matches: #  heading, ##   heading, etc. (2+ spaces after #)
	multipleSpaceRegex := regexp.MustCompile(`^(\s*)(#{1,6})(\s{2,})(.*)$`)

	// Process each line
	for i, line := range params.Lines {
		lineNumber := i + 1

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check for ATX heading with multiple spaces after hash
		matches := multipleSpaceRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		indent := matches[1]
		hashes := matches[2]
		spaces := matches[3]

		// Count the number of excess spaces (should be only 1)
		excessSpaces := len(spaces) - 1

		violation := value.NewViolation(
			[]string{"MD019", "no-multiple-space-atx"},
			"Multiple spaces after hash on ATX style heading",
			nil,
			lineNumber,
		)

		detail := fmt.Sprintf("Expected: 1 space, Actual: %d spaces", len(spaces))
		violation = violation.WithErrorDetail(detail)
		violation = violation.WithErrorContext(strings.TrimSpace(line))
		violation = violation.WithColumn(len(indent) + len(hashes) + 2) // Position after first space
		violation = violation.WithLength(excessSpaces)

		// Add fix information - remove excess spaces
		fixInfo := value.NewFixInfo().
			WithLineNumber(lineNumber).
			WithEditColumn(len(indent) + len(hashes) + 2). // Position after first space
			WithDeleteLength(excessSpaces).                // Delete excess spaces
			WithReplaceText("")                            // Replace with nothing

		violation = violation.WithFixInfo(*fixInfo)

		violations = append(violations, *violation)
	}

	return functional.Ok(violations)
}
