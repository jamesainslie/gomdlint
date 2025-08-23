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

// MD036 - Emphasis used instead of a heading
func NewMD036Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md036.md")

	return entity.NewRule(
		[]string{"MD036", "no-emphasis-as-heading"},
		"Emphasis used instead of a heading",
		[]string{"emphasis", "headings"},
		infoURL,
		"commonmark",
		map[string]interface{}{
			"punctuation": ".,;:!?。，；：！？", // Punctuation that suggests it's not a heading
		},
		md036Function,
	)
}

func md036Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Get configuration
	punctuation := getStringConfig(params.Config, "punctuation", ".,;:!?。，；：！？")

	// Create set of punctuation characters
	punctuationSet := make(map[rune]bool)
	for _, char := range punctuation {
		punctuationSet[char] = true
	}

	// Regex for emphasized text that might be used as heading
	// Matches lines that are only emphasis (strong or regular) and nothing else
	strongOnlyRegex := regexp.MustCompile(`^\s*(\*\*|__)([^*_]+)(\*\*|__)\s*$`)
	emphasisOnlyRegex := regexp.MustCompile(`^\s*([*_])([^*_]+)([*_])\s*$`)

	// Process each line
	for i, line := range params.Lines {
		lineNumber := i + 1

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		var isViolation bool
		var emphasisText string
		var matches []string

		// Check for strong emphasis used as heading
		if matches = strongOnlyRegex.FindStringSubmatch(line); matches != nil {
			emphasisText = matches[2]
			isViolation = true
		}

		// Check for regular emphasis used as heading (less common)
		if !isViolation {
			if matches = emphasisOnlyRegex.FindStringSubmatch(line); matches != nil {
				emphasisText = matches[2]
				isViolation = true
			}
		}

		if isViolation {
			// Check if this looks like a heading vs. regular emphasis

			// Skip if text contains punctuation (likely not a heading)
			endsWithPunctuation := false
			if len(emphasisText) > 0 {
				lastChar := rune(emphasisText[len(emphasisText)-1])
				endsWithPunctuation = punctuationSet[lastChar]
			}

			if endsWithPunctuation {
				continue // Likely regular emphasis, not heading-like
			}

			// Skip if text is very long (unlikely to be heading)
			if len(emphasisText) > 100 {
				continue
			}

			// Skip if followed by non-blank content (part of paragraph)
			if i < len(params.Lines)-1 {
				nextLine := strings.TrimSpace(params.Lines[i+1])
				if nextLine != "" && !isHeadingLike(nextLine) {
					continue
				}
			}

			// Skip if preceded by non-blank content (part of paragraph)
			if i > 0 {
				prevLine := strings.TrimSpace(params.Lines[i-1])
				if prevLine != "" && !isHeadingLike(prevLine) {
					continue
				}
			}

			// This looks like emphasis used as a heading
			violation := value.NewViolation(
				[]string{"MD036", "no-emphasis-as-heading"},
				"Emphasis used instead of a heading",
				nil,
				lineNumber,
			)

			violation = violation.WithErrorDetail("Consider using a heading instead of emphasis")
			violation = violation.WithErrorContext(strings.TrimSpace(line))

			violations = append(violations, *violation)
		}
	}

	return functional.Ok(violations)
}

// isHeadingLike checks if a line looks like it could be a heading or heading-related
func isHeadingLike(line string) bool {
	// Check for ATX headings
	if strings.HasPrefix(strings.TrimSpace(line), "#") {
		return true
	}

	// Check for setext underlines
	trimmed := strings.TrimSpace(line)
	if regexp.MustCompile(`^=+$`).MatchString(trimmed) || regexp.MustCompile(`^-+$`).MatchString(trimmed) {
		return true
	}

	return false
}
