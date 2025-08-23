package gomdlint

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gomdlint/gomdlint/internal/app/service"
	"github.com/gomdlint/gomdlint/internal/domain/value"
)

// Version returns the version of the gomdlint library.
const Version = "1.0.0"

// LintOptions represents configuration options for linting operations.
// This provides a public interface compatible with markdownlint.
type LintOptions struct {
	// Input sources
	Files   []string          `json:"files,omitempty"`
	Strings map[string]string `json:"strings,omitempty"`

	// Rule configuration
	Config map[string]interface{} `json:"config,omitempty"`

	// Parser configuration
	FrontMatter        string `json:"frontMatter,omitempty"`        // Regex pattern for front matter
	NoInlineConfig     bool   `json:"noInlineConfig,omitempty"`     // Disable inline config
	ResultVersion      int    `json:"resultVersion,omitempty"`      // Result format version
	HandleRuleFailures bool   `json:"handleRuleFailures,omitempty"` // Handle rule failures

	// Custom rules and parsers
	CustomRules   []interface{} `json:"customRules,omitempty"`
	ConfigParsers []interface{} `json:"configParsers,omitempty"`
}

// LintResult represents the result of a linting operation.
type LintResult struct {
	// Violations by file/identifier
	Results map[string][]Violation `json:"results"`

	// Summary
	TotalViolations int `json:"totalViolations"`
	TotalFiles      int `json:"totalFiles"`
	TotalErrors     int `json:"totalErrors"`
	TotalWarnings   int `json:"totalWarnings"`
}

// Violation represents a single linting violation.
type Violation struct {
	LineNumber      int      `json:"lineNumber"`
	RuleNames       []string `json:"ruleNames"`
	RuleDescription string   `json:"ruleDescription"`
	RuleInformation string   `json:"ruleInformation,omitempty"`
	ErrorDetail     string   `json:"errorDetail,omitempty"`
	ErrorContext    string   `json:"errorContext,omitempty"`
	ErrorRange      []int    `json:"errorRange,omitempty"` // [column, length]
	FixInfo         *FixInfo `json:"fixInfo,omitempty"`
}

// FixInfo represents auto-fix information for a violation.
type FixInfo struct {
	LineNumber   *int    `json:"lineNumber,omitempty"`
	DeleteCount  *int    `json:"deleteCount,omitempty"`
	InsertText   *string `json:"insertText,omitempty"`
	EditColumn   *int    `json:"editColumn,omitempty"`
	DeleteLength *int    `json:"deleteLength,omitempty"`
	ReplaceText  *string `json:"replaceText,omitempty"`
}

// String returns a formatted string representation of the result.
func (lr *LintResult) String() string {
	return lr.ToFormattedString(false)
}

// ToFormattedString returns a formatted string representation.
// If useAliases is true, rule aliases are used instead of MD### names.
func (lr *LintResult) ToFormattedString(useAliases bool) string {
	if lr.TotalViolations == 0 {
		return ""
	}

	var output string
	for filename, violations := range lr.Results {
		for _, violation := range violations {
			if output != "" {
				output += "\n"
			}

			ruleName := violation.RuleNames[0]
			if useAliases && len(violation.RuleNames) > 1 {
				// Use first alias if available
				for i := 1; i < len(violation.RuleNames); i++ {
					name := violation.RuleNames[i]
					if len(name) < 3 || name[:2] != "MD" {
						ruleName = name
						break
					}
				}
			}

			location := fmt.Sprintf("%d", violation.LineNumber)
			description := violation.RuleDescription

			detail := ""
			if violation.ErrorDetail != "" {
				detail += violation.ErrorDetail
			}
			if violation.ErrorContext != "" {
				if detail != "" {
					detail += " "
				}
				detail += fmt.Sprintf("[Context: %q]", violation.ErrorContext)
			}
			if len(violation.ErrorRange) > 0 {
				if detail != "" {
					detail += " "
				}
				detail += fmt.Sprintf("[Column: %d]", violation.ErrorRange[0])
			}

			if detail != "" {
				output += fmt.Sprintf("%s: %s: %s %s %s", filename, location, ruleName, description, detail)
			} else {
				output += fmt.Sprintf("%s: %s: %s %s", filename, location, ruleName, description)
			}
		}
	}

	return output
}

