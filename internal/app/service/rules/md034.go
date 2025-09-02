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

// MD034 - Bare URL used
func NewMD034Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md034.md")

	return entity.NewRule(
		[]string{"MD034", "no-bare-urls"},
		"Bare URL used",
		[]string{"links", "url"},
		infoURL,
		"commonmark",
		map[string]interface{}{},
		md034Function,
	)
}

func md034Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Regex patterns
	urlRegex := regexp.MustCompile(`https?/[^\s<>\[\]]+`)
	emailRegex := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)

	// Patterns to exclude
	angleBracketRegex := regexp.MustCompile(`<(https?/[^\s<>]+|[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})>`)
	codeSpanRegex := regexp.MustCompile("`[^`]*`")
	shortcutLinkRegex := regexp.MustCompile(`\[[^\]]+\]`)
	inlineLinkRegex := regexp.MustCompile(`\[[^\]]*\]\([^)]*\)`)

	// Process each line
	for i, line := range params.Lines {
		lineNumber := i + 1

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Find all URLs and emails
		urlMatches := urlRegex.FindAllStringIndex(line, -1)
		emailMatches := emailRegex.FindAllStringIndex(line, -1)

		// Combine all matches
		allMatches := append(urlMatches, emailMatches...)

		// Check each match
		for _, match := range allMatches {
			start := match[0]
			end := match[1]
			matchText := line[start:end]

			// Skip if already in angle brackets
			if isInAngleBrackets(line, start, end, angleBracketRegex) {
				continue
			}

			// Skip if in code span
			if isInCodeSpan(line, start, end, codeSpanRegex) {
				continue
			}

			// Skip if part of a link
			if isInLink(line, start, end, inlineLinkRegex) {
				continue
			}

			// Skip if looks like a shortcut link (but this could be ambiguous)
			if isShortcutLink(line, start, end, shortcutLinkRegex) {
				continue
			}

			// This is a bare URL/email
			violation := value.NewViolation(
				[]string{"MD034", "no-bare-urls"},
				"Bare URL used",
				nil,
				lineNumber,
			)

			var detail string
			if strings.Contains(matchText, "@") {
				detail = "Bare email address found"
			} else {
				detail = "Bare URL found"
			}

			violation = violation.WithErrorDetail(detail)
			violation = violation.WithErrorContext(matchText)
			violation = violation.WithColumn(start + 1) // 1-based column
			violation = violation.WithLength(end - start)

			// Add fix information - wrap in angle brackets
			fixedText := "<" + matchText + ">"

			fixInfo := value.NewFixInfo().
				WithLineNumber(lineNumber).
				WithEditColumn(start + 1).
				WithDeleteLength(end - start).
				WithReplaceText(fixedText)

			violation = violation.WithFixInfo(*fixInfo)
			violations = append(violations, *violation)
		}
	}

	return functional.Ok(violations)
}

// Helper functions to check if URL is already properly formatted

func isInAngleBrackets(line string, start, end int, regex *regexp.Regexp) bool {
	matches := regex.FindAllStringIndex(line, -1)
	for _, match := range matches {
		if start >= match[0]+1 && end <= match[1]-1 { // +1 and -1 to account for < >
			return true
		}
	}
	return false
}

func isInCodeSpan(line string, start, end int, regex *regexp.Regexp) bool {
	matches := regex.FindAllStringIndex(line, -1)
	for _, match := range matches {
		if start >= match[0] && end <= match[1] {
			return true
		}
	}
	return false
}

func isInLink(line string, start, end int, regex *regexp.Regexp) bool {
	matches := regex.FindAllStringIndex(line, -1)
	for _, match := range matches {
		if start >= match[0] && end <= match[1] {
			return true
		}
	}
	return false
}

func isShortcutLink(line string, start, end int, regex *regexp.Regexp) bool {
	// This is more complex - we'd need to check if [url] pattern exists
	// For now, just check if the URL is immediately preceded by [
	if start > 0 && line[start-1] == '[' {
		// Look for closing ]
		if end < len(line) && line[end] == ']' {
			return true
		}
	}
	return false
}
