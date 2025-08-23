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

// MD011 - Reversed link syntax
func NewMD011Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md011.md")

	return entity.NewRule(
		[]string{"MD011", "no-reversed-links"},
		"Reversed link syntax",
		[]string{"links"},
		infoURL,
		"commonmark",
		map[string]interface{}{},
		md011Function,
	)
}

func md011Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Regex to detect reversed link syntax: (text)[url]
	// But exclude footnotes like (example)[^1]
	reversedLinkRegex := regexp.MustCompile(`\(([^)]+)\)\[([^\]^][^\]]*)\]`)
	footnoteRegex := regexp.MustCompile(`\([^)]*\)\[\^[^\]]+\]`)

	// Process each line
	for i, line := range params.Lines {
		lineNumber := i + 1

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Find all potential reversed links
		matches := reversedLinkRegex.FindAllStringSubmatch(line, -1)
		matchPositions := reversedLinkRegex.FindAllStringIndex(line, -1)

		for j, match := range matches {
			// Skip footnotes (Markdown Extra style)
			if footnoteRegex.MatchString(match[0]) {
				continue
			}

			text := match[1]
			url := match[2]
			pos := matchPositions[j]

			violation := value.NewViolation(
				[]string{"MD011", "no-reversed-links"},
				"Reversed link syntax",
				nil,
				lineNumber,
			)

			violation = violation.WithErrorDetail("Link syntax is reversed")
			violation = violation.WithErrorContext(match[0])
			violation = violation.WithColumn(pos[0] + 1) // 1-based column
			violation = violation.WithLength(pos[1] - pos[0])

			// Add fix information - swap the syntax
			correctSyntax := "[" + text + "](" + url + ")"

			fixInfo := value.NewFixInfo().
				WithLineNumber(lineNumber).
				WithEditColumn(pos[0] + 1).
				WithDeleteLength(pos[1] - pos[0]).
				WithReplaceText(correctSyntax)

			violation = violation.WithFixInfo(*fixInfo)
			violations = append(violations, *violation)
		}
	}

	return functional.Ok(violations)
}
