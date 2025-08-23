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

// MD003 - Heading style
func NewMD003Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md003.md")

	return entity.NewRule(
		[]string{"MD003", "heading-style"},
		"Heading style",
		[]string{"headings"},
		infoURL,
		"commonmark",
		map[string]interface{}{
			"style": "consistent", // consistent|atx|atx_closed|setext|setext_with_atx|setext_with_atx_closed
		},
		md003Function,
	)
}

type HeadingStyle int

const (
	StyleUnknown HeadingStyle = iota
	StyleATX
	StyleATXClosed
	StyleSetext
)

func md003Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Get configuration
	styleConfig := getStringConfig(params.Config, "style", "consistent")

	// Compile regexes for heading detection
	atxRegex := regexp.MustCompile(`^(\s*)(#{1,6})(\s+)([^#]*?)(#*)\s*$`)
	atxClosedRegex := regexp.MustCompile(`^(\s*)(#{1,6})(\s+)([^#]*?)(\s+)(#{1,6})\s*$`)
	setextRegex := regexp.MustCompile(`^(=+|-+)\s*$`)

	var expectedStyle HeadingStyle
	var firstHeadingLine int

	// Parse expected style
	if styleConfig != "consistent" {
		switch styleConfig {
		case "atx":
			expectedStyle = StyleATX
		case "atx_closed":
			expectedStyle = StyleATXClosed
		case "setext", "setext_with_atx", "setext_with_atx_closed":
			expectedStyle = StyleSetext
		}
	}

	// Process each line to find headings
	for i, line := range params.Lines {
		lineNumber := i + 1
		trimmedLine := strings.TrimSpace(line)

		if trimmedLine == "" {
			continue
		}

		var currentStyle HeadingStyle
		var level int
		var isHeading bool

		// Check for ATX headings (# ## ###)
		if matches := atxRegex.FindStringSubmatch(line); matches != nil {
			level = len(matches[2])

			// Check if it's closed ATX (ends with #)
			if len(strings.TrimSpace(matches[5])) > 0 {
				// Verify proper closed ATX format
				if atxClosedRegex.MatchString(line) {
					currentStyle = StyleATXClosed
				} else {
					currentStyle = StyleATX // Malformed closed ATX, treat as regular ATX
				}
			} else {
				currentStyle = StyleATX
			}
			isHeading = true
		}

		// Check for Setext headings (underlined)
		if !isHeading && i > 0 && setextRegex.MatchString(line) {
			prevLine := strings.TrimSpace(params.Lines[i-1])
			if prevLine != "" {
				currentStyle = StyleSetext
				level = 1
				if line[0] == '-' {
					level = 2
				}
				isHeading = true
				lineNumber = i // Point to the underline for consistency
			}
		}

		if !isHeading {
			continue
		}

		// For consistent style, establish the expected style from first heading
		if styleConfig == "consistent" && expectedStyle == StyleUnknown {
			expectedStyle = currentStyle
			firstHeadingLine = lineNumber
		}

		// Check for style violations
		if !isStyleAllowed(currentStyle, expectedStyle, styleConfig, level) {
			violation := value.NewViolation(
				[]string{"MD003", "heading-style"},
				"Heading style",
				nil,
				lineNumber,
			)

			expectedStyleName := getStyleName(expectedStyle, styleConfig)
			actualStyleName := getStyleName(currentStyle, styleConfig)

			detail := fmt.Sprintf("Expected: %s, Actual: %s", expectedStyleName, actualStyleName)
			if styleConfig == "consistent" && firstHeadingLine > 0 {
				detail += fmt.Sprintf(" [Expected: %s (based on line %d)]", expectedStyleName, firstHeadingLine)
			}

			violation = violation.WithErrorDetail(detail)
			violation = violation.WithErrorContext(strings.TrimSpace(line))

			violations = append(violations, *violation)
		}
	}

	return functional.Ok(violations)
}

// isStyleAllowed checks if a heading style is allowed given the configuration
func isStyleAllowed(currentStyle, expectedStyle HeadingStyle, styleConfig string, level int) bool {
	// For setext_with_atx variants, allow ATX for levels 3+
	if styleConfig == "setext_with_atx" || styleConfig == "setext_with_atx_closed" {
		if level >= 3 {
			if styleConfig == "setext_with_atx_closed" {
				return currentStyle == StyleATXClosed
			}
			return currentStyle == StyleATX
		}
		return currentStyle == StyleSetext
	}

	return currentStyle == expectedStyle
}

// getStyleName returns a human-readable name for a heading style
func getStyleName(style HeadingStyle, config string) string {
	switch style {
	case StyleATX:
		return "ATX"
	case StyleATXClosed:
		return "ATX closed"
	case StyleSetext:
		return "setext"
	default:
		return config
	}
}

// getStringConfig safely extracts a string configuration value
func getStringConfig(config map[string]interface{}, key string, defaultValue string) string {
	if value, exists := config[key]; exists {
		if strValue, ok := value.(string); ok {
			return strValue
		}
	}
	return defaultValue
}
