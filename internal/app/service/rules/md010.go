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

// MD010 - Hard tabs
func NewMD010Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md010.md")

	return entity.NewRule(
		[]string{"MD010", "no-hard-tabs"},
		"Hard tabs",
		[]string{"whitespace", "hard_tab"},
		infoURL,
		"commonmark",
		map[string]interface{}{
			"code_blocks":           true,            // Check code blocks by default
			"ignore_code_languages": []interface{}{}, // Languages to ignore
			"spaces_per_tab":        4,               // Number of spaces to replace tabs with
		},
		md010Function,
	)
}

func md010Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Get configuration
	codeBlocks := getBoolConfig(params.Config, "code_blocks", true)
	ignoreCodeLanguages := getStringSliceConfig(params.Config, "ignore_code_languages")
	spacesPerTab := getIntConfig(params.Config, "spaces_per_tab", 4)

	// Convert ignore languages to a set for faster lookup
	ignoreLanguageSet := make(map[string]bool)
	for _, lang := range ignoreCodeLanguages {
		ignoreLanguageSet[strings.ToLower(lang)] = true
	}

	// Check each line for hard tabs
	for i, line := range params.Lines {
		lineNumber := i + 1

		// Find all tab characters in the line
		tabPositions := findTabPositions(line)
		if len(tabPositions) == 0 {
			continue
		}

		// Check if this line should be ignored based on context
		if shouldIgnoreLine(params.Tokens, lineNumber, codeBlocks, ignoreLanguageSet) {
			continue
		}

		// Create violations for each tab
		for _, tabPos := range tabPositions {
			violation := value.NewViolation(
				[]string{"MD010", "no-hard-tabs"},
				"Hard tabs",
				nil,
				lineNumber,
			)

			violation = violation.WithColumn(tabPos + 1) // Convert to 1-based
			violation = violation.WithErrorDetail(fmt.Sprintf("Column: %d", tabPos+1))

			// Add fix information
			fixInfo := value.NewFixInfo().
				WithLineNumber(lineNumber).
				WithEditColumn(tabPos + 1).
				WithDeleteLength(1).
				WithReplaceText(strings.Repeat(" ", spacesPerTab))

			violation = violation.WithFixInfo(*fixInfo)

			violations = append(violations, *violation)
		}
	}

	return functional.Ok(violations)
}

// findTabPositions returns the positions of all tab characters in a line
func findTabPositions(line string) []int {
	var positions []int
	for i, char := range line {
		if char == '\t' {
			positions = append(positions, i)
		}
	}
	return positions
}

// shouldIgnoreLine determines if a line should be ignored based on its context
func shouldIgnoreLine(tokens []value.Token, lineNumber int, codeBlocks bool, ignoreLanguageSet map[string]bool) bool {
	// Find the token that contains this line
	containingToken := findTokenContainingLine(tokens, lineNumber)
	if containingToken == nil {
		return false
	}

	// Check if we're in a code block that should be ignored
	if !codeBlocks && containingToken.IsCodeBlock() {
		return true
	}

	// Check if we're in a code block with an ignored language
	if containingToken.IsType(value.TokenTypeCodeFenced) {
		if language, exists := containingToken.GetStringProperty("language"); exists {
			if ignoreLanguageSet[strings.ToLower(language)] {
				return true
			}
		}
	}

	return false
}

// findTokenContainingLine finds the token that contains the specified line number
func findTokenContainingLine(tokens []value.Token, lineNumber int) *value.Token {
	var findInTokens func([]value.Token) *value.Token
	findInTokens = func(tokenList []value.Token) *value.Token {
		for _, token := range tokenList {
			// Check if this token contains the line
			if token.StartLine() <= lineNumber && lineNumber <= token.EndLine() {
				// Check children first (more specific)
				if token.HasChildren() {
					if childResult := findInTokens(token.Children); childResult != nil {
						return childResult
					}
				}
				// Return this token if no child contains the line
				return &token
			}
		}
		return nil
	}

	return findInTokens(tokens)
}

// Helper functions to safely extract configuration values
func getBoolConfig(config map[string]interface{}, key string, defaultValue bool) bool {
	if value, exists := config[key]; exists {
		if boolValue, ok := value.(bool); ok {
			return boolValue
		}
	}
	return defaultValue
}

func getStringSliceConfig(config map[string]interface{}, key string) []string {
	if value, exists := config[key]; exists {
		if slice, ok := value.([]interface{}); ok {
			var result []string
			for _, item := range slice {
				if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
		if slice, ok := value.([]string); ok {
			return slice
		}
	}
	return []string{}
}

func getIntConfig(config map[string]interface{}, key string, defaultValue int) int {
	if value, exists := config[key]; exists {
		if intValue, ok := value.(int); ok {
			return intValue
		}
		if floatValue, ok := value.(float64); ok {
			return int(floatValue)
		}
	}
	return defaultValue
}
