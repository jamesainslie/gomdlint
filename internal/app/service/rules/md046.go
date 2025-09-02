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

// MD046 - Code block style
func NewMD046Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md046.md")

	return entity.NewRule(
		[]string{"MD046", "code-block-style"},
		"Code block style",
		[]string{"code"},
		infoURL,
		"commonmark",
		map[string]interface{}{
			"style": "consistent", // consistent|fenced|indented
		},
		md046Function,
	)
}

type CodeBlockStyle int

const (
	CodeBlockUnknown CodeBlockStyle = iota
	CodeBlockFenced
	CodeBlockIndented
)

func md046Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Get configuration
	styleConfig := getStringConfig(params.Config, "style", "consistent")

	var expectedStyle CodeBlockStyle
	var firstCodeBlockLine int

	// Parse expected style if not consistent
	if styleConfig != "consistent" {
		switch styleConfig {
		case "fenced":
			expectedStyle = CodeBlockFenced
		case "indented":
			expectedStyle = CodeBlockIndented
		}
	}

	// Track code blocks
	inFencedBlock := false

	// Process each line
	for i, line := range params.Lines {
		lineNumber := i + 1
		trimmed := strings.TrimSpace(line)

		// Check for fenced code blocks
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			if !inFencedBlock {
				// Starting a fenced code block
				inFencedBlock = true

				// For consistent style, establish expected style from first code block
				if styleConfig == "consistent" && expectedStyle == CodeBlockUnknown {
					expectedStyle = CodeBlockFenced
					firstCodeBlockLine = lineNumber
				}

				// Check for style violations
				if expectedStyle == CodeBlockIndented {
					violation := value.NewViolation(
						[]string{"MD046", "code-block-style"},
						"Code block style",
						nil,
						lineNumber,
					)

					detail := fmt.Sprintf("Expected: indented, Actual: fenced")
					if styleConfig == "consistent" && firstCodeBlockLine > 0 {
						detail += fmt.Sprintf(" [Expected: indented (based on line %d)]", firstCodeBlockLine)
					}

					violation = violation.WithErrorDetail(detail)
					violation = violation.WithErrorContext(strings.TrimSpace(line))

					// Add fix information - convert fenced to indented
					fixInfo := createFencedToIndentedFix(params.Lines, i)
					if fixInfo != nil {
						violation = violation.WithFixInfo(*fixInfo)
					}

					violations = append(violations, *violation)
				}
			} else {
				// Ending a fenced code block
				inFencedBlock = false
			}
		} else if !inFencedBlock && (strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t")) {
			// This might be an indented code block
			if isIndentedCodeBlock(params.Lines, i) {
				// For consistent style, establish expected style from first code block
				if styleConfig == "consistent" && expectedStyle == CodeBlockUnknown {
					expectedStyle = CodeBlockIndented
					firstCodeBlockLine = lineNumber
				}

				// Check for style violations
				if expectedStyle == CodeBlockFenced {
					violation := value.NewViolation(
						[]string{"MD046", "code-block-style"},
						"Code block style",
						nil,
						lineNumber,
					)

					detail := fmt.Sprintf("Expected: fenced, Actual: indented")
					if styleConfig == "consistent" && firstCodeBlockLine > 0 {
						detail += fmt.Sprintf(" [Expected: fenced (based on line %d)]", firstCodeBlockLine)
					}

					violation = violation.WithErrorDetail(detail)
					violation = violation.WithErrorContext(strings.TrimSpace(line))

					// Add fix information - convert indented to fenced
					fixInfo := createIndentedToFencedFix(params.Lines, i)
					if fixInfo != nil {
						violation = violation.WithFixInfo(*fixInfo)
					}

					violations = append(violations, *violation)
				}
			}
		}
	}

	return functional.Ok(violations)
}

// isIndentedCodeBlock determines if a line is part of an indented code block
func isIndentedCodeBlock(lines []string, lineIndex int) bool {
	line := lines[lineIndex]

	// Must be indented by at least 4 spaces or 1 tab
	if !strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "\t") {
		return false
	}

	// Check if previous line is blank (which would start an indented code block)
	if lineIndex == 0 {
		return true // First line, assume code block
	}

	prevLine := strings.TrimSpace(lines[lineIndex-1])
	if prevLine == "" {
		// Previous line is blank, this could be start of code block
		return true
	}

	// Check if previous line is also indented (continuation of code block)
	if strings.HasPrefix(lines[lineIndex-1], "    ") || strings.HasPrefix(lines[lineIndex-1], "\t") {
		return true
	}

	// Check if this is inside a list item (then indentation might be for list content)
	// This is a simplified check
	if isIndentedInList(lines, lineIndex) {
		return false
	}

	return false
}

