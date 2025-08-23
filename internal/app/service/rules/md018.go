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

// MD018 - No space after hash on ATX style heading
func NewMD018Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md018.md")

	return entity.NewRule(
		[]string{"MD018", "no-missing-space-atx"},
		"No space after hash on ATX style heading",
		[]string{"atx", "headings", "spaces"},
		infoURL,
		"commonmark",
		map[string]interface{}{},
		md018Function,
	)
}

func md018Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Regex to match ATX headings without space after hash
	// Matches: #heading, ##heading, etc. (without space after #)
	noSpaceRegex := regexp.MustCompile(`^(\s*)(#{1,6})([^#\s][^#]*?)(\s*)$`)

	// Process each line
	for i, line := range params.Lines {
		lineNumber := i + 1

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check for ATX heading without space after hash
		matches := noSpaceRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		indent := matches[1]
		hashes := matches[2]

		// This is a heading without space after hash
		violation := value.NewViolation(
			[]string{"MD018", "no-missing-space-atx"},
			"No space after hash on ATX style heading",
			nil,
			lineNumber,
		)

		violation = violation.WithErrorDetail("Missing space after hash(es)")
		violation = violation.WithErrorContext(strings.TrimSpace(line))
		violation = violation.WithColumn(len(indent) + len(hashes) + 1) // Position after last hash

		// Add fix information - insert a space after the hashes
		fixInfo := value.NewFixInfo().
			WithLineNumber(lineNumber).
			WithEditColumn(len(indent) + len(hashes) + 1). // Position after last hash
			WithDeleteLength(0).                           // Don't delete anything
			WithReplaceText(" ")                           // Insert a space

		violation = violation.WithFixInfo(*fixInfo)

		violations = append(violations, *violation)
	}

	return functional.Ok(violations)
}
