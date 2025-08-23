package rules

import (
	"context"
	"net/url"
	"strings"

	"github.com/gomdlint/gomdlint/internal/domain/entity"
	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// MD014 - Dollar signs used before commands without showing output
func NewMD014Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md014.md")

	return entity.NewRule(
		[]string{"MD014", "commands-show-output"},
		"Dollar signs used before commands without showing output",
		[]string{"code"},
		infoURL,
		"commonmark",
		map[string]interface{}{},
		md014Function,
	)
}

func md014Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	// Find code blocks
	inCodeBlock := false
	codeBlockLines := []string{}
	codeBlockStart := 0

	for i, line := range params.Lines {
		lineNumber := i + 1
		trimmed := strings.TrimSpace(line)

		// Detect fenced code blocks
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			if !inCodeBlock {
				// Starting a code block
				inCodeBlock = true
				codeBlockStart = lineNumber + 1
				codeBlockLines = []string{}
			} else {
				// Ending a code block - check it
				violations = append(violations, checkCodeBlock(codeBlockLines, codeBlockStart)...)
				inCodeBlock = false
				codeBlockLines = []string{}
			}
			continue
		}

		// Collect lines within code block
		if inCodeBlock {
			codeBlockLines = append(codeBlockLines, line)
		}

		// Also check indented code blocks (4 spaces or 1 tab)
		if !inCodeBlock && (strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t")) {
			// This is an indented code block line
			violations = append(violations, checkIndentedCodeLine(line, lineNumber)...)
		}
	}

	// Check final code block if file ends while in one
	if inCodeBlock {
		violations = append(violations, checkCodeBlock(codeBlockLines, codeBlockStart)...)
	}

	return functional.Ok(violations)
}

// checkCodeBlock checks a fenced code block for dollar sign violations
func checkCodeBlock(lines []string, startLineNumber int) []value.Violation {
	var violations []value.Violation

	if len(lines) == 0 {
		return violations
	}

	// Check if ALL non-empty lines start with $
	allLinesHaveDollar := true
	commandLines := []int{}
	hasOutput := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if strings.HasPrefix(trimmed, "$ ") {
			commandLines = append(commandLines, i)
		} else {
			allLinesHaveDollar = false
			if len(commandLines) > 0 {
				// There's output after commands
				hasOutput = true
			}
		}
	}

	// Only flag if ALL commands have $ and there's NO output
	if allLinesHaveDollar && len(commandLines) > 0 && !hasOutput {
		for _, lineIndex := range commandLines {
			lineNumber := startLineNumber + lineIndex
			line := lines[lineIndex]

			violation := value.NewViolation(
				[]string{"MD014", "commands-show-output"},
				"Dollar signs used before commands without showing output",
				nil,
				lineNumber,
			)

			violation = violation.WithErrorDetail("Unnecessary dollar sign in code block")
			violation = violation.WithErrorContext(strings.TrimSpace(line))

			// Find the position of the dollar sign
			dollarPos := strings.Index(line, "$")
			violation = violation.WithColumn(dollarPos + 1)

			// Add fix information - remove "$ "
			fixInfo := value.NewFixInfo().
				WithLineNumber(lineNumber).
				WithEditColumn(dollarPos + 1).
				WithDeleteLength(2). // Remove "$ "
				WithReplaceText("")

			violation = violation.WithFixInfo(*fixInfo)
			violations = append(violations, *violation)
		}
	}

	return violations
}

// checkIndentedCodeLine checks an indented code block line for dollar sign violations
func checkIndentedCodeLine(line string, lineNumber int) []value.Violation {
	var violations []value.Violation

	// Extract the code content (after indentation)
	content := line
	if strings.HasPrefix(line, "    ") {
		content = line[4:]
	} else if strings.HasPrefix(line, "\t") {
		content = line[1:]
	}

	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, "$ ") {
		// This might be a violation - but we need context to be sure
		// For indented code blocks, it's harder to determine context
		// We'll be conservative and only flag obvious cases

		// Simple heuristic: if it's a standalone indented line starting with $
		violation := value.NewViolation(
			[]string{"MD014", "commands-show-output"},
			"Dollar signs used before commands without showing output",
			nil,
			lineNumber,
		)

		violation = violation.WithErrorDetail("Unnecessary dollar sign in code block")
		violation = violation.WithErrorContext(trimmed)

		violations = append(violations, *violation)
	}

	return violations
}
