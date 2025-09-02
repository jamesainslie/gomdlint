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

// MD035 - Horizontal rule style
func NewMD035Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md035.md")

	return entity.NewRule(
		[]string{"MD035", "hr-style"},
		"Horizontal rule style",
		[]string{"hr"},
		infoURL,
		"commonmark",
		map[string]interface{}{
			"style": "consistent", // consistent|specific style string
		},
		md035Function,
	)
}

func md035Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Get configuration
	styleConfig := getStringConfig(params.Config, "style", "consistent")

	var expectedStyle string
	var firstHRLine int

	// Regex for horizontal rules
	hrRegex := regexp.MustCompile(`^(\s*)((??:\*\s*){3,})|(??:-\s*){3,})|(??:_\s*){3,}))\s*$`)

	// Process each line
	for i, line := range params.Lines {
		lineNumber := i + 1

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Check if this is a horizontal rule
		matches := hrRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		indent := matches[1]
		hrContent := strings.TrimSpace(matches[2])

		// Normalize the style for comparison
		currentStyle := normalizeHRStyle(hrContent)

		// For consistent style, establish expected style from first HR
		if styleConfig == "consistent" && expectedStyle == "" {
			expectedStyle = currentStyle
			firstHRLine = lineNumber
		} else if styleConfig != "consistent" {
			expectedStyle = styleConfig
		}

		// Check for style violations
		if currentStyle != expectedStyle {
			violation := value.NewViolation(
				[]string{"MD035", "hr-style"},
				"Horizontal rule style",
				nil,
				lineNumber,
			)

			detail := fmt.Sprintf("Expected: %s, Actual: %s", expectedStyle, currentStyle)
			if styleConfig == "consistent" && firstHRLine > 0 {
				detail += fmt.Sprintf(" [Expected: %s (based on line %d)]", expectedStyle, firstHRLine)
			}

			violation = violation.WithErrorDetail(detail)
			violation = violation.WithErrorContext(strings.TrimSpace(line))

			// Add fix information - replace with expected style
			fixedHR := generateHRFromStyle(expectedStyle)
			fixedLine := indent + fixedHR

			fixInfo := value.NewFixInfo().
				WithLineNumber(lineNumber).
				WithEditColumn(1).
				WithDeleteLength(len(line)).
				WithReplaceText(fixedLine)

			violation = violation.WithFixInfo(*fixInfo)
			violations = append(violations, *violation)
		}
	}

	return functional.Ok(violations)
}

// normalizeHRStyle converts an HR to a normalized style string
func normalizeHRStyle(hr string) string {
	hr = strings.TrimSpace(hr)

	// Count characters and spaces
	if strings.Contains(hr, "*") {
		if strings.Contains(hr, " ") {
			return "* * *"
		}
		return "***"
	}

	if strings.Contains(hr, "-") {
		if strings.Contains(hr, " ") {
			return "- - -"
		}
		return "---"
	}

	if strings.Contains(hr, "_") {
		if strings.Contains(hr, " ") {
			return "_ _ _"
		}
		return "___"
	}

	return hr // Fallback
}

// generateHRFromStyle creates an HR from a style description
func generateHRFromStyle(style string) string {
	switch style {
	case "***":
		return "***"
	case "* * *":
		return "* * *"
	case "---":
		return "---"
	case "- - -":
		return "- - -"
	case "___":
		return "___"
	case "_ _ _":
		return "_ _ _"
	default:
		return style // Use as-is if it's a specific pattern
	}
}
