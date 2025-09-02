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

// MD059 - Link text should be descriptive
func NewMD059Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md059.md")

	return entity.NewRule(
		[]string{"MD059", "descriptive-link-text"},
		"Link text should be descriptive",
		[]string{"accessibility", "links"},
		infoURL,
		"commonmark",
		map[string]interface{}{
			"prohibited_texts": []interface{}{"click here", "here", "link", "more"}, // Prohibited link texts
		},
		md059Function,
	)
}

func md059Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Get configuration
	prohibitedTexts := getStringSliceConfig(params.Config, "prohibited_texts")
	if len(prohibitedTexts) == 0 {
		// Default prohibited texts
		prohibitedTexts = []string{"click here", "here", "link", "more"}
	}

	// Create set for fast lookup (case-insensitive)
	prohibitedSet := make(map[string]bool)
	for _, text := range prohibitedTexts {
		prohibitedSet[strings.ToLower(strings.TrimSpace(text))] = true
	}

	// Regex patterns for Markdown links (HTML links are ignored per spec)
	inlineLinkRegex := regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)     // [text](url)
	referenceLinkRegex := regexp.MustCompile(`\[([^\]]+)\]\[[^\]]*\]`) // [text][ref]
	shortcutLinkRegex := regexp.MustCompile(`\[([^\]]+)\]`)            // [text]

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
			linkText := strings.ToLower(strings.TrimSpace(match[1]))
			pos := positions[j]

			if prohibitedSet[linkText] {
				violation := value.NewViolation(
					[]string{"MD059", "descriptive-link-text"},
					"Link text should be descriptive",
					nil,
					lineNumber,
				)

				violation = violation.WithErrorDetail("Link text is not descriptive: '" + match[1] + "'")
				violation = violation.WithErrorContext(match[0])
				violation = violation.WithColumn(pos[0] + 1)
				violation = violation.WithLength(pos[1] - pos[0])

				violations = append(violations, *violation)
			}
		}

		// Check reference links [text][ref]
		refMatches := referenceLinkRegex.FindAllStringSubmatch(line, -1)
		refPositions := referenceLinkRegex.FindAllStringIndex(line, -1)

		for j, match := range refMatches {
			linkText := strings.ToLower(strings.TrimSpace(match[1]))
			pos := refPositions[j]

			if prohibitedSet[linkText] {
				violation := value.NewViolation(
					[]string{"MD059", "descriptive-link-text"},
					"Link text should be descriptive",
					nil,
					lineNumber,
				)

				violation = violation.WithErrorDetail("Reference link text is not descriptive: '" + match[1] + "'")
				violation = violation.WithErrorContext(match[0])
				violation = violation.WithColumn(pos[0] + 1)
				violation = violation.WithLength(pos[1] - pos[0])

				violations = append(violations, *violation)
			}
		}

		// Check shortcut links [text]
		shortcutMatches := shortcutLinkRegex.FindAllStringSubmatch(line, -1)
		shortcutPositions := shortcutLinkRegex.FindAllStringIndex(line, -1)

		for j, match := range shortcutMatches {
			pos := shortcutPositions[j]

			// Skip if this match is part of an inline or reference link
			matchEnd := pos[1]
			if matchEnd < len(line) {
				nextChars := line[matchEnd:]
				if strings.HasPrefix(nextChars, "(") || strings.HasPrefix(nextChars, "[") {
					continue // This is an inline or reference link, not a shortcut link
				}
			}

			linkText := strings.ToLower(strings.TrimSpace(match[1]))

			if prohibitedSet[linkText] {
				violation := value.NewViolation(
					[]string{"MD059", "descriptive-link-text"},
					"Link text should be descriptive",
					nil,
					lineNumber,
				)

				violation = violation.WithErrorDetail("Shortcut link text is not descriptive: '" + match[1] + "'")
				violation = violation.WithErrorContext(match[0])
				violation = violation.WithColumn(pos[0] + 1)
				violation = violation.WithLength(pos[1] - pos[0])

				violations = append(violations, *violation)
			}
		}
	}

	return functional.Ok(violations)
}
