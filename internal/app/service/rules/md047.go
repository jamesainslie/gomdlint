package rules

import (
	"context"
	"net/url"
	"strings"

	"github.com/gomdlint/gomdlint/internal/domain/entity"
	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// MD047 - Files should end with a single newline character
func NewMD047Rule() functional.Result[*entity.Rule] {
	infoURL, _ := url.Parse("https://github.com/gomdlint/gomdlint/blob/main/docs/rules/md047.md")

	return entity.NewRule(
		[]string{"MD047", "single-trailing-newline"},
		"Files should end with a single newline character",
		[]string{"blank_lines"},
		infoURL,
		"commonmark",
		map[string]interface{}{},
		md047Function,
	)
}

func md047Function(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	var violations []value.Violation

	if len(params.Lines) == 0 {
		// Empty file is valid - no violation needed
		return functional.Ok(violations)
	}

	lastLineIndex := len(params.Lines) - 1
	lastLine := params.Lines[lastLineIndex]

	// Check if file ends with newline by looking at the last line
	// In Go's strings.Split(), if the original string ends with the separator,
	// the last element will be empty

	// Also check the raw content to be more accurate
	// This is a heuristic - in a real implementation, you'd want to check
	// the original file content directly
	fileEndsProperlyWithNewline := false

	if lastLineIndex > 0 {
		// If we have multiple lines and the last line is empty,
		// it likely means the file ended with a newline
		fileEndsProperlyWithNewline = lastLine == ""
	} else if lastLine == "" {
		// Single line that is empty means the file was just "\n"
		fileEndsProperlyWithNewline = true
	} else {
		// Single line file with content - check if it ends with newline
		fileEndsProperlyWithNewline = strings.HasSuffix(lastLine, "\n")
	}

	if !fileEndsProperlyWithNewline {
		lineNumber := lastLineIndex + 1
		if lastLine == "" && lastLineIndex > 0 {
			lineNumber = lastLineIndex // Point to the previous line
		}

		violation := value.NewViolation(
			[]string{"MD047", "single-trailing-newline"},
			"Files should end with a single newline character",
			nil,
			lineNumber,
		)

		violation = violation.WithErrorDetail("File should end with a newline character")
		violation = violation.WithErrorContext(lastLine)

		// Add fix information - add newline at end of file
		fixInfo := value.NewFixInfo().
			WithLineNumber(lineNumber).
			WithEditColumn(len(lastLine) + 1). // Position at end of last line
			WithDeleteLength(0).               // Don't delete anything
			WithReplaceText("\n")              // Add newline

		violation = violation.WithFixInfo(*fixInfo)
		violations = append(violations, *violation)
	}

	return functional.Ok(violations)
}