// ToJSON returns the result as a JSON string.
func (lr *LintResult) ToJSON() (string, error) {
	data, err := json.Marshal(lr.Results)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Lint performs markdown linting with the given options.
// This is the main entry point for the library.
func Lint(ctx context.Context, options LintOptions) (*LintResult, error) {
	// Convert public options to internal format
	internalOptions := convertToInternalOptions(options)

	// Create linter service
	linter, err := service.NewLinterService(internalOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create linter: %w", err)
	}

	// Perform linting
	result := linter.Lint(ctx)
	if result.IsErr() {
		return nil, result.Error()
	}

	// Convert internal result to public format
	return convertToPublicResult(result.Unwrap()), nil
}

// LintString is a convenience function for linting a single string.
func LintString(ctx context.Context, content string, options ...LintOptions) (*LintResult, error) {
	var opts LintOptions
	if len(options) > 0 {
		opts = options[0]
	}

	opts.Strings = map[string]string{"content": content}
	return Lint(ctx, opts)
}

// LintFile is a convenience function for linting a single file.
func LintFile(ctx context.Context, filename string, options ...LintOptions) (*LintResult, error) {
	var opts LintOptions
	if len(options) > 0 {
		opts = options[0]
	}

	opts.Files = []string{filename}
	return Lint(ctx, opts)
}

// LintFiles is a convenience function for linting multiple files.
func LintFiles(ctx context.Context, filenames []string, options ...LintOptions) (*LintResult, error) {
	var opts LintOptions
	if len(options) > 0 {
		opts = options[0]
	}

	opts.Files = filenames
	return Lint(ctx, opts)
}

// GetVersion returns the version of the gomdlint library.
func GetVersion() string {
	return Version
}

// convertToInternalOptions converts public options to internal format.
func convertToInternalOptions(options LintOptions) *value.LintOptions {
	internalOptions := value.NewLintOptions().
		WithFiles(options.Files).
		WithStrings(options.Strings).
		WithConfig(options.Config).
		WithNoInlineConfig(options.NoInlineConfig).
		WithResultVersion(options.ResultVersion).
		WithHandleRuleFailures(options.HandleRuleFailures)

	// Handle front matter regex
	if options.FrontMatter != "" {
		// TODO: Compile regex pattern
		// For now, use default
	}

	return internalOptions
}

// convertToPublicResult converts internal result to public format.
func convertToPublicResult(internalResult *value.LintResult) *LintResult {
	result := &LintResult{
		Results:         make(map[string][]Violation),
		TotalViolations: internalResult.TotalViolations,
		TotalFiles:      internalResult.TotalFiles,
		TotalErrors:     internalResult.TotalErrors,
		TotalWarnings:   internalResult.TotalWarnings,
	}

	for identifier, violations := range internalResult.Results {
		publicViolations := make([]Violation, len(violations))
		for i, v := range violations {
			publicViolations[i] = convertToPublicViolation(v)
		}
		result.Results[identifier] = publicViolations
	}

	return result
}

// convertToPublicViolation converts internal violation to public format.
func convertToPublicViolation(v value.Violation) Violation {
	violation := Violation{
		LineNumber:      v.LineNumber,
		RuleNames:       v.RuleNames,
		RuleDescription: v.RuleDescription,
		ErrorDetail:     v.ErrorDetail.UnwrapOr(""),
		ErrorContext:    v.ErrorContext.UnwrapOr(""),
	}

	if v.RuleInformation != nil {
		violation.RuleInformation = v.RuleInformation.String()
	}

	if v.ErrorRange.IsSome() {
		errorRange := v.ErrorRange.Unwrap()
		violation.ErrorRange = []int{
			errorRange.Start.Column,
			errorRange.End.Column - errorRange.Start.Column,
		}
	}

	if v.FixInfo.IsSome() {
		fixInfo := v.FixInfo.Unwrap()
		publicFixInfo := &FixInfo{}

		if fixInfo.LineNumber.IsSome() {
			lineNum := fixInfo.LineNumber.Unwrap()
			publicFixInfo.LineNumber = &lineNum
		}
		if fixInfo.DeleteCount.IsSome() {
			deleteCount := fixInfo.DeleteCount.Unwrap()
			publicFixInfo.DeleteCount = &deleteCount
		}
		if fixInfo.InsertText.IsSome() {
			insertText := fixInfo.InsertText.Unwrap()
			publicFixInfo.InsertText = &insertText
		}
		if fixInfo.EditColumn.IsSome() {
			editColumn := fixInfo.EditColumn.Unwrap()
			publicFixInfo.EditColumn = &editColumn
		}
		if fixInfo.DeleteLength.IsSome() {
			deleteLength := fixInfo.DeleteLength.Unwrap()
			publicFixInfo.DeleteLength = &deleteLength
		}
		if fixInfo.ReplaceText.IsSome() {
			replaceText := fixInfo.ReplaceText.Unwrap()
			publicFixInfo.ReplaceText = &replaceText
		}

		violation.FixInfo = publicFixInfo
	}

	return violation
}
