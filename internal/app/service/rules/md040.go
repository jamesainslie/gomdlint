package rules

import (
	"context"
	"net/url"
	"strings"

	"github.com/gomdlint/gomdlint/internal/domain/entity"
	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// MD040 - Fenced code blocks should have a language specified
func NewMD040Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md040.md")

	return entity.NewRule(
		[]string{"MD040", "fenced-code-language"},
		"Fenced code blocks should have a language specified",
		[]string{"code", "language"},
		infoURL,
		"commonmark",
		map[string]interface{}{
			"allowed_languages": []interface{}{}, // List of allowed languages (empty = any)
			"language_only":     false,           // Require language only (no extra info)
		},
		md040Function,
	)
}

func md040Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Get configuration
	allowedLanguages := getStringSliceConfig(params.Config, "allowed_languages")
	languageOnly := getBoolConfig(params.Config, "language_only", false)

	// Create set of allowed languages for fast lookup
	allowedSet := make(map[string]bool)
	for _, lang := range allowedLanguages {
		allowedSet[strings.ToLower(lang)] = true
	}
	allowAll := len(allowedLanguages) == 0

	// Process each line to find fenced code blocks
	for i, line := range params.Lines {
		lineNumber := i + 1
		trimmed := strings.TrimSpace(line)

		// Check for fenced code block start
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			fence := ""
			if strings.HasPrefix(trimmed, "```") {
				fence = "```"
			} else {
				fence = "~~~"
			}

			// Extract info string (language and optional parameters)
			infoString := strings.TrimSpace(trimmed[len(fence):])

			// Check if language is missing
			if infoString == "" {
				violation := value.NewViolation(
					[]string{"MD040", "fenced-code-language"},
					"Fenced code blocks should have a language specified",
					nil,
					lineNumber,
				)

				violation = violation.WithErrorDetail("Missing language specification")
				violation = violation.WithErrorContext(trimmed)

				// Add fix information - add "text" as default language
				fixedLine := strings.Replace(line, fence, fence+"text", 1)

				fixInfo := value.NewFixInfo().
					WithLineNumber(lineNumber).
					WithEditColumn(1).
					WithDeleteLength(len(line)).
					WithReplaceText(fixedLine)

				violation = violation.WithFixInfo(*fixInfo)
				violations = append(violations, *violation)
			} else {
				// Language is specified, check if it's allowed
				parts := strings.Fields(infoString)
				if len(parts) > 0 {
					language := strings.ToLower(parts[0])

					// Check allowed languages
					if !allowAll && !allowedSet[language] {
						violation := value.NewViolation(
							[]string{"MD040", "fenced-code-language"},
							"Fenced code blocks should have a language specified",
							nil,
							lineNumber,
						)

						violation = violation.WithErrorDetail("Language '" + parts[0] + "' is not in allowed list")
						violation = violation.WithErrorContext(trimmed)

						violations = append(violations, *violation)
					}

					// Check language_only requirement
					if languageOnly && (len(parts) > 1 || strings.TrimSpace(infoString) != parts[0]) {
						violation := value.NewViolation(
							[]string{"MD040", "fenced-code-language"},
							"Fenced code blocks should have a language specified",
							nil,
							lineNumber,
						)

						violation = violation.WithErrorDetail("Only language name allowed, no additional info")
						violation = violation.WithErrorContext(trimmed)

						// Add fix information - keep only the language
						fixedLine := strings.Replace(line, infoString, parts[0], 1)

						fixInfo := value.NewFixInfo().
							WithLineNumber(lineNumber).
							WithEditColumn(1).
							WithDeleteLength(len(line)).
							WithReplaceText(fixedLine)

						violation = violation.WithFixInfo(*fixInfo)
						violations = append(violations, *violation)
					}
				}
			}
		}
	}

	return functional.Ok(violations)
}
