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

// MD038 - Spaces inside code span elements
func NewMD038Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md038.md")

	return entity.NewRule(
		[]string{"MD038", "no-space-in-code"},
		"Spaces inside code span elements",
		[]string{"code", "whitespace"},
		infoURL,
		"commonmark",
		map[string]interface{}{},
		md038Function,
	)
}

func md038Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Regex for code spans with various patterns
	// Single backtick: `code`
	singleCodeRegex := regexp.MustCompile("(`)([ \t]+)([^`]*?)([  \t]+)(`)")
	// Double backtick: ``code``
	doubleCodeRegex := regexp.MustCompile("(`{2})([ \t]*?)([^`]*?)([ \t]*?)(`{2})")
	// Multiple backticks
	multiCodeRegex := regexp.MustCompile("(`{3,})([ \t]*?)([^`]*?)([ \t]*?)(`{3,})")

	// Process each line
	for i, line := range params.Lines {
		lineNumber := i + 1

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check different code span patterns
		patterns := []struct {
			regex *regexp.Regexp
			name  string
		}{
			{multiCodeRegex, "multi-backtick"},
			{doubleCodeRegex, "double-backtick"},
			{singleCodeRegex, "single-backtick"},
		}

		for _, pattern := range patterns {
			matches := pattern.regex.FindAllStringSubmatch(line, -1)
			positions := pattern.regex.FindAllStringIndex(line, -1)

			for j, match := range matches {
				openTicks := match[1]
				leadingSpace := match[2]
				content := match[3]
				trailingSpace := match[4]
				closeTicks := match[5]
				pos := positions[j]

				// Skip if backticks don't match
				if len(openTicks) != len(closeTicks) {
					continue
				}

				// Check for unnecessary spaces
				hasUnnecessarySpaces := false

				// Special case: single leading and trailing space is allowed for spans that start/end with backtick
				if len(openTicks) == 1 && len(leadingSpace) == 1 && len(trailingSpace) == 1 {
					if strings.HasPrefix(content, "`") || strings.HasSuffix(content, "`") {
						// This is allowed - backtick at start/end needs space
						continue
					}
				}

				// Code spans containing only spaces are allowed
				if strings.TrimSpace(content) == "" {
					continue
				}

				// Check for any unnecessary spaces
				if len(leadingSpace) > 0 || len(trailingSpace) > 0 {
					// Exception: if both leading and trailing have exactly 1 space and content has backticks
					if len(leadingSpace) == 1 && len(trailingSpace) == 1 {
						if strings.HasPrefix(content, "`") || strings.HasSuffix(content, "`") {
							continue // This is the allowed case
						}
					}
					hasUnnecessarySpaces = true
				}

				if hasUnnecessarySpaces {
					violation := value.NewViolation(
						[]string{"MD038", "no-space-in-code"},
						"Spaces inside code span elements",
						nil,
						lineNumber,
					)

					violation = violation.WithErrorDetail("Unnecessary spaces inside code span")
					violation = violation.WithErrorContext(match[0])
					violation = violation.WithColumn(pos[0] + 1)
					violation = violation.WithLength(pos[1] - pos[0])

					// Add fix information - remove unnecessary spaces
					fixedContent := content

					// Special handling: if content starts/ends with backtick, preserve single space
					if strings.HasPrefix(content, "`") || strings.HasSuffix(content, "`") {
						fixedText := openTicks + " " + content + " " + closeTicks

						fixInfo := value.NewFixInfo().
							WithLineNumber(lineNumber).
							WithEditColumn(pos[0] + 1).
							WithDeleteLength(pos[1] - pos[0]).
							WithReplaceText(fixedText)

						violation = violation.WithFixInfo(*fixInfo)
					} else {
						// Remove all leading/trailing spaces
						fixedText := openTicks + fixedContent + closeTicks

						fixInfo := value.NewFixInfo().
							WithLineNumber(lineNumber).
							WithEditColumn(pos[0] + 1).
							WithDeleteLength(pos[1] - pos[0]).
							WithReplaceText(fixedText)

						violation = violation.WithFixInfo(*fixInfo)
					}

					violations = append(violations, *violation)
				}
			}
		}
	}

	return functional.Ok(violations)
}
