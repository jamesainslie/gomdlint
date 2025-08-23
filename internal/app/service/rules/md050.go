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

// MD050 - Strong style
func NewMD050Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md050.md")

	return entity.NewRule(
		[]string{"MD050", "strong-style"},
		"Strong style",
		[]string{"emphasis"},
		infoURL,
		"commonmark",
		map[string]interface{}{
			"style": "consistent", // consistent|asterisk|underscore
		},
		md050Function,
	)
}

type StrongStyle int

const (
	StrongUnknown    StrongStyle = iota
	StrongAsterisk               // **text**
	StrongUnderscore             // __text__
)

func md050Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Get configuration
	styleConfig := getStringConfig(params.Config, "style", "consistent")

	var expectedStyle StrongStyle
	var firstStrongLine int

	// Parse expected style if not consistent
	if styleConfig != "consistent" {
		switch styleConfig {
		case "asterisk":
			expectedStyle = StrongAsterisk
		case "underscore":
			expectedStyle = StrongUnderscore
		}
	}

	// Regex for strong emphasis (double ** or __)
	// Note: Strong emphasis within words is restricted to asterisk
	strongRegex := regexp.MustCompile(`(?:^|[^*_\w])([*_]{2})([^*_\s][^*_]*?)([*_]{2})(?:[^*_\w]|$)`)

	// Process each line
	for i, line := range params.Lines {
		lineNumber := i + 1

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Find all strong emphasis matches
		matches := strongRegex.FindAllStringSubmatch(line, -1)
		positions := strongRegex.FindAllStringIndex(line, -1)

		for j, match := range matches {
			openMarker := match[1]
			closeMarker := match[3]
			content := match[2]

			// Skip mismatched markers
			if openMarker != closeMarker {
				continue
			}

			var currentStyle StrongStyle
			if openMarker == "**" {
				currentStyle = StrongAsterisk
			} else {
				currentStyle = StrongUnderscore
			}

			// Check for strong emphasis within words - only asterisk allowed
			pos := positions[j]
			isWithinWord := false

			if pos[0] > 0 && isWordCharacter(rune(line[pos[0]-1])) {
				isWithinWord = true
			}
			if pos[1] < len(line) && isWordCharacter(rune(line[pos[1]])) {
				isWithinWord = true
			}

			if isWithinWord && currentStyle != StrongAsterisk {
				violation := value.NewViolation(
					[]string{"MD050", "strong-style"},
					"Strong style",
					nil,
					lineNumber,
				)

				violation = violation.WithErrorDetail("Strong emphasis within words must use asterisk")
				violation = violation.WithErrorContext(match[0])
				violation = violation.WithColumn(pos[0] + 1)
				violation = violation.WithLength(pos[1] - pos[0])

				// Add fix information - convert to asterisk
				fixedText := "**" + content + "**"

				fixInfo := value.NewFixInfo().
					WithLineNumber(lineNumber).
					WithEditColumn(pos[0] + 1).
					WithDeleteLength(pos[1] - pos[0]).
					WithReplaceText(fixedText)

				violation = violation.WithFixInfo(*fixInfo)
				violations = append(violations, *violation)
				continue
			}

			// For consistent style, establish expected style from first strong emphasis
			if styleConfig == "consistent" && expectedStyle == StrongUnknown {
				expectedStyle = currentStyle
				firstStrongLine = lineNumber
			}

			// Check for style violations
			if currentStyle != expectedStyle && !isWithinWord {
				violation := value.NewViolation(
					[]string{"MD050", "strong-style"},
					"Strong style",
					nil,
					lineNumber,
				)

				expectedStyleName := getStrongStyleName(expectedStyle)
				actualStyleName := getStrongStyleName(currentStyle)

				detail := fmt.Sprintf("Expected: %s, Actual: %s", expectedStyleName, actualStyleName)
				if styleConfig == "consistent" && firstStrongLine > 0 {
					detail += fmt.Sprintf(" [Expected: %s (based on line %d)]", expectedStyleName, firstStrongLine)
				}

				violation = violation.WithErrorDetail(detail)
				violation = violation.WithErrorContext(match[0])
				violation = violation.WithColumn(pos[0] + 1)
				violation = violation.WithLength(pos[1] - pos[0])

				// Add fix information - convert to expected style
				var newMarker string
				if expectedStyle == StrongAsterisk {
					newMarker = "**"
				} else {
					newMarker = "__"
				}

				fixedText := newMarker + content + newMarker

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

// getStrongStyleName returns a human-readable name for strong style
func getStrongStyleName(style StrongStyle) string {
	switch style {
	case StrongAsterisk:
		return "asterisk"
	case StrongUnderscore:
		return "underscore"
	default:
		return "unknown"
	}
}
