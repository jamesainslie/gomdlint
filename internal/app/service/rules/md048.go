package rules

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/gomdlint/gomdlint/internal/domain/entity"
	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// MD048 - Code fence style
func NewMD048Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md048.md")

	return entity.NewRule(
		[]string{"MD048", "code-fence-style"},
		"Code fence style",
		[]string{"code"},
		infoURL,
		"commonmark",
		map[string]interface{}{
			"style": "consistent", // consistent|backtick|tilde
		},
		md048Function,
	)
}

type FenceStyle int

const (
	FenceUnknown  FenceStyle = iota
	FenceBacktick            // ```
	FenceTilde               // ~~~
)

func md048Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Get configuration
	styleConfig := getStringConfig(params.Config, "style", "consistent")

	var expectedStyle FenceStyle
	var firstFenceLine int

	// Parse expected style if not consistent
	if styleConfig != "consistent" {
		switch styleConfig {
		case "backtick":
			expectedStyle = FenceBacktick
		case "tilde":
			expectedStyle = FenceTilde
		}
	}

	// Process each line to find code fences
	for i, line := range params.Lines {
		lineNumber := i + 1
		trimmed := strings.TrimSpace(line)

		// Skip empty lines
		if trimmed == "" {
			continue
		}

		var currentStyle FenceStyle
		var isFence bool

		// Check for backtick fence
		if strings.HasPrefix(trimmed, "```") {
			currentStyle = FenceBacktick
			isFence = true
		}

		// Check for tilde fence
		if strings.HasPrefix(trimmed, "~~~") {
			currentStyle = FenceTilde
			isFence = true
		}

		if !isFence {
			continue
		}

		// For consistent style, establish expected style from first fence
		if styleConfig == "consistent" && expectedStyle == FenceUnknown {
			expectedStyle = currentStyle
			firstFenceLine = lineNumber
		}

		// Check for style violations
		if currentStyle != expectedStyle {
			violation := value.NewViolation(
				[]string{"MD048", "code-fence-style"},
				"Code fence style",
				nil,
				lineNumber,
			)

			expectedStyleName := getFenceStyleName(expectedStyle)
			actualStyleName := getFenceStyleName(currentStyle)

			detail := fmt.Sprintf("Expected: %s, Actual: %s", expectedStyleName, actualStyleName)
			if styleConfig == "consistent" && firstFenceLine > 0 {
				detail += fmt.Sprintf(" [Expected: %s (based on line %d)]", expectedStyleName, firstFenceLine)
			}

			violation = violation.WithErrorDetail(detail)
			violation = violation.WithErrorContext(strings.TrimSpace(line))

			// Add fix information - replace fence style
			var fixedLine string
			if expectedStyle == FenceBacktick {
				// Replace ~~~ with ```
				if strings.HasPrefix(trimmed, "~~~") {
					fenceLength := 0
					for _, char := range trimmed {
						if char == '~' {
							fenceLength++
						} else {
							break
						}
					}
					replacement := strings.Repeat("`", fenceLength) + trimmed[fenceLength:]
					fixedLine = strings.Replace(line, trimmed, replacement, 1)
				} else {
					fixedLine = line
				}
			} else {
				// Replace ``` with ~~~
				if strings.HasPrefix(trimmed, "```") {
					fenceLength := 0
					for _, char := range trimmed {
						if char == '`' {
							fenceLength++
						} else {
							break
						}
					}
					replacement := strings.Repeat("~", fenceLength) + trimmed[fenceLength:]
					fixedLine = strings.Replace(line, trimmed, replacement, 1)
				} else {
					fixedLine = line
				}
			}

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

// getFenceStyleName returns a human-readable name for a fence style
func getFenceStyleName(style FenceStyle) string {
	switch style {
	case FenceBacktick:
		return "backtick"
	case FenceTilde:
		return "tilde"
	default:
		return "unknown"
	}
}
