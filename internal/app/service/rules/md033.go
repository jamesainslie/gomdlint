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

// MD033 - Inline HTML
func NewMD033Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md033.md")

	return entity.NewRule(
		[]string{"MD033", "no-inline-html"},
		"Inline HTML",
		[]string{"html"},
		infoURL,
		"commonmark",
		map[string]interface{}{
			"allowed_elements": []interface{}{}, // List of allowed HTML elements
		},
		md033Function,
	)
}

func md033Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Get configuration
	allowedElements := getStringSliceConfig(params.Config, "allowed_elements")

	// Create set for fast lookup (case-insensitive)
	allowedSet := make(map[string]bool)
	for _, element := range allowedElements {
		allowedSet[strings.ToLower(element)] = true
	}

	// Regex patterns for HTML elements
	htmlTagRegex := regexp.MustCompile(`</?([a-zA-Z][a-zA-Z0-9]*)[^>]*>`)
	htmlCommentRegex := regexp.MustCompile(`<!--[^>]*-->`)
	htmlEntityRegex := regexp.MustCompile(`&[a-zA-Z0-9#]+;`)

	// Process each line
	for i, line := range params.Lines {
		lineNumber := i + 1

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Find HTML tags
		matches := htmlTagRegex.FindAllStringSubmatch(line, -1)
		positions := htmlTagRegex.FindAllStringIndex(line, -1)

		for j, match := range matches {
			tagName := strings.ToLower(match[1])
			pos := positions[j]

			// Skip if this element is allowed
			if allowedSet[tagName] {
				continue
			}

			violation := value.NewViolation(
				[]string{"MD033", "no-inline-html"},
				"Inline HTML",
				nil,
				lineNumber,
			)

			violation = violation.WithErrorDetail("HTML element not allowed: " + match[1])
			violation = violation.WithErrorContext(match[0])
			violation = violation.WithColumn(pos[0] + 1)
			violation = violation.WithLength(pos[1] - pos[0])

			violations = append(violations, *violation)
		}

		// Find HTML comments (generally not allowed unless specifically permitted)
		if !allowedSet["comment"] {
			commentMatches := htmlCommentRegex.FindAllString(line, -1)
			commentPositions := htmlCommentRegex.FindAllStringIndex(line, -1)

			for j, match := range commentMatches {
				pos := commentPositions[j]

				violation := value.NewViolation(
					[]string{"MD033", "no-inline-html"},
					"Inline HTML",
					nil,
					lineNumber,
				)

				violation = violation.WithErrorDetail("HTML comment not allowed")
				violation = violation.WithErrorContext(match)
				violation = violation.WithColumn(pos[0] + 1)
				violation = violation.WithLength(pos[1] - pos[0])

				violations = append(violations, *violation)
			}
		}

		// Find HTML entities (generally not allowed unless specifically permitted)
		if !allowedSet["entity"] {
			entityMatches := htmlEntityRegex.FindAllString(line, -1)
			entityPositions := htmlEntityRegex.FindAllStringIndex(line, -1)

			for j, match := range entityMatches {
				pos := entityPositions[j]

				violation := value.NewViolation(
					[]string{"MD033", "no-inline-html"},
					"Inline HTML",
					nil,
					lineNumber,
				)

				violation = violation.WithErrorDetail("HTML entity not allowed")
				violation = violation.WithErrorContext(match)
				violation = violation.WithColumn(pos[0] + 1)
				violation = violation.WithLength(pos[1] - pos[0])

				violations = append(violations, *violation)
			}
		}
	}

	return functional.Ok(violations)
}
