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

// MD049 - Emphasis style
func NewMD049Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md049.md")

	return entity.NewRule(
		[]string{"MD049", "emphasis-style"},
		"Emphasis style",
		[]string{"emphasis"},
		infoURL,
		"commonmark",
		map[string]interface{}{
			"style": "consistent", // consistent|asterisk|underscore
		},
		md049Function,
	)
}

type EmphasisStyle int

const (
	EmphasisUnknown    EmphasisStyle = iota
	EmphasisAsterisk                 // *text*
	EmphasisUnderscore               // _text_
)

func md049Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Get configuration
	styleConfig := getStringConfig(params.Config, "style", "consistent")

	var expectedStyle EmphasisStyle
	var firstEmphasisLine int

	// Parse expected style if not consistent
	if styleConfig != "consistent" {
		switch styleConfig {
		case "asterisk":
			expectedStyle = EmphasisAsterisk
		case "underscore":
			expectedStyle = EmphasisUnderscore
		}
	}

	// Regex for emphasis (single * or _ - not strong)
	// Note: Emphasis within words is restricted to asterisk
	emphasisRegex := regexp.MustCompile(`(?:^|[^*_\w])([*_])([^*_\s][^*_]*?)([*_])(?:[^*_\w]|$)`)

	// Process each line
	for i, line := range params.Lines {
		lineNumber := i + 1

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Find all emphasis matches
		matches := emphasisRegex.FindAllStringSubmatch(line, -1)
		positions := emphasisRegex.FindAllStringIndex(line, -1)

		for j, match := range matches {
			// Skip if this is strong emphasis (** or __)
			if len(match[1]) > 1 || len(match[3]) > 1 {
				continue
			}

			openMarker := match[1]
			closeMarker := match[3]
			content := match[2]

			// Skip mismatched markers
			if openMarker != closeMarker {
				continue
			}

			var currentStyle EmphasisStyle
			if openMarker == "*" {
				currentStyle = EmphasisAsterisk
			} else {
				currentStyle = EmphasisUnderscore
			}

			// Check for emphasis within words - only asterisk allowed
			pos := positions[j]
			isWithinWord := false

			if pos[0] > 0 && isWordCharacter(rune(line[pos[0]-1])) {
				isWithinWord = true
			}
			if pos[1] < len(line) && isWordCharacter(rune(line[pos[1]])) {
				isWithinWord = true
			}

			if isWithinWord && currentStyle != EmphasisAsterisk {
				violation := value.NewViolation(
					[]string{"MD049", "emphasis-style"},
					"Emphasis style",
					nil,
					lineNumber,
				)

				violation = violation.WithErrorDetail("Emphasis within words must use asterisk")
				violation = violation.WithErrorContext(match[0])
				violation = violation.WithColumn(pos[0] + 1)
				violation = violation.WithLength(pos[1] - pos[0])

				// Add fix information - convert to asterisk
				fixedText := "*" + content + "*"

				fixInfo := value.NewFixInfo().
					WithLineNumber(lineNumber).
					WithEditColumn(pos[0] + 1).
					WithDeleteLength(pos[1] - pos[0]).
					WithReplaceText(fixedText)

				violation = violation.WithFixInfo(*fixInfo)
				violations = append(violations, *violation)
				continue
			}

			// For consistent style, establish expected style from first emphasis
			if styleConfig == "consistent" && expectedStyle == EmphasisUnknown {
				expectedStyle = currentStyle
				firstEmphasisLine = lineNumber
			}

			// Check for style violations
			if currentStyle != expectedStyle && !isWithinWord {
				violation := value.NewViolation(
					[]string{"MD049", "emphasis-style"},
					"Emphasis style",
					nil,
					lineNumber,
				)

				expectedStyleName := getEmphasisStyleName(expectedStyle)
				actualStyleName := getEmphasisStyleName(currentStyle)

				detail := fmt.Sprintf("Expected: %s, Actual: %s", expectedStyleName, actualStyleName)
				if styleConfig == "consistent" && firstEmphasisLine > 0 {
					detail += fmt.Sprintf(" [Expected: %s (based on line %d)]", expectedStyleName, firstEmphasisLine)
				}

				violation = violation.WithErrorDetail(detail)
				violation = violation.WithErrorContext(match[0])
				violation = violation.WithColumn(pos[0] + 1)
				violation = violation.WithLength(pos[1] - pos[0])

				// Add fix information - convert to expected style
				var newMarker string
				if expectedStyle == EmphasisAsterisk {
					newMarker = "*"
				} else {
					newMarker = "_"
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

// getEmphasisStyleName returns a human-readable name for emphasis style
func getEmphasisStyleName(style EmphasisStyle) string {
	switch style {
	case EmphasisAsterisk:
		return "asterisk"
	case EmphasisUnderscore:
		return "underscore"
	default:
		return "unknown"
	}
}

// isWordCharacter checks if a character is part of a word
func isWordCharacter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
}
