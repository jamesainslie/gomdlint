package rules

import (
	"context"
	"net/url"
	"regexp"

	"github.com/gomdlint/gomdlint/internal/domain/entity"
	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// MD054 - Link and image style
func NewMD054Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md054.md")

	return entity.NewRule(
		[]string{"MD054", "link-image-style"},
		"Link and image style",
		[]string{"images", "links"},
		infoURL,
		"commonmark",
		map[string]interface{}{
			"autolink":       true, // Allow autolinks
			"inline":         true, // Allow inline links
			"full_reference": true, // Allow full reference links [text][label]
			"collapsed":      true, // Allow collapsed reference links [text][]
			"shortcut":       true, // Allow shortcut reference links [text]
			"url_inline":     true, // Allow inline URLs in angle brackets
		},
		md054Function,
	)
}

func md054Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Get configuration
	allowAutolink := getBoolConfig(params.Config, "autolink", true)
	allowInline := getBoolConfig(params.Config, "inline", true)
	allowFullReference := getBoolConfig(params.Config, "full_reference", true)
	allowCollapsed := getBoolConfig(params.Config, "collapsed", true)
	allowShortcut := getBoolConfig(params.Config, "shortcut", true)

	// Regex patterns for different link/image styles
	autolinkRegex := regexp.MustCompile(`<(https?://[^>\s]+|[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})>`)
	inlineLinkRegex := regexp.MustCompile(`\[([^\]]*)\]\(([^)]+)\)`)
	inlineImageRegex := regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	fullReferenceLinkRegex := regexp.MustCompile(`\[([^\]]*)\]\[([^\]]+)\]`)
	fullReferenceImageRegex := regexp.MustCompile(`!\[([^\]]*)\]\[([^\]]+)\]`)
	collapsedReferenceLinkRegex := regexp.MustCompile(`\[([^\]]+)\]\[\s*\]`)
	collapsedReferenceImageRegex := regexp.MustCompile(`!\[([^\]]+)\]\[\s*\]`)
	shortcutLinkRegex := regexp.MustCompile(`\[([^\]]+)\](?!\[|\(|:)`)
	shortcutImageRegex := regexp.MustCompile(`!\[([^\]]+)\](?!\[|\()`)

	// Process each line
	for i, line := range params.Lines {
		lineNumber := i + 1

		// Skip reference definitions
		if regexp.MustCompile(`^\s*\[[^\]]+\]:\s*`).MatchString(line) {
			continue
		}

		// Check autolinks
		if !allowAutolink {
			matches := autolinkRegex.FindAllStringSubmatch(line, -1)
			positions := autolinkRegex.FindAllStringIndex(line, -1)

			for j, match := range matches {
				pos := positions[j]

				violation := value.NewViolation(
					[]string{"MD054", "link-image-style"},
					"Link and image style",
					nil,
					lineNumber,
				)

				violation = violation.WithErrorDetail("Autolink style not allowed")
				violation = violation.WithErrorContext(match[0])
				violation = violation.WithColumn(pos[0] + 1)
				violation = violation.WithLength(pos[1] - pos[0])

				violations = append(violations, *violation)
			}
		}

		// Check inline links
		if !allowInline {
			violations = append(violations, checkInlineStyle(line, lineNumber, inlineLinkRegex, "Inline link style not allowed")...)
			violations = append(violations, checkInlineStyle(line, lineNumber, inlineImageRegex, "Inline image style not allowed")...)
		}

		// Check full reference links
		if !allowFullReference {
			violations = append(violations, checkInlineStyle(line, lineNumber, fullReferenceLinkRegex, "Full reference link style not allowed")...)
			violations = append(violations, checkInlineStyle(line, lineNumber, fullReferenceImageRegex, "Full reference image style not allowed")...)
		}

		// Check collapsed reference links
		if !allowCollapsed {
			violations = append(violations, checkInlineStyle(line, lineNumber, collapsedReferenceLinkRegex, "Collapsed reference link style not allowed")...)
			violations = append(violations, checkInlineStyle(line, lineNumber, collapsedReferenceImageRegex, "Collapsed reference image style not allowed")...)
		}

		// Check shortcut reference links
		if !allowShortcut {
			violations = append(violations, checkInlineStyle(line, lineNumber, shortcutLinkRegex, "Shortcut reference link style not allowed")...)
			violations = append(violations, checkInlineStyle(line, lineNumber, shortcutImageRegex, "Shortcut reference image style not allowed")...)
		}
	}

	return functional.Ok(violations)
}

// checkInlineStyle is a helper function to check for disallowed styles
func checkInlineStyle(line string, lineNumber int, regex *regexp.Regexp, errorMsg string) []value.Violation {
	var violations []value.Violation

	matches := regex.FindAllStringSubmatch(line, -1)
	positions := regex.FindAllStringIndex(line, -1)

	for j, match := range matches {
		pos := positions[j]

		violation := value.NewViolation(
			[]string{"MD054", "link-image-style"},
			"Link and image style",
			nil,
			lineNumber,
		)

		violation = violation.WithErrorDetail(errorMsg)
		violation = violation.WithErrorContext(match[0])
		violation = violation.WithColumn(pos[0] + 1)
		violation = violation.WithLength(pos[1] - pos[0])

		violations = append(violations, *violation)
	}

	return violations
}
