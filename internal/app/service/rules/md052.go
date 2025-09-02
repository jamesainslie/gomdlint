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

// MD052 - Reference links and images should use defined labels
func NewMD052Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md052.md")

	return entity.NewRule(
		[]string{"MD052", "reference-links-images"},
		"Reference links and images should use defined labels",
		[]string{"links", "images"},
		infoURL,
		"commonmark",
		map[string]interface{}{
			"shortcut_syntax": false, // Allow shortcut reference syntax [label] without [label][]
		},
		md052Function,
	)
}

func md052Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Get configuration
	allowShortcut := getBoolConfig(params.Config, "shortcut_syntax", false)

	// First pass: collect all reference definitions
	definedLabels := make(map[string]bool)

	// Regex for reference definitions: [label]: url "title"
	refDefRegex := regexp.MustCompile(`^\s*\[([^\]]+)\]:\s*(.*)$`)

	for _, line := range params.Lines {
		if matches := refDefRegex.FindStringSubmatch(line); matches != nil {
			label := strings.ToLower(strings.TrimSpace(matches[1]))
			definedLabels[label] = true
		}
	}

	// Second pass: check reference links and images
	referenceLinkRegex := regexp.MustCompile(`\[([^\]]*)\]\[([^\]]*)\]`)   // [text][label]
	referenceImageRegex := regexp.MustCompile(`!\[([^\]]*)\]\[([^\]]*)\]`) // ![alt][label]

	// Shortcut reference syntax (if not allowed, we'll check these too)
	shortcutLinkRegex := regexp.MustCompile(`\[([^\]]+)\]`)   // [label] (we'll check manually if not followed by [ ( or 
	shortcutImageRegex := regexp.MustCompile(`!\[([^\]]+)\]`) // ![label] (we'll check manually if not followed by [ or ()

	for i, line := range params.Lines {
		lineNumber := i + 1

		// Skip reference definitions themselves
		if refDefRegex.MatchString(line) {
			continue
		}

		// Check reference links [text][label]
		matches := referenceLinkRegex.FindAllStringSubmatch(line, -1)
		positions := referenceLinkRegex.FindAllStringIndex(line, -1)

		for j, match := range matches {
			text := match[1]
			label := strings.ToLower(strings.TrimSpace(match[2]))
			pos := positions[j]

			// Empty label means use text as label
			if label == "" {
				label = strings.ToLower(strings.TrimSpace(text))
			}

			if !definedLabels[label] {
				violation := value.NewViolation(
					[]string{"MD052", "reference-links-images"},
					"Reference links and images should use defined labels",
					nil,
					lineNumber,
				)

				violation = violation.WithErrorDetail("Reference link label is not defined: " + match[2])
				violation = violation.WithErrorContext(match[0])
				violation = violation.WithColumn(pos[0] + 1)
				violation = violation.WithLength(pos[1] - pos[0])

				violations = append(violations, *violation)
			}
		}

		// Check reference images ![alt][label]
		imgMatches := referenceImageRegex.FindAllStringSubmatch(line, -1)
		imgPositions := referenceImageRegex.FindAllStringIndex(line, -1)

		for j, match := range imgMatches {
			alt := match[1]
			label := strings.ToLower(strings.TrimSpace(match[2]))
			pos := imgPositions[j]

			// Empty label means use alt text as label
			if label == "" {
				label = strings.ToLower(strings.TrimSpace(alt))
			}

			if !definedLabels[label] {
				violation := value.NewViolation(
					[]string{"MD052", "reference-links-images"},
					"Reference links and images should use defined labels",
					nil,
					lineNumber,
				)

				violation = violation.WithErrorDetail("Reference image label is not defined: " + match[2])
				violation = violation.WithErrorContext(match[0])
				violation = violation.WithColumn(pos[0] + 1)
				violation = violation.WithLength(pos[1] - pos[0])

				violations = append(violations, *violation)
			}
		}

		// Check shortcut reference links [label] (if not allowed)
		if !allowShortcut {
			shortcutMatches := shortcutLinkRegex.FindAllStringSubmatch(line, -1)
			shortcutPositions := shortcutLinkRegex.FindAllStringIndex(line, -1)

			for j, match := range shortcutMatches {
				pos := shortcutPositions[j]

				// Check if this is actually a regular markdown link [text](url) or anchor link [text](#anchor)
				if pos[1] < len(line) {
					nextChar := line[pos[1]]
					if nextChar == '(' {
						// This is a regular markdown link [text](url), skip it
						continue
					}
					if nextChar == '[' {
						// This is a reference link [text][label], skip it (handled by referenceLinkRegex)
						continue
					}
				}

				// Check if this is part of a reference definition [label]:
				remainingLine := line[pos[1]:]
				if strings.HasPrefix(remainingLine, ":") {
					// This is a reference definition, skip it
					continue
				}

				label := strings.ToLower(strings.TrimSpace(match[1]))

				// Skip anchor links (internal links starting with #)
				if strings.HasPrefix(match[1], "#") {
					continue
				}

				if !definedLabels[label] {
					violation := value.NewViolation(
						[]string{"MD052", "reference-links-images"},
						"Reference links and images should use defined labels",
						nil,
						lineNumber,
					)

					violation = violation.WithErrorDetail("Shortcut reference link label is not defined: " + match[1])
					violation = violation.WithErrorContext(match[0])
					violation = violation.WithColumn(pos[0] + 1)
					violation = violation.WithLength(pos[1] - pos[0])

					violations = append(violations, *violation)
				}
			}

			// Check shortcut reference images ![label]
			shortcutImgMatches := shortcutImageRegex.FindAllStringSubmatch(line, -1)
			shortcutImgPositions := shortcutImageRegex.FindAllStringIndex(line, -1)

			for j, match := range shortcutImgMatches {
				pos := shortcutImgPositions[j]

				// Check if this is actually an inline image ![alt](url) or reference image ![alt][label]
				// by checking what follows the closing bracket
				if pos[1] < len(line) {
					nextChar := line[pos[1]]
					if nextChar == '[' || nextChar == '(' {
						continue // This is not a shortcut reference, skip it
					}
				}

				label := strings.ToLower(strings.TrimSpace(match[1]))

				if !definedLabels[label] {
					violation := value.NewViolation(
						[]string{"MD052", "reference-links-images"},
						"Reference links and images should use defined labels",
						nil,
						lineNumber,
					)

					violation = violation.WithErrorDetail("Shortcut reference image label is not defined: " + match[1])
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
