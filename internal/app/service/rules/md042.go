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

// MD042 - No empty links
func NewMD042Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md042.md")

	return entity.NewRule(
		[]string{"MD042", "no-empty-links"},
		"No empty links",
		[]string{"links"},
		infoURL,
		"commonmark",
		map[string]interface{}{},
		md042Function,
	)
}

func md042Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Regex patterns for different link types
	inlineLinkRegex := regexp.MustCompile(`\[([^\]]*)\]\(([^)]*)\)`)
	referenceLinkRegex := regexp.MustCompile(`\[([^\]]*)\]\[([^\]]*)\]`)

	// Process each line
	for i, line := range params.Lines {
		lineNumber := i + 1

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check inline links [text](url)
		matches := inlineLinkRegex.FindAllStringSubmatch(line, -1)
		positions := inlineLinkRegex.FindAllStringIndex(line, -1)

		for j, match := range matches {
			url := strings.TrimSpace(match[2])

			// Check for empty URL or just fragment
			if url == "" || url == "#" {
				pos := positions[j]

				violation := value.NewViolation(
					[]string{"MD042", "no-empty-links"},
					"No empty links",
					nil,
					lineNumber,
				)

				var detail string
				if url == "" {
					detail = "Link has empty destination"
				} else {
					detail = "Link has empty fragment"
				}

				violation = violation.WithErrorDetail(detail)
				violation = violation.WithErrorContext(match[0])
				violation = violation.WithColumn(pos[0] + 1) // 1-based column
				violation = violation.WithLength(pos[1] - pos[0])

				violations = append(violations, *violation)
			}
		}

		// Check reference links [text][ref]
		refMatches := referenceLinkRegex.FindAllStringSubmatch(line, -1)
		refPositions := referenceLinkRegex.FindAllStringIndex(line, -1)

		for j, match := range refMatches {
			ref := strings.TrimSpace(match[2])

			// Empty reference in [text][]
			if ref == "" {
				pos := refPositions[j]

				violation := value.NewViolation(
					[]string{"MD042", "no-empty-links"},
					"No empty links",
					nil,
					lineNumber,
				)

				violation = violation.WithErrorDetail("Reference link has empty reference")
				violation = violation.WithErrorContext(match[0])
				violation = violation.WithColumn(pos[0] + 1)
				violation = violation.WithLength(pos[1] - pos[0])

				violations = append(violations, *violation)
			}
		}

		// Note: For shortcut links [text], we would need to check if the reference
		// definition exists elsewhere in the document. This is more complex and
		// would require parsing all reference definitions first.
	}

	return functional.Ok(violations)
}