// isIndentedInList checks if the indented line is part of list item content
func isIndentedInList(lines []string, lineIndex int) bool {
	// Look backwards for a list marker
	for i := lineIndex - 1; i >= 0; i-- {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		// Check for list markers
		if strings.HasPrefix(strings.TrimLeft(line, " \t"), "- ") ||
			strings.HasPrefix(strings.TrimLeft(line, " \t"), "* ") ||
			strings.HasPrefix(strings.TrimLeft(line, " \t"), "+ ") {
			// Found unordered list marker
			listIndent := len(line) - len(strings.TrimLeft(line, " \t"))
			currentIndent := len(lines[lineIndex]) - len(strings.TrimLeft(lines[lineIndex], " \t"))

			// If current line is indented more than list marker, it's list content
			return currentIndent > listIndent
		}

		// Stop looking if we hit non-indented content
		if !strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "\t") && trimmed != "" {
			break
		}
	}

	return false
}

// createFencedToIndentedFix creates fix information to convert fenced code block to indented
func createFencedToIndentedFix(lines []string, fenceLineIndex int) *value.FixInfo {
	// Find the complete fenced code block
	startFence := fenceLineIndex
	endFence := -1

	// Find the closing fence
	fenceMarker := ""
	if strings.HasPrefix(strings.TrimSpace(lines[startFence]), "```") {
		fenceMarker = "```"
	} else if strings.HasPrefix(strings.TrimSpace(lines[startFence]), "~~~") {
		fenceMarker = "~~~"
	} else {
		return nil // Not a fenced block
	}

	for i := startFence + 1; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), fenceMarker) {
			endFence = i
			break
		}
	}

	if endFence == -1 {
		return nil // No closing fence found
	}

	// Build the replacement indented code block
	var newLines []string

	// Add content lines with 4-space indentation
	for i := startFence + 1; i < endFence; i++ {
		contentLine := lines[i]
		// Add 4 spaces to the beginning of each line
		indentedLine := "    " + contentLine
		newLines = append(newLines, indentedLine)
	}

	replacement := strings.Join(newLines, "\n")

	// Replace the entire fenced block (including fences)
	return value.NewFixInfo().
		WithLineNumber(startFence + 1).
		WithDeleteCount(endFence - startFence + 1).
		WithInsertText(replacement)
}

// createIndentedToFencedFix creates fix information to convert indented code block to fenced
func createIndentedToFencedFix(lines []string, codeLineIndex int) *value.FixInfo {
	// Find the complete indented code block
	start := codeLineIndex
	end := codeLineIndex

	// Find start of code block (look backwards)
	for i := codeLineIndex - 1; i >= 0; i-- {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			// Blank line - continue looking
			continue
		} else if strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t") {
			// Still part of code block
			start = i
		} else {
			// Found non-indented line, stop here
			break
		}
	}

	// Find end of code block (look forwards)
	for i := codeLineIndex + 1; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			// Blank line - might be part of code block, keep looking
			continue
		} else if strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t") {
			// Still part of code block
			end = i
		} else {
			// Found non-indented line, code block ends here
			break
		}
	}

	// Build the replacement fenced code block
	var newLines []string
	newLines = append(newLines, "```") // Opening fence

	// Add content lines with indentation removed
	for i := start; i <= end; i++ {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			newLines = append(newLines, "") // Preserve blank lines
		} else {
			// Remove 4 spaces or 1 tab from beginning
			var unindentedLine string
			if strings.HasPrefix(line, "    ") {
				unindentedLine = line[4:]
			} else if strings.HasPrefix(line, "\t") {
				unindentedLine = line[1:]
			} else {
				unindentedLine = line // Shouldn't happen, but handle gracefully
			}
			newLines = append(newLines, unindentedLine)
		}
	}

	newLines = append(newLines, "```") // Closing fence
	replacement := strings.Join(newLines, "\n")

	// Replace the entire indented code block
	return value.NewFixInfo().
		WithLineNumber(start + 1).
		WithDeleteCount(end - start + 1).
		WithInsertText(replacement)
}
