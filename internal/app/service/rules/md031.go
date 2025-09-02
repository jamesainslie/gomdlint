package rules

import (
	"context"
	"net/url"
	"strings"

	"github.com/gomdlint/gomdlint/internal/domain/entity"
	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// MD031 - Fenced code blocks should be surrounded by blank lines
func NewMD031Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md031.md")

	return entity.NewRule(
		[]string{"MD031", "blanks-around-fences"},
		"Fenced code blocks should be surrounded by blank lines",
		[]string{"blank_lines", "code"},
		infoURL,
		"commonmark",
		map[string]interface{}{
			"list_items": true, // Include list items
		},
		md031Function,
	)
}

func md031Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Get configuration
	checkListItems := getBoolConfig(params.Config, "list_items", true)

	// Track code block state to avoid double-processing fences
	inCodeBlock := false
	fenceChar := ""
	minFenceLength := 0

	// Find all fenced code blocks
	for i, line := range params.Lines {
		lineNumber := i + 1
		trimmed := strings.TrimSpace(line)

		// Check for fenced code block start/end
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			currentFenceChar := string(trimmed[0])

			// Count fence length
			currentFenceLength := 0
			for _, char := range trimmed {
				if string(char) == currentFenceChar {
					currentFenceLength++
				} else {
					break
				}
			}

			if !inCodeBlock {
				// This is a code block start
				// Skip if in list items and list_items is false
				if !checkListItems && isInListItem(params.Lines, i) {
					continue
				}

				inCodeBlock = true
				fenceChar = currentFenceChar
				minFenceLength = currentFenceLength

				// Check line above
				if i > 0 {
					prevLine := strings.TrimSpace(params.Lines[i-1])
					if prevLine != "" {
						violation := value.NewViolation(
							[]string{"MD031", "blanks-around-fences"},
							"Fenced code blocks should be surrounded by blank lines",
							nil,
							lineNumber,
						)

						violation = violation.WithErrorDetail("Missing blank line before fenced code block")
						violation = violation.WithErrorContext(strings.TrimSpace(line))

						// Add fix information - insert blank line before
						fixInfo := value.NewFixInfo().
							WithLineNumber(lineNumber).
							WithEditColumn(1).
							WithDeleteLength(0).
							WithReplaceText("\n")

						violation = violation.WithFixInfo(*fixInfo)
						violations = append(violations, *violation)
					}
				}
			} else if currentFenceChar == fenceChar && currentFenceLength >= minFenceLength {
				// This is a code block end
				inCodeBlock = false

				// Check line after closing fence
				if i < len(params.Lines)-1 {
					nextLine := strings.TrimSpace(params.Lines[i+1])
					if nextLine != "" {
						violation := value.NewViolation(
							[]string{"MD031", "blanks-around-fences"},
							"Fenced code blocks should be surrounded by blank lines",
							nil,
							lineNumber,
						)

						violation = violation.WithErrorDetail("Missing blank line after fenced code block")
						violation = violation.WithErrorContext(strings.TrimSpace(line))

						// Add fix information - insert blank line after
						fixInfo := value.NewFixInfo().
							WithLineNumber(lineNumber + 1). // After the closing fence
							WithEditColumn(1).
							WithDeleteLength(0).
							WithReplaceText("\n")

						violation = violation.WithFixInfo(*fixInfo)
						violations = append(violations, *violation)
					}
				}
			}
		}
	}

	return functional.Ok(violations)
}

// findClosingFence finds the matching closing fence for a code block
func findClosingFence(lines []string, startIndex int, fenceChar string, minLength int) int {
	for i := startIndex; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, fenceChar) {
			// Count fence characters
			fenceLength := 0
			for _, char := range line {
				if string(char) == fenceChar {
					fenceLength++
				} else {
					break
				}
			}

			if fenceLength >= minLength {
				return i // Found closing fence (0-based index)
			}
		}
	}
	return -1 // No closing fence found
}

// isInListItem checks if a line is part of a list item
func isInListItem(lines []string, lineIndex int) bool {
	// Simple heuristic: check if line is indented and preceded by a list marker
	if lineIndex == 0 {
		return false
	}

	line := lines[lineIndex]
	leadingSpaces := len(line) - len(strings.TrimLeft(line, " \t"))

	if leadingSpaces == 0 {
		return false
	}

	// Look backwards for list marker at lower indentation
	for i := lineIndex - 1; i >= 0; i-- {
		prevLine := lines[i]
		prevTrimmed := strings.TrimSpace(prevLine)

		if prevTrimmed == "" {
			continue
		}

		prevIndent := len(prevLine) - len(strings.TrimLeft(prevLine, " \t"))
		if prevIndent >= leadingSpaces {
			continue
		}

		// Check if this line has a list marker
		if strings.HasPrefix(strings.TrimLeft(prevLine, " \t"), "- ") ||
			strings.HasPrefix(strings.TrimLeft(prevLine, " \t"), "* ") ||
			strings.HasPrefix(strings.TrimLeft(prevLine, " \t"), "+ ") {
			return true
		}

		break
	}

	return false
}
