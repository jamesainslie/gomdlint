package rules

import (
	"context"
	"fmt"
	"net/url"

	"github.com/gomdlint/gomdlint/internal/domain/entity"
	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// MD001 - Heading levels should only increment by one level at a time
func NewMD001Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md001.md")

	return entity.NewRule(
		[]string{"MD001", "heading-increment"},
		"Heading levels should only increment by one level at a time",
		[]string{"headings"},
		infoURL,
		"commonmark",
		map[string]interface{}{},
		md001Function,
	)
}

func md001Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Filter tokens to find heading tokens
	headingTokens := filterHeadings(params.Tokens)

	// Track the previous heading level
	prevLevel := 0

	for _, heading := range headingTokens {
		level := getHeadingLevel(heading)

		// Check if the level increment is too large
		if level > prevLevel+1 && prevLevel > 0 {
			violation := value.NewViolation(
				[]string{"MD001", "heading-increment"},
				"Heading levels should only increment by one level at a time",
				nil, // Will be set by rule engine
				heading.StartLine(),
			)

			expectedLevel := prevLevel + 1
			violation = violation.WithErrorDetail(fmt.Sprintf("Expected h%d, found h%d", expectedLevel, level))
			violation = violation.WithErrorContext(heading.Text)

			violations = append(violations, *violation)
		}

		prevLevel = level
	}

	return functional.Ok(violations)
}

// filterHeadings returns only heading tokens from the token list
func filterHeadings(tokens []value.Token) []value.Token {
	var headings []value.Token

	// Recursively find all heading tokens
	var findHeadings func([]value.Token)
	findHeadings = func(tokenList []value.Token) {
		for _, token := range tokenList {
			if token.IsHeading() {
				headings = append(headings, token)
			}
			// Recursively search children
			if token.HasChildren() {
				findHeadings(token.Children)
			}
		}
	}

	findHeadings(tokens)
	return headings
}

// getHeadingLevel extracts the heading level from a heading token
func getHeadingLevel(heading value.Token) int {
	if level, exists := heading.GetIntProperty("level"); exists {
		return level
	}

	// Fallback: analyze the heading text for ATX headings
	if heading.IsType(value.TokenTypeATXHeading) {
		text := heading.Text
		level := 0
		for i := 0; i < len(text) && text[i] == '#'; i++ {
			level++
		}
		return level
	}

	// Setext headings are level 1 or 2
	if heading.IsType(value.TokenTypeSetextHeading) {
		// This would need to be determined by the underline character
		return 1 // Simplified - would need proper detection
	}

	return 1 // Default
}
