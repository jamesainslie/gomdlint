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

// MD041 - First line in a file should be a top-level heading
func NewMD041Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md041.md")

	return entity.NewRule(
		[]string{"MD041", "first-line-h1", "first-line-heading"},
		"First line in a file should be a top-level heading",
		[]string{"headings"},
		infoURL,
		"commonmark",
		map[string]interface{}{
			"level":              1,                  // Heading level to require
			"front_matter_title": `^\s*title\s*[:=]`, // RegExp for front matter title
			"allow_preamble":     false,              // Allow content before first heading
		},
		md041Function,
	)
}

func md041Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Get configuration
	requiredLevel := getIntConfig(params.Config, "level", 1)
	frontMatterTitleRegex := getStringConfig(params.Config, "front_matter_title", `^\s*title\s*[:=]`)
	allowPreamble := getBoolConfig(params.Config, "allow_preamble", false)

	if len(params.Lines) == 0 {
		return functional.Ok(violations)
	}

	// Compile front matter regex
	var titleRegex *regexp.Regexp
	if frontMatterTitleRegex != "" {
		titleRegex = regexp.MustCompile(`(?i)` + frontMatterTitleRegex)
	}

	// Check for front matter title first
	hasFrontMatterTitle := false
	inFrontMatter := false
	frontMatterDelimiter := ""
	contentStartLine := 0

	for i, line := range params.Lines {
		lineNumber := i + 1
		trimmed := strings.TrimSpace(line)

		// Detect front matter boundaries
		if lineNumber == 1 && (trimmed == "---" || trimmed == "+++") {
			inFrontMatter = true
			frontMatterDelimiter = trimmed
			continue
		}

		if inFrontMatter && trimmed == frontMatterDelimiter {
			inFrontMatter = false
			contentStartLine = i + 1 // Next line after front matter
			break
		}

		if inFrontMatter && titleRegex != nil && titleRegex.MatchString(line) {
			hasFrontMatterTitle = true
		}

		if !inFrontMatter {
			contentStartLine = i
			break
		}
	}

	// If there's a front matter title, no heading is required
	if hasFrontMatterTitle {
		return functional.Ok(violations)
	}

	// Find first non-empty line after front matter
	var firstContentLine string
	var firstContentLineNumber int

	for i := contentStartLine; i < len(params.Lines); i++ {
		line := params.Lines[i]
		if strings.TrimSpace(line) != "" {
			firstContentLine = line
			firstContentLineNumber = i + 1
			break
		}
	}

	if firstContentLine == "" {
		return functional.Ok(violations) // Empty file
	}

	// Check if first content line is a heading of required level
	atxRegex := regexp.MustCompile(`^(\s*)(#{1,6})(\s+)(.*)$`)
	setextRegex := regexp.MustCompile(`^(=+|-+)\s*$`)
	htmlHeadingRegex := regexp.MustCompile(`^<h[1-6][^>]*>`)

	isValidHeading := false

	// Check ATX heading
	if matches := atxRegex.FindStringSubmatch(firstContentLine); matches != nil {
		level := len(matches[2])
		if level == requiredLevel {
			isValidHeading = true
		}
	}

	// Check Setext heading (only for levels 1 and 2)
	if !isValidHeading && requiredLevel <= 2 && firstContentLineNumber < len(params.Lines) {
		nextLineIndex := firstContentLineNumber // 0-based index of next line
		if nextLineIndex < len(params.Lines) {
			nextLine := params.Lines[nextLineIndex]
			if setextRegex.MatchString(nextLine) {
				trimmed := strings.TrimSpace(nextLine)
				level := 1
				if strings.HasPrefix(trimmed, "-") {
					level = 2
				}
				if level == requiredLevel {
					isValidHeading = true
				}
			}
		}
	}

	// Check HTML heading
	if !isValidHeading && htmlHeadingRegex.MatchString(firstContentLine) {
		// Extract level from HTML tag
		htmlLevelRegex := regexp.MustCompile(`<h([1-6])`)
		if matches := htmlLevelRegex.FindStringSubmatch(firstContentLine); matches != nil {
			level := int(matches[1][0] - '0') // Convert char to int
			if level == requiredLevel {
				isValidHeading = true
			}
		}
	}

	// Handle preamble allowance
	if !isValidHeading && allowPreamble {
		// Look for heading after preamble content
		for i := firstContentLineNumber; i < len(params.Lines); i++ {
			line := params.Lines[i]
			if matches := atxRegex.FindStringSubmatch(line); matches != nil {
				level := len(matches[2])
				if level == requiredLevel {
					isValidHeading = true
					break
				}
			}

			// Check setext after this line
			if i+1 < len(params.Lines) {
				nextLine := params.Lines[i+1]
				if setextRegex.MatchString(nextLine) {
					trimmed := strings.TrimSpace(nextLine)
					level := 1
					if strings.HasPrefix(trimmed, "-") {
						level = 2
					}
					if level == requiredLevel {
						isValidHeading = true
						break
					}
				}
			}
		}
	}

	if !isValidHeading {
		violation := value.NewViolation(
			[]string{"MD041", "first-line-h1", "first-line-heading"},
			"First line in a file should be a top-level heading",
			nil,
			firstContentLineNumber,
		)

		levelName := "h" + string(rune('0'+requiredLevel))
		violation = violation.WithErrorDetail("First line should be a " + levelName + " heading")
		violation = violation.WithErrorContext(strings.TrimSpace(firstContentLine))

		violations = append(violations, *violation)
	}

	return functional.Ok(violations)
}
