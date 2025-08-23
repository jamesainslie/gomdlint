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

// MD004 - Unordered list style
func NewMD004Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md004.md")

	return entity.NewRule(
		[]string{"MD004", "ul-style"},
		"Unordered list style",
		[]string{"bullet", "ul"},
		infoURL,
		"commonmark",
		map[string]interface{}{
			"style": "consistent", // consistent|asterisk|plus|dash|sublist
		},
		md004Function,
	)
}

type BulletStyle int

const (
	BulletUnknown  BulletStyle = iota
	BulletAsterisk             // *
	BulletPlus                 // +
	BulletDash                 // -
)

func md004Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Get configuration
	styleConfig := getStringConfig(params.Config, "style", "consistent")

	// Compile regex for unordered list items
	ulRegex := regexp.MustCompile(`^(\s*)([-*+])(\s+)(.*)$`)

	var expectedStyle BulletStyle
	var firstListLine int

	// For sublist mode, track style by indentation level
	levelStyles := make(map[int]BulletStyle)

	// Parse expected style if not consistent or sublist
	if styleConfig != "consistent" && styleConfig != "sublist" {
		switch styleConfig {
		case "asterisk":
			expectedStyle = BulletAsterisk
		case "plus":
			expectedStyle = BulletPlus
		case "dash":
			expectedStyle = BulletDash
		}
	}

	// Process each line
	for i, line := range params.Lines {
		lineNumber := i + 1

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check if this is an unordered list item
		matches := ulRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		indent := len(matches[1])
		marker := matches[2]

		// Determine current bullet style
		var currentStyle BulletStyle
		switch marker {
		case "*":
			currentStyle = BulletAsterisk
		case "+":
			currentStyle = BulletPlus
		case "-":
			currentStyle = BulletDash
		}

		// Handle different style configurations
		var violation *value.Violation

		switch styleConfig {
		case "consistent":
			// All bullets should be the same
			if expectedStyle == BulletUnknown {
				expectedStyle = currentStyle
				firstListLine = lineNumber
			} else if currentStyle != expectedStyle {
				violation = createStyleViolation(lineNumber, line, currentStyle, expectedStyle, styleConfig, firstListLine)
			}

		case "sublist":
			// Each indentation level should be consistent, different from parent
			if existingStyle, exists := levelStyles[indent]; exists {
				// This indentation level has been seen before
				if currentStyle != existingStyle {
					violation = createStyleViolation(lineNumber, line, currentStyle, existingStyle, styleConfig, 0)
				}
			} else {
				// New indentation level - check it's different from parent
				levelStyles[indent] = currentStyle

				// Find parent level (highest indent < current)
				parentIndent := -1
				for lvl := range levelStyles {
					if lvl < indent && lvl > parentIndent {
						parentIndent = lvl
					}
				}

				if parentIndent >= 0 {
					parentStyle := levelStyles[parentIndent]
					if currentStyle == parentStyle {
						violation = &value.Violation{}
						*violation = *value.NewViolation(
							[]string{"MD004", "ul-style"},
							"Unordered list style",
							nil,
							lineNumber,
						)
						violation = violation.WithErrorDetail("Sublist style should differ from parent list")
						violation = violation.WithErrorContext(strings.TrimSpace(line))
					}
				}
			}

		default:
			// Specific style required
			if currentStyle != expectedStyle {
				violation = createStyleViolation(lineNumber, line, currentStyle, expectedStyle, styleConfig, 0)
			}
		}

		if violation != nil {
			violations = append(violations, *violation)
		}
	}

	return functional.Ok(violations)
}

// createStyleViolation creates a style violation with appropriate details
func createStyleViolation(lineNumber int, line string, actual, expected BulletStyle, styleConfig string, firstLine int) *value.Violation {
	violation := value.NewViolation(
		[]string{"MD004", "ul-style"},
		"Unordered list style",
		nil,
		lineNumber,
	)

	actualName := getBulletStyleName(actual)
	expectedName := getBulletStyleName(expected)

	detail := fmt.Sprintf("Expected: %s, Actual: %s", expectedName, actualName)
	if styleConfig == "consistent" && firstLine > 0 {
		detail += fmt.Sprintf(" [Expected: %s (based on line %d)]", expectedName, firstLine)
	}

	violation = violation.WithErrorDetail(detail)
	violation = violation.WithErrorContext(strings.TrimSpace(line))

	// Add fix information - replace the bullet character
	ulRegex := regexp.MustCompile(`^(\s*)([-*+])(\s+)(.*)$`)
	if matches := ulRegex.FindStringSubmatch(line); matches != nil {
		markerPos := len(matches[1]) + 1 // Position of the marker (1-based)
		newMarker := getBulletChar(expected)

		fixInfo := value.NewFixInfo().
			WithLineNumber(lineNumber).
			WithEditColumn(markerPos).
			WithDeleteLength(1).
			WithReplaceText(newMarker)

		violation = violation.WithFixInfo(*fixInfo)
	}

	return violation
}

// getBulletStyleName returns a human-readable name for a bullet style
func getBulletStyleName(style BulletStyle) string {
	switch style {
	case BulletAsterisk:
		return "asterisk"
	case BulletPlus:
		return "plus"
	case BulletDash:
		return "dash"
	default:
		return "unknown"
	}
}

// getBulletChar returns the character for a bullet style
func getBulletChar(style BulletStyle) string {
	switch style {
	case BulletAsterisk:
		return "*"
	case BulletPlus:
		return "+"
	case BulletDash:
		return "-"
	default:
		return "*"
	}
}
