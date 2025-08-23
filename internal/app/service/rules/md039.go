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

// MD039 - Spaces inside link text
func NewMD039Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md039.md")

	return entity.NewRule(
		[]string{"MD039", "no-space-in-links"},
		"Spaces inside link text",
		[]string{"links", "whitespace"},
		infoURL,
		"commonmark",
		map[string]interface{}{},
		md039Function,
	)
}

func md039Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Regex patterns for different link types with spaces in text
	inlineLinkSpaceRegex := regexp.MustCompile(`\[(\s+)([^\]]*?)(\s+)\]\([^)]*\)`)     // [ text ](url)
	referenceLinkSpaceRegex := regexp.MustCompile(`\[(\s+)([^\]]*?)(\s+)\]\[[^\]]*\]`) // [ text ][ref]
	shortcutLinkSpaceRegex := regexp.MustCompile(`\[(\s+)([^\]]*?)(\s+)\](?!\[|\()`)   // [ text ]

	// Process each line
	for i, line := range params.Lines {
		lineNumber := i + 1

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check inline links [text](url)
		matches := inlineLinkSpaceRegex.FindAllStringSubmatch(line, -1)
		positions := inlineLinkSpaceRegex.FindAllStringIndex(line, -1)

		for j, match := range matches {
			leadingSpace := match[1]
			linkText := match[2]
			trailingSpace := match[3]
			pos := positions[j]

			if len(leadingSpace) > 0 || len(trailingSpace) > 0 {
				violation := value.NewViolation(
					[]string{"MD039", "no-space-in-links"},
					"Spaces inside link text",
					nil,
					lineNumber,
				)

				violation = violation.WithErrorDetail("Spaces found inside link text")
				violation = violation.WithErrorContext(match[0])
				violation = violation.WithColumn(pos[0] + 1)
				violation = violation.WithLength(pos[1] - pos[0])

				// Add fix information - remove spaces around link text
				// Extract the URL part
				fullMatch := line[pos[0]:pos[1]]
				urlStart := strings.Index(fullMatch, "](") + 2
				urlEnd := strings.LastIndex(fullMatch, ")")
				url := fullMatch[urlStart:urlEnd]

				fixedText := "[" + strings.TrimSpace(linkText) + "](" + url + ")"

				fixInfo := value.NewFixInfo().
					WithLineNumber(lineNumber).
					WithEditColumn(pos[0] + 1).
					WithDeleteLength(pos[1] - pos[0]).
					WithReplaceText(fixedText)

				violation = violation.WithFixInfo(*fixInfo)
				violations = append(violations, *violation)
			}
		}

		// Check reference links [text][ref]
		refMatches := referenceLinkSpaceRegex.FindAllStringSubmatch(line, -1)
		refPositions := referenceLinkSpaceRegex.FindAllStringIndex(line, -1)

		for j, match := range refMatches {
			leadingSpace := match[1]
			linkText := match[2]
			trailingSpace := match[3]
			pos := refPositions[j]

			if len(leadingSpace) > 0 || len(trailingSpace) > 0 {
				violation := value.NewViolation(
					[]string{"MD039", "no-space-in-links"},
					"Spaces inside link text",
					nil,
					lineNumber,
				)

				violation = violation.WithErrorDetail("Spaces found inside reference link text")
				violation = violation.WithErrorContext(match[0])
				violation = violation.WithColumn(pos[0] + 1)
				violation = violation.WithLength(pos[1] - pos[0])

				// Add fix information - remove spaces around link text
				// Extract the reference part
				fullMatch := line[pos[0]:pos[1]]
				refStart := strings.Index(fullMatch, "][") + 2
				refEnd := strings.LastIndex(fullMatch, "]")
				ref := fullMatch[refStart:refEnd]

				fixedText := "[" + strings.TrimSpace(linkText) + "][" + ref + "]"

				fixInfo := value.NewFixInfo().
					WithLineNumber(lineNumber).
					WithEditColumn(pos[0] + 1).
					WithDeleteLength(pos[1] - pos[0]).
					WithReplaceText(fixedText)

				violation = violation.WithFixInfo(*fixInfo)
				violations = append(violations, *violation)
			}
		}

		// Check shortcut links [text]
		shortcutMatches := shortcutLinkSpaceRegex.FindAllStringSubmatch(line, -1)
		shortcutPositions := shortcutLinkSpaceRegex.FindAllStringIndex(line, -1)

		for j, match := range shortcutMatches {
			leadingSpace := match[1]
			linkText := match[2]
			trailingSpace := match[3]
			pos := shortcutPositions[j]

			if len(leadingSpace) > 0 || len(trailingSpace) > 0 {
				violation := value.NewViolation(
					[]string{"MD039", "no-space-in-links"},
					"Spaces inside link text",
					nil,
					lineNumber,
				)

				violation = violation.WithErrorDetail("Spaces found inside shortcut link text")
				violation = violation.WithErrorContext(match[0])
				violation = violation.WithColumn(pos[0] + 1)
				violation = violation.WithLength(pos[1] - pos[0])

				// Add fix information - remove spaces around link text
				fixedText := "[" + strings.TrimSpace(linkText) + "]"

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
