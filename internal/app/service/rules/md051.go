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

// MD051 - Link fragments should be valid
func NewMD051Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md051.md")

	return entity.NewRule(
		[]string{"MD051", "link-fragments"},
		"Link fragments should be valid",
		[]string{"links"},
		infoURL,
		"commonmark",
		map[string]interface{}{},
		md051Function,
	)
}

func md051Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Collect all headings that could be valid fragment targets
	validFragments := make(map[string]bool)

	// Regex for headings
	atxRegex := regexp.MustCompile(`^(\s*)(#{1,6})(\s+)(.*)$`)
	setextRegex := regexp.MustCompile(`^(=+|-+)\s*$`)

	// First pass: collect all possible fragment targets
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
			// Convert heading text to fragment identifier
			fragment := headingToFragment(headingText)
			validFragments[fragment] = true
		}
	}

	// Second pass: check link fragments
	linkRegex := regexp.MustCompile(`\[([^\]]*)\]\(([^)]*)\)`)

	for i, line := range params.Lines {
		lineNumber := i + 1

		// Find all links
		matches := linkRegex.FindAllStringSubmatch(line, -1)
		positions := linkRegex.FindAllStringIndex(line, -1)

		for j, match := range matches {
			linkURL := match[2]
			pos := positions[j]

			// Check if this is a fragment link (starts with #)
			if strings.HasPrefix(linkURL, "#") {
				fragment := linkURL[1:] // Remove the #

				// Skip empty fragments
				if fragment == "" {
					continue
				}

				// Check if fragment exists as a valid heading
				if !validFragments[fragment] {
					violation := value.NewViolation(
						[]string{"MD051", "link-fragments"},
						"Link fragments should be valid",
						nil,
						lineNumber,
					)

					violation = violation.WithErrorDetail("Link fragment does not correspond to any heading: #" + fragment)
					violation = violation.WithErrorContext(match[0])
					violation = violation.WithColumn(pos[0] + 1)
					violation = violation.WithLength(pos[1] - pos[0])

					violations = append(violations, *violation)
				}
			}
		}
	}

	return functional.Ok(violations)
}

// headingToFragment converts a heading text to a URL fragment identifier
// This follows GitHub-style fragment generation rules
func headingToFragment(heading string) string {
	// Convert to lowercase
	fragment := strings.ToLower(heading)

	// Replace spaces with hyphens
	fragment = regexp.MustCompile(`\s+`).ReplaceAllString(fragment, "-")

	// Remove non-alphanumeric characters except hyphens
	fragment = regexp.MustCompile(`[^a-z0-9\-]`).ReplaceAllString(fragment, "")

	// Remove leading/trailing hyphens
	fragment = strings.Trim(fragment, "-")

	// Collapse multiple hyphens
	fragment = regexp.MustCompile(`-+`).ReplaceAllString(fragment, "-")

	return fragment
}
