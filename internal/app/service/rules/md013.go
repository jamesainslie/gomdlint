package rules

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/gomdlint/gomdlint/internal/domain/entity"
	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// MD013 - Line length
func NewMD013Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md013.md")

	return entity.NewRule(
		[]string{"MD013", "line-length"},
		"Line length",
		[]string{"line_length"},
		infoURL,
		"commonmark",
		map[string]interface{}{
			"line_length":            80,    // Maximum line length
			"heading_line_length":    80,    // Maximum heading line length
			"code_block_line_length": 80,    // Maximum code block line length
			"code_blocks":            true,  // Check code blocks
			"tables":                 true,  // Check tables
			"headings":               true,  // Check headings
			"headers":                true,  // Alias for headings
			"stern":                  false, // Strict mode - no exceptions
		},
		md013Function,
	)
}

func md013Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Get configuration
	lineLength := getIntConfig(params.Config, "line_length", 80)
	headingLineLength := getIntConfig(params.Config, "heading_line_length", lineLength)
	codeBlockLineLength := getIntConfig(params.Config, "code_block_line_length", lineLength)
	checkCodeBlocks := getBoolConfig(params.Config, "code_blocks", true)
	checkTables := getBoolConfig(params.Config, "tables", true)
	checkHeadings := getBoolConfig(params.Config, "headings", true) || getBoolConfig(params.Config, "headers", true)
	stern := getBoolConfig(params.Config, "stern", false)

	// Pre-compile regex for URL detection (URLs are often exempt from length limits)
	urlRegex := regexp.MustCompile(`https?://\S+`)

	// Check each line
	for i, line := range params.Lines {
		lineNumber := i + 1

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Get the context for this line
		lineContext := getLineContext(params.Tokens, lineNumber)

		// Determine which length limit to use
		maxLength := getMaxLengthForContext(lineContext, lineLength, headingLineLength, codeBlockLineLength)

		// Check if this line type should be checked
		if !shouldCheckLineType(lineContext, checkCodeBlocks, checkTables, checkHeadings) {
			continue
		}

		// Calculate the effective line length
		effectiveLength := calculateEffectiveLength(line, lineContext, stern, urlRegex)

		// Check if line exceeds maximum length
		if effectiveLength > maxLength {
			violation := value.NewViolation(
				[]string{"MD013", "line-length"},
				"Line length",
				nil,
				lineNumber,
			)

			violation = violation.WithErrorDetail(fmt.Sprintf("Expected: <=%d, Actual: %d", maxLength, effectiveLength))
			violation = violation.WithColumn(maxLength + 1)
			violation = violation.WithLength(effectiveLength - maxLength)

			// Add error range for the excess characters
			errorRange := value.Range{
				Start: value.Position{Line: lineNumber, Column: maxLength + 1, Offset: maxLength},
				End:   value.Position{Line: lineNumber, Column: effectiveLength + 1, Offset: effectiveLength},
			}
			violation = violation.WithErrorRange(errorRange)

			violations = append(violations, *violation)
		}
	}

	return functional.Ok(violations)
}

// LineContext represents the context/type of a line
type LineContext int

const (
	ContextParagraph LineContext = iota
	ContextHeading
	ContextCodeBlock
	ContextTable
	ContextBlockquote
	ContextList
	ContextHTML
)

// getLineContext determines the context/type of a specific line
func getLineContext(tokens []value.Token, lineNumber int) LineContext {
	containingToken := findTokenContainingLine(tokens, lineNumber)
	if containingToken == nil {
		return ContextParagraph
	}

	switch {
	case containingToken.IsHeading():
		return ContextHeading
	case containingToken.IsCodeBlock():
		return ContextCodeBlock
	case containingToken.IsType(value.TokenTypeTableRow):
		return ContextTable
	case containingToken.IsType(value.TokenTypeBlockQuote):
		return ContextBlockquote
	case containingToken.IsType(value.TokenTypeListItem):
		return ContextList
	case containingToken.IsType(value.TokenTypeHTMLFlow) || containingToken.IsType(value.TokenTypeHTMLText):
		return ContextHTML
	default:
		return ContextParagraph
	}
}

// getMaxLengthForContext returns the appropriate maximum length for the given context
func getMaxLengthForContext(context LineContext, lineLength, headingLineLength, codeBlockLineLength int) int {
	switch context {
	case ContextHeading:
		return headingLineLength
	case ContextCodeBlock:
		return codeBlockLineLength
	default:
		return lineLength
	}
}

// shouldCheckLineType determines if a line of the given type should be checked
func shouldCheckLineType(context LineContext, checkCodeBlocks, checkTables, checkHeadings bool) bool {
	switch context {
	case ContextCodeBlock:
		return checkCodeBlocks
	case ContextTable:
		return checkTables
	case ContextHeading:
		return checkHeadings
	default:
		return true
	}
}

// calculateEffectiveLength calculates the effective length of a line considering various factors
func calculateEffectiveLength(line string, context LineContext, stern bool, urlRegex *regexp.Regexp) int {
	// Start with the actual character count (UTF-8 aware)
	length := utf8.RuneCountInString(line)

	// In stern mode, use actual length without any adjustments
	if stern {
		return length
	}

	// For code blocks, tabs are often expanded to spaces, so adjust accordingly
	if context == ContextCodeBlock {
		// Count tabs and assume they expand to 4 spaces
		tabCount := strings.Count(line, "\t")
		length += tabCount * 3 // Add 3 more for each tab (4 total - 1 for the tab character)
	}

	// URLs are often exempt from length checks in many markdown contexts
	// Find URLs and subtract their length beyond a reasonable display length
	urlMatches := urlRegex.FindAllString(line, -1)
	for _, url := range urlMatches {
		urlLength := utf8.RuneCountInString(url)
		// Allow URLs up to 30 characters without penalty
		if urlLength > 30 {
			length -= (urlLength - 30)
		}
	}

	// Ensure we don't return a negative length
	if length < 0 {
		length = 0
	}

	return length
}
