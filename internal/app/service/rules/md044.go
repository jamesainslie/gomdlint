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

// MD044 - Proper names should have the correct capitalization
func NewMD044Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md044.md")

	return entity.NewRule(
		[]string{"MD044", "proper-names"},
		"Proper names should have the correct capitalization",
		[]string{"spelling"},
		infoURL,
		"commonmark",
		map[string]interface{}{
			"names":         []interface{}{}, // Array of proper names with correct capitalization
			"code_blocks":   true,            // Check inside code blocks
			"html_elements": true,            // Check inside HTML elements
		},
		md044Function,
	)
}

func md044Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Get configuration
	properNames := getStringSliceConfig(params.Config, "names")
	checkCodeBlocks := getBoolConfig(params.Config, "code_blocks", true)
	checkHTMLElements := getBoolConfig(params.Config, "html_elements", true)

	// If no proper names specified, skip rule
	if len(properNames) == 0 {
		return functional.Ok(violations)
	}

	// Create case-insensitive lookup map
	nameMap := make(map[string]string) // lowercase -> correct case
	for _, name := range properNames {
		nameMap[strings.ToLower(name)] = name
	}

	// Track state for code blocks
	inFencedCodeBlock := false

	// Process each line
	for i, line := range params.Lines {
		lineNumber := i + 1

		// Track fenced code block state
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFencedCodeBlock = !inFencedCodeBlock
			continue
		}

		// Skip code blocks if not checking them
		if inFencedCodeBlock && !checkCodeBlocks {
			continue
		}

		// Skip indented code blocks if not checking them
		if !checkCodeBlocks && (strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t")) {
			continue
		}

		// Process the line for proper names
		violations = append(violations, checkLineForProperNames(line, lineNumber, nameMap, checkHTMLElements)...)
	}

	return functional.Ok(violations)
}

// checkLineForProperNames checks a line for proper name violations
func checkLineForProperNames(line string, lineNumber int, nameMap map[string]string, checkHTMLElements bool) []value.Violation {
	var violations []value.Violation

	// If not checking HTML elements, remove them from consideration
	processLine := line
	if !checkHTMLElements {
		// Simple approach: replace HTML tags with spaces
		htmlTagRegex := regexp.MustCompile(`<[^>]*>`)
		processLine = htmlTagRegex.ReplaceAllString(line, " ")
	}

	// Also skip inline code spans
	codeSpanRegex := regexp.MustCompile("`[^`]*`")
	processLine = codeSpanRegex.ReplaceAllString(processLine, " ")

	// Check each proper name
	for lowerName, correctName := range nameMap {
		// Create regex to find the name as a whole word (case-insensitive)
		// Use word boundaries to avoid partial matches
		pattern := `(?i)\b` + regexp.QuoteMeta(lowerName) + `\b`
		regex := regexp.MustCompile(pattern)

		// Find all matches
		matches := regex.FindAllString(processLine, -1)
		positions := regex.FindAllStringIndex(processLine, -1)

		for j, match := range matches {
			// Check if the found match has incorrect capitalization
			if match != correctName {
				pos := positions[j]

				violation := value.NewViolation(
					[]string{"MD044", "proper-names"},
					"Proper names should have the correct capitalization",
					nil,
					lineNumber,
				)

				violation = violation.WithErrorDetail("'" + match + "' should be '" + correctName + "'")
				violation = violation.WithErrorContext(match)
				violation = violation.WithColumn(pos[0] + 1) // 1-based column
				violation = violation.WithLength(pos[1] - pos[0])

				// Add fix information
				fixInfo := value.NewFixInfo().
					WithLineNumber(lineNumber).
					WithEditColumn(pos[0] + 1).
					WithDeleteLength(pos[1] - pos[0]).
					WithReplaceText(correctName)

				violation = violation.WithFixInfo(*fixInfo)
				violations = append(violations, *violation)
			}
		}
	}

	return violations
}
