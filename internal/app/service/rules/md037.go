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

// MD037 - Spaces inside emphasis markers
func NewMD037Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md037.md")

	return entity.NewRule(
		[]string{"MD037", "no-space-in-emphasis"},
		"Spaces inside emphasis markers",
		[]string{"emphasis", "whitespace"},
		infoURL,
		"commonmark",
		map[string]interface{}{},
		md037Function,
	)
}

func md037Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Regex patterns for emphasis and strong with spaces
	emphasisSpaceRegex := regexp.MustCompile(`([*_])\s+([^*_]*?)\s+([*_])`)
	strongSpaceRegex := regexp.MustCompile(`([*_]{2})\s+([^*_]*?)\s+([*_]{2})`)

	// Process each line
	for i, line := range params.Lines {
		lineNumber := i + 1

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check for strong emphasis with spaces first (to avoid conflicts)
		strongMatches := strongSpaceRegex.FindAllStringSubmatch(line, -1)
		strongPositions := strongSpaceRegex.FindAllStringIndex(line, -1)

		for j, match := range strongMatches {
			openMarker := match[1]
			content := match[2]
			closeMarker := match[3]
			pos := strongPositions[j]

			// Only process if markers match
			if openMarker == closeMarker {
				violation := value.NewViolation(
					[]string{"MD037", "no-space-in-emphasis"},
					"Spaces inside emphasis markers",
					nil,
					lineNumber,
				)

				violation = violation.WithErrorDetail("Spaces found inside strong emphasis markers")
				violation = violation.WithErrorContext(match[0])
				violation = violation.WithColumn(pos[0] + 1)
				violation = violation.WithLength(pos[1] - pos[0])

				// Add fix information - remove spaces
				fixedText := openMarker + strings.TrimSpace(content) + closeMarker

				fixInfo := value.NewFixInfo().
					WithLineNumber(lineNumber).
					WithEditColumn(pos[0] + 1).
					WithDeleteLength(pos[1] - pos[0]).
					WithReplaceText(fixedText)

				violation = violation.WithFixInfo(*fixInfo)
				violations = append(violations, *violation)
			}
		}

		// Remove strong emphasis matches from line to avoid conflicts
		tempLine := line
		for i := len(strongPositions) - 1; i >= 0; i-- {
			pos := strongPositions[i]
			tempLine = tempLine[:pos[0]] + strings.Repeat("X", pos[1]-pos[0]) + tempLine[pos[1]:]
		}

		// Check for single emphasis with spaces
		emphasisMatches := emphasisSpaceRegex.FindAllStringSubmatch(tempLine, -1)
		emphasisPositions := emphasisSpaceRegex.FindAllStringIndex(tempLine, -1)

		for j, match := range emphasisMatches {
			// Skip if this was replaced (contains X)
			if strings.Contains(match[0], "X") {
				continue
			}

			openMarker := match[1]
			content := match[2]
			closeMarker := match[3]
			pos := emphasisPositions[j]

			// Only process if markers match
			if openMarker == closeMarker {
				// Get original text from original line
				originalMatch := line[pos[0]:pos[1]]

				violation := value.NewViolation(
					[]string{"MD037", "no-space-in-emphasis"},
					"Spaces inside emphasis markers",
					nil,
					lineNumber,
				)

				violation = violation.WithErrorDetail("Spaces found inside emphasis markers")
				violation = violation.WithErrorContext(originalMatch)
				violation = violation.WithColumn(pos[0] + 1)
				violation = violation.WithLength(pos[1] - pos[0])

				// Add fix information - remove spaces
				fixedText := openMarker + strings.TrimSpace(content) + closeMarker

				fixInfo := value.NewFixInfo().
					WithLineNumber(lineNumber).
					WithEditColumn(pos[0] + 1).
					WithDeleteLength(pos[1] - pos[0]).
					WithReplaceText(fixedText)

				violation = violation.WithFixInfo(*fixInfo)
				violations = append(violations, *violation)
			}
		}
	}

	return functional.Ok(violations)
}
