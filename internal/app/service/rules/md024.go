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

// MD024 - Multiple headings with the same content
func NewMD024Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md024.md")

	return entity.NewRule(
		[]string{"MD024", "no-duplicate-heading"},
		"Multiple headings with the same content",
		[]string{"headings"},
		infoURL,
		"commonmark",
		map[string]interface{}{
			"siblings_only": false, // Only check sibling headings (same parent level)
		},
		md024Function,
	)
}

type headingInfo struct {
	text   string
	level  int
	line   int
	parent *headingInfo
}

func md024Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Get configuration
	siblingsOnly := getBoolConfig(params.Config, "siblings_only", false)

	// Regexes for different heading types
	atxRegex := regexp.MustCompile(`^(\s*)(#{1,6})(\s+)(.*)$`)
	setextRegex := regexp.MustCompile(`^(=+|-+)\s*$`)

	var headings []headingInfo

	// Process each line to collect headings
	for i, line := range params.Lines {
		lineNumber := i + 1
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		var heading headingInfo
		var isHeading bool

		// Check for ATX headings
		if matches := atxRegex.FindStringSubmatch(line); matches != nil {
			heading = headingInfo{
				text:  strings.TrimSpace(matches[4]),
				level: len(matches[2]),
				line:  lineNumber,
			}
			isHeading = true
		}

		// Check for Setext headings
		if !isHeading && setextRegex.MatchString(line) && i > 0 {
			prevLine := strings.TrimSpace(params.Lines[i-1])
			if prevLine != "" {
				level := 1
				if strings.HasPrefix(trimmed, "-") {
					level = 2
				}
				heading = headingInfo{
					text:  prevLine,
					level: level,
					line:  i, // Line number of the text (i is 0-based, but we want the text line)
				}
				isHeading = true
			}
		}

		if isHeading {
			// Find parent heading (last heading with lower level)
			for j := len(headings) - 1; j >= 0; j-- {
				if headings[j].level < heading.level {
					heading.parent = &headings[j]
					break
				}
			}

			headings = append(headings, heading)
		}
	}

	// Check for duplicates
	for i, heading := range headings {
		for j := i + 1; j < len(headings); j++ {
			other := headings[j]

			// Normalize text for comparison (case-insensitive, trim)
			headingText := strings.ToLower(strings.TrimSpace(heading.text))
			otherText := strings.ToLower(strings.TrimSpace(other.text))

			if headingText == otherText && headingText != "" {
				// Check if we should flag this based on siblings_only setting
				shouldFlag := true

				if siblingsOnly {
					// Only flag if they have the same parent
					if heading.parent == nil && other.parent == nil {
						// Both are top-level
						shouldFlag = true
					} else if heading.parent != nil && other.parent != nil {
						// Both have parents - check if same parent
						shouldFlag = heading.parent == other.parent
					} else {
						// One has parent, one doesn't - not siblings
						shouldFlag = false
					}
				}

				if shouldFlag {
					violation := value.NewViolation(
						[]string{"MD024", "no-duplicate-heading"},
						"Multiple headings with the same content",
						nil,
						other.line,
					)

					violation = violation.WithErrorDetail("Duplicate heading text")
					violation = violation.WithErrorContext(other.text)

					violations = append(violations, *violation)
				}
			}
		}
	}

	return functional.Ok(violations)
}
