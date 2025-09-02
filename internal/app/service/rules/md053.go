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

// MD053 - Link and image reference definitions should be needed
func NewMD053Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md053.md")

	return entity.NewRule(
		[]string{"MD053", "link-image-reference-definitions"},
		"Link and image reference definitions should be needed",
		[]string{"links", "images"},
		infoURL,
		"commonmark",
		map[string]interface{}{
			"ignored_definitions": []interface{}{}, // Array of reference labels to ignore
		},
		md053Function,
	)
}

func md053Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Get configuration
	ignoredDefs := getStringSliceConfig(params.Config, "ignored_definitions")
	ignoredSet := make(map[string]bool)
	for _, ignored := range ignoredDefs {
		ignoredSet[strings.ToLower(ignored)] = true
	}

	// First pass: collect all reference definitions with their line numbers
	definedLabels := make(map[string]int) // label -> line number

	// Regex for reference definitions: [label]: url "title"
	refDefRegex := regexp.MustCompile(`^\s*\[([^\]]+)\]:\s*(.*)$`)

	for i, line := range params.Lines {
		if matches := refDefRegex.FindStringSubmatch(line); matches != nil {
			label := strings.ToLower(strings.TrimSpace(matches[1]))
			definedLabels[label] = i + 1 // Store 1-based line number
		}
	}

	// Second pass: collect all used reference labels
	usedLabels := make(map[string]bool)

	// Regex patterns for different reference types
	referenceLinkRegex := regexp.MustCompile(`\[([^\]]*)\]\[([^\]]*)\]`)   // [text][label]
	referenceImageRegex := regexp.MustCompile(`!\[([^\]]*)\]\[([^\]]*)\]`) // ![alt][label]
	shortcutLinkRegex := regexp.MustCompile(`\[([^\]]+)\](?!\[|\(|`)     // [label] (not followed by [ ( or 
	shortcutImageRegex := regexp.MustCompile(`!\[([^\]]+)\](?!\[|\()`)     // ![label] (not followed by [ or ()

	for _, line := range params.Lines {
		// Skip reference definitions themselves
		if refDefRegex.MatchString(line) {
			continue
		}

		// Check reference links [text][label]
		matches := referenceLinkRegex.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			text := match[1]
			label := strings.ToLower(strings.TrimSpace(match[2]))

			// Empty label means use text as label
			if label == "" {
				label = strings.ToLower(strings.TrimSpace(text))
			}

			usedLabels[label] = true
		}

		// Check reference images ![alt][label]
		imgMatches := referenceImageRegex.FindAllStringSubmatch(line, -1)
		for _, match := range imgMatches {
			alt := match[1]
			label := strings.ToLower(strings.TrimSpace(match[2]))

			// Empty label means use alt text as label
			if label == "" {
				label = strings.ToLower(strings.TrimSpace(alt))
			}

			usedLabels[label] = true
		}

		// Check shortcut reference links [label]
		shortcutMatches := shortcutLinkRegex.FindAllStringSubmatch(line, -1)
		for _, match := range shortcutMatches {
			label := strings.ToLower(strings.TrimSpace(match[1]))
			usedLabels[label] = true
		}

		// Check shortcut reference images ![label]
		shortcutImgMatches := shortcutImageRegex.FindAllStringSubmatch(line, -1)
		for _, match := range shortcutImgMatches {
			label := strings.ToLower(strings.TrimSpace(match[1]))
			usedLabels[label] = true
		}
	}

	// Third pass: find unused reference definitions
	for label, lineNumber := range definedLabels {
		// Skip if this label is in the ignored list
		if ignoredSet[label] {
			continue
		}

		// Check if this label is used anywhere
		if !usedLabels[label] {
			violation := value.NewViolation(
				[]string{"MD053", "link-image-reference-definitions"},
				"Link and image reference definitions should be needed",
				nil,
				lineNumber,
			)

			violation = violation.WithErrorDetail("Reference definition is not used: " + label)
			violation = violation.WithErrorContext("[" + label + "]: ...")

			violations = append(violations, *violation)
		}
	}

	return functional.Ok(violations)
}
