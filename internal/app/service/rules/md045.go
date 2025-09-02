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

// MD045 - Images should have alternate text (alt text)
func NewMD045Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md045.md")

	return entity.NewRule(
		[]string{"MD045", "no-alt-text"},
		"Images should have alternate text (alt text)",
		[]string{"accessibility", "images"},
		infoURL,
		"commonmark",
		map[string]interface{}{},
		md045Function,
	)
}

func md045Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Regex patterns for different image formats
	markdownImageRegex := regexp.MustCompile(`!\[([^\]]*)\]\([^)]+\)`)   // ![alt](url)
	referenceImageRegex := regexp.MustCompile(`!\[([^\]]*)\]\[[^\]]*\]`) // ![alt][ref]
	shortcutImageRegex := regexp.MustCompile(`!\[([^\]]+)\]`)            // ![alt]

	// HTML image regex with various patterns
	htmlImageRegex := regexp.MustCompile(`<img[^>]*>`)

	// Process each line
	for i, line := range params.Lines {
		lineNumber := i + 1

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check Markdown images ![alt](url)
		matches := markdownImageRegex.FindAllStringSubmatch(line, -1)
		positions := markdownImageRegex.FindAllStringIndex(line, -1)

		for j, match := range matches {
			altText := strings.TrimSpace(match[1])
			pos := positions[j]

			if altText == "" {
				violation := value.NewViolation(
					[]string{"MD045", "no-alt-text"},
					"Images should have alternate text (alt text)",
					nil,
					lineNumber,
				)

				violation = violation.WithErrorDetail("Image is missing alt text")
				violation = violation.WithErrorContext(match[0])
				violation = violation.WithColumn(pos[0] + 1) // 1-based column
				violation = violation.WithLength(pos[1] - pos[0])

				violations = append(violations, *violation)
			}
		}

		// Check reference images ![alt][ref]
		refMatches := referenceImageRegex.FindAllStringSubmatch(line, -1)
		refPositions := referenceImageRegex.FindAllStringIndex(line, -1)

		for j, match := range refMatches {
			altText := strings.TrimSpace(match[1])
			pos := refPositions[j]

			if altText == "" {
				violation := value.NewViolation(
					[]string{"MD045", "no-alt-text"},
					"Images should have alternate text (alt text)",
					nil,
					lineNumber,
				)

				violation = violation.WithErrorDetail("Reference image is missing alt text")
				violation = violation.WithErrorContext(match[0])
				violation = violation.WithColumn(pos[0] + 1)
				violation = violation.WithLength(pos[1] - pos[0])

				violations = append(violations, *violation)
			}
		}

		// Check shortcut images ![alt]
		shortcutMatches := shortcutImageRegex.FindAllStringSubmatch(line, -1)
		shortcutPositions := shortcutImageRegex.FindAllStringIndex(line, -1)

		for j, match := range shortcutMatches {
			pos := shortcutPositions[j]

			// Skip if this match is part of a markdown or reference image
			matchEnd := pos[1]
			if matchEnd < len(line) {
				nextChars := line[matchEnd:]
				if strings.HasPrefix(nextChars, "(") || strings.HasPrefix(nextChars, "[") {
					continue // This is a markdown or reference image, not a shortcut image
				}
			}

			altText := strings.TrimSpace(match[1])

			if altText == "" {
				violation := value.NewViolation(
					[]string{"MD045", "no-alt-text"},
					"Images should have alternate text (alt text)",
					nil,
					lineNumber,
				)

				violation = violation.WithErrorDetail("Shortcut image is missing alt text")
				violation = violation.WithErrorContext(match[0])
				violation = violation.WithColumn(pos[0] + 1)
				violation = violation.WithLength(pos[1] - pos[0])

				violations = append(violations, *violation)
			}
		}

		// Check HTML images
		htmlMatches := htmlImageRegex.FindAllString(line, -1)
		htmlPositions := htmlImageRegex.FindAllStringIndex(line, -1)

		for j, match := range htmlMatches {
			pos := htmlPositions[j]

			// Check if aria-hidden="true" is present (which exempts from alt text requirement)
			ariaHiddenRegex := regexp.MustCompile(`aria-hidden\s*=\s*["']true["']`)
			if ariaHiddenRegex.MatchString(match) {
				continue // Skip images with aria-hidden="true"
			}

			// Check for alt attribute
			altAttrRegex := regexp.MustCompile(`alt\s*=\s*["']([^"']*)["']`)
			altMatches := altAttrRegex.FindStringSubmatch(match)

			hasAltAttribute := altMatches != nil
			altText := ""
			if hasAltAttribute {
				altText = strings.TrimSpace(altMatches[1])
			}

			if !hasAltAttribute || altText == "" {
				violation := value.NewViolation(
					[]string{"MD045", "no-alt-text"},
					"Images should have alternate text (alt text)",
					nil,
					lineNumber,
				)

				var detail string
				if !hasAltAttribute {
					detail = "HTML image is missing alt attribute"
				} else {
					detail = "HTML image has empty alt attribute"
				}

				violation = violation.WithErrorDetail(detail)
				violation = violation.WithErrorContext(match)
				violation = violation.WithColumn(pos[0] + 1)
				violation = violation.WithLength(pos[1] - pos[0])

				violations = append(violations, *violation)
			}
		}
	}

	return functional.Ok(violations)
}
