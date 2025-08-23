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

// MD043 - Required heading structure
func NewMD043Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md043.md")

	return entity.NewRule(
		[]string{"MD043", "required-headings"},
		"Required heading structure",
		[]string{"headings"},
		infoURL,
		"commonmark",
		map[string]interface{}{
			"headings":   []interface{}{}, // Array of required heading texts or patterns
			"match_case": false,           // Whether to match case
		},
		md043Function,
	)
}

func md043Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Get configuration
	requiredHeadings := getStringSliceConfig(params.Config, "headings")
	matchCase := getBoolConfig(params.Config, "match_case", false)

	// If no required headings specified, skip rule
	if len(requiredHeadings) == 0 {
		return functional.Ok(violations)
	}

	// Extract all headings from document
	var foundHeadings []string

	// Regexes for different heading types
	atxRegex := regexp.MustCompile(`^(\s*)(#{1,6})(\s+)(.*)$`)
	setextRegex := regexp.MustCompile(`^(=+|-+)\s*$`)

	// Process each line to find headings
	for i, line := range params.Lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		var headingText string
		var isHeading bool

		// Check for ATX headings
		if matches := atxRegex.FindStringSubmatch(line); matches != nil {
			headingText = strings.TrimSpace(matches[4])
			isHeading = true
		}

		// Check for Setext headings
		if !isHeading && setextRegex.MatchString(line) && i > 0 {
			prevLine := strings.TrimSpace(params.Lines[i-1])
			if prevLine != "" {
				headingText = prevLine
				isHeading = true
			}
		}

		if isHeading && headingText != "" {
			if matchCase {
				foundHeadings = append(foundHeadings, headingText)
			} else {
				foundHeadings = append(foundHeadings, strings.ToLower(headingText))
			}
		}
	}

	// Check if required headings are present
	for _, required := range requiredHeadings {
		requiredText := required
		if !matchCase {
			requiredText = strings.ToLower(required)
		}

		found := false
		for _, heading := range foundHeadings {
			// Support regex patterns by trying to compile as regex
			if matched, err := regexp.MatchString(requiredText, heading); err == nil && matched {
				found = true
				break
			}

			// Also check exact match
			if heading == requiredText {
				found = true
				break
			}
		}

		if !found {
			violation := value.NewViolation(
				[]string{"MD043", "required-headings"},
				"Required heading structure",
				nil,
				1, // Report at top of file
			)

			violation = violation.WithErrorDetail("Missing required heading: " + required)

			violations = append(violations, *violation)
		}
	}

	return functional.Ok(violations)
}
