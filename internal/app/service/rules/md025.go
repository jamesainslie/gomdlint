package rules

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/gomdlint/gomdlint/internal/domain/entity"
	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// MD025 - Multiple top-level headings in the same document
func NewMD025Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md025.md")

	return entity.NewRule(
		[]string{"MD025", "single-h1", "single-title"},
		"Multiple top-level headings in the same document",
		[]string{"headings"},
		infoURL,
		"commonmark",
		map[string]interface{}{
			"level":              1,                  // Heading level to check (default: 1)
			"front_matter_title": `^\s*title\s*[:=]`, // RegExp for matching title in front matter
		},
		md025Function,
	)
}

func md025Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Get configuration
	targetLevel := getIntConfig(params.Config, "level", 1)
	frontMatterTitleRegex := getStringConfig(params.Config, "front_matter_title", `^\s*title\s*[:=]`)

	// Compile regex for front matter title detection
	var titleRegex *regexp.Regexp
	if frontMatterTitleRegex != "" {
		titleRegex = regexp.MustCompile(`(?i)` + frontMatterTitleRegex) // Case insensitive
	}

	// ATX heading regex
	atxRegex := regexp.MustCompile(`^(\s*)(#{1,6})(\s+)(.*)$`)
	// Setext heading regex (underline)
	setextRegex := regexp.MustCompile(`^(=+|-+)\s*$`)

	var topLevelHeadings []int
	hasFrontMatterTitle := false
	inFrontMatter := false
	frontMatterDelimiter := ""

	// First pass: check for front matter title
	if titleRegex != nil {
		for i, line := range params.Lines {
			lineNumber := i + 1
			trimmedLine := strings.TrimSpace(line)

			// Detect front matter boundaries
			if lineNumber == 1 && (trimmedLine == "---" || trimmedLine == "+++") {
				inFrontMatter = true
				frontMatterDelimiter = trimmedLine
				continue
			}

			if inFrontMatter && trimmedLine == frontMatterDelimiter {
				inFrontMatter = false
				break
			}

			if inFrontMatter && titleRegex.MatchString(line) {
				hasFrontMatterTitle = true
				break
			}
		}
	}

	// Second pass: find all top-level headings
	for i, line := range params.Lines {
		lineNumber := i + 1
		trimmedLine := strings.TrimSpace(line)

		if trimmedLine == "" {
			continue
		}

		var headingLevel int
		var isHeading bool

		// Check for ATX headings
		if matches := atxRegex.FindStringSubmatch(line); matches != nil {
			headingLevel = len(matches[2])
			isHeading = true
		}

		// Check for Setext headings (only if not already found ATX)
		if !isHeading && i > 0 && setextRegex.MatchString(line) {
			prevLine := strings.TrimSpace(params.Lines[i-1])
			if prevLine != "" {
				headingLevel = 1 // = underline is h1
				if strings.HasPrefix(trimmedLine, "-") {
					headingLevel = 2 // - underline is h2
				}
				isHeading = true
			}
		}

		// Check if this is a top-level heading
		if isHeading && headingLevel == targetLevel {
			topLevelHeadings = append(topLevelHeadings, lineNumber)
		}
	}

	// Determine how many top-level headings are allowed
	allowedCount := 1
	if hasFrontMatterTitle {
		allowedCount = 0 // No top-level headings allowed if front matter has title
	}

	// Check for violations
	if len(topLevelHeadings) > allowedCount {
		// Report violations for all headings after the first allowed one
		startIndex := allowedCount
		if hasFrontMatterTitle {
			startIndex = 0 // Report all if front matter title exists
		}

		for i := startIndex; i < len(topLevelHeadings); i++ {
			lineNumber := topLevelHeadings[i]

			violation := value.NewViolation(
				[]string{"MD025", "single-h1", "single-title"},
				"Multiple top-level headings in the same document",
				nil,
				lineNumber,
			)

			var detail string
			if hasFrontMatterTitle {
				detail = "Document has front matter title, no top-level headings should be used"
			} else {
				detail = fmt.Sprintf("Expected: 1 h%d, found multiple", targetLevel)
			}

			violation = violation.WithErrorDetail(detail)
			violation = violation.WithErrorContext(strings.TrimSpace(params.Lines[lineNumber-1]))

			violations = append(violations, *violation)
		}
	}

	return functional.Ok(violations)
}
