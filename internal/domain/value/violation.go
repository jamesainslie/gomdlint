package value

import (
	"fmt"
	"net/url"

	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// Severity represents the severity level of a violation.
type Severity int

const (
	SeverityError Severity = iota
	SeverityWarning
	SeverityInfo
)

// String returns the string representation of the severity.
func (s Severity) String() string {
	switch s {
	case SeverityError:
		return "error"
	case SeverityWarning:
		return "warning"
	case SeverityInfo:
		return "info"
	default:
		return "unknown"
	}
}

// FixInfo represents information needed to automatically fix a violation.
// This follows the markdownlint fixInfo structure for compatibility.
type FixInfo struct {
	// Line-based fixes
	LineNumber  functional.Option[int]    // 1-based line number to fix
	DeleteCount functional.Option[int]    // Number of lines to delete
	InsertText  functional.Option[string] // Text to insert

	// Column-based fixes (within a line)
	EditColumn   functional.Option[int]    // 1-based column to start edit
	DeleteLength functional.Option[int]    // Number of characters to delete
	ReplaceText  functional.Option[string] // Text to replace with
}

// NewFixInfo creates a new FixInfo with the specified parameters.
func NewFixInfo() *FixInfo {
	return &FixInfo{
		LineNumber:   functional.None[int](),
		DeleteCount:  functional.None[int](),
		InsertText:   functional.None[string](),
		EditColumn:   functional.None[int](),
		DeleteLength: functional.None[int](),
		ReplaceText:  functional.None[string](),
	}
}

// WithLineNumber sets the line number for the fix.
func (f *FixInfo) WithLineNumber(lineNumber int) *FixInfo {
	newFix := *f
	newFix.LineNumber = functional.Some(lineNumber)
	return &newFix
}

// WithDeleteCount sets the number of lines to delete.
func (f *FixInfo) WithDeleteCount(count int) *FixInfo {
	newFix := *f
	newFix.DeleteCount = functional.Some(count)
	return &newFix
}

// WithInsertText sets the text to insert.
func (f *FixInfo) WithInsertText(text string) *FixInfo {
	newFix := *f
	newFix.InsertText = functional.Some(text)
	return &newFix
}

// WithEditColumn sets the column for character-level edits.
func (f *FixInfo) WithEditColumn(column int) *FixInfo {
	newFix := *f
	newFix.EditColumn = functional.Some(column)
	return &newFix
}

// WithDeleteLength sets the number of characters to delete.
func (f *FixInfo) WithDeleteLength(length int) *FixInfo {
	newFix := *f
	newFix.DeleteLength = functional.Some(length)
	return &newFix
}

// WithReplaceText sets the replacement text for character-level edits.
func (f *FixInfo) WithReplaceText(text string) *FixInfo {
	newFix := *f
	newFix.ReplaceText = functional.Some(text)
	return &newFix
}

// IsLineFix returns true if this is a line-based fix.
func (f *FixInfo) IsLineFix() bool {
	return f.LineNumber.IsSome()
}

// IsColumnFix returns true if this is a column-based fix.
func (f *FixInfo) IsColumnFix() bool {
	return f.EditColumn.IsSome()
}

// Violation represents a rule violation found in markdown content.
// Violations are immutable value objects that describe issues and potential fixes.
type Violation struct {
	// Rule identification
	RuleNames       []string // All names/aliases for the rule
	RuleDescription string   // Human-readable description
	RuleInformation *url.URL // URL to rule documentation

	// Location in document
	LineNumber   int                    // 1-based line number
	ColumnNumber functional.Option[int] // 1-based column number (if applicable)
	Length       functional.Option[int] // Length of the problematic text

	// Violation details
	Severity     Severity                  // Severity level
	ErrorDetail  functional.Option[string] // Additional detail about the error
	ErrorContext functional.Option[string] // Context text showing the problem
	ErrorRange   functional.Option[Range]  // Precise range of the error

	// Fix information
	FixInfo functional.Option[FixInfo] // Auto-fix information
}

// NewViolation creates a new Violation with the specified rule and location.
func NewViolation(
	ruleNames []string,
	ruleDescription string,
	ruleInformation *url.URL,
	lineNumber int,
) *Violation {
	// Ensure we have a copy of rule names for immutability
	names := make([]string, len(ruleNames))
	copy(names, ruleNames)

	return &Violation{
		RuleNames:       names,
		RuleDescription: ruleDescription,
		RuleInformation: ruleInformation,
		LineNumber:      lineNumber,
		ColumnNumber:    functional.None[int](),
		Length:          functional.None[int](),
		Severity:        SeverityError, // Default severity
		ErrorDetail:     functional.None[string](),
		ErrorContext:    functional.None[string](),
		ErrorRange:      functional.None[Range](),
		FixInfo:         functional.None[FixInfo](),
	}
}

// WithColumn sets the column number for the violation.
func (v *Violation) WithColumn(column int) *Violation {
	newV := *v
	newV.ColumnNumber = functional.Some(column)
	return &newV
}

// WithLength sets the length of the problematic text.
func (v *Violation) WithLength(length int) *Violation {
	newV := *v
	newV.Length = functional.Some(length)
	return &newV
}

// WithSeverity sets the severity level.
func (v *Violation) WithSeverity(severity Severity) *Violation {
	newV := *v
	newV.Severity = severity
	return &newV
}

// WithErrorDetail sets additional error detail.
func (v *Violation) WithErrorDetail(detail string) *Violation {
	newV := *v
	newV.ErrorDetail = functional.Some(detail)
	return &newV
}

// WithErrorContext sets the error context.
func (v *Violation) WithErrorContext(context string) *Violation {
	newV := *v
	newV.ErrorContext = functional.Some(context)
	return &newV
}

// WithErrorRange sets the precise error range.
func (v *Violation) WithErrorRange(errorRange Range) *Violation {
	newV := *v
	newV.ErrorRange = functional.Some(errorRange)
	return &newV
}

// WithFixInfo sets the auto-fix information.
func (v *Violation) WithFixInfo(fixInfo FixInfo) *Violation {
	newV := *v
	newV.FixInfo = functional.Some(fixInfo)
	return &newV
}

// PrimaryRuleName returns the primary (first) rule name.
func (v *Violation) PrimaryRuleName() string {
	if len(v.RuleNames) > 0 {
		return v.RuleNames[0]
	}
	return "unknown-rule"
}

// IsFixable returns true if the violation can be automatically fixed.
func (v *Violation) IsFixable() bool {
	return v.FixInfo.IsSome()
}

// GetLocation returns a human-readable location string.
func (v *Violation) GetLocation() string {
	if v.ColumnNumber.IsSome() {
		return fmt.Sprintf("%d:%d", v.LineNumber, v.ColumnNumber.Unwrap())
	}
	return fmt.Sprintf("%d", v.LineNumber)
}

// GetDetailString returns a formatted detail string for display.
func (v *Violation) GetDetailString() string {
	var parts []string

	// Add error detail if present
	if v.ErrorDetail.IsSome() {
		parts = append(parts, v.ErrorDetail.Unwrap())
	}

	// Add context if present
	if v.ErrorContext.IsSome() {
		context := v.ErrorContext.Unwrap()
		if len(context) > 0 {
			parts = append(parts, fmt.Sprintf("[Context: %q]", context))
		}
	}

	// Add column information if available
	if v.ColumnNumber.IsSome() {
		column := v.ColumnNumber.Unwrap()
		if v.Length.IsSome() {
			length := v.Length.Unwrap()
			parts = append(parts, fmt.Sprintf("[Column: %d, Length: %d]", column, length))
		} else {
			parts = append(parts, fmt.Sprintf("[Column: %d]", column))
		}
	}

	if len(parts) == 0 {
		return ""
	}

	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += " " + parts[i]
	}

	return result
}

// String implements the Stringer interface for debugging and display.
func (v *Violation) String() string {
	location := v.GetLocation()
	ruleName := v.PrimaryRuleName()
	description := v.RuleDescription

	detail := v.GetDetailString()
	if detail != "" {
		return fmt.Sprintf("%s: %s/%s %s %s",
			location, ruleName, v.getRuleAlias(), description, detail)
	}

	return fmt.Sprintf("%s: %s/%s %s",
		location, ruleName, v.getRuleAlias(), description)
}

// getRuleAlias returns the first non-MD### alias, or the primary name if none exists.
func (v *Violation) getRuleAlias() string {
	if len(v.RuleNames) < 2 {
		return v.PrimaryRuleName()
	}

	// Look for the first alias (non-MD### name)
	for i := 1; i < len(v.RuleNames); i++ {
		name := v.RuleNames[i]
		if len(name) < 3 || name[:2] != "MD" {
			return name
		}
	}

	// Fall back to primary name
	return v.PrimaryRuleName()
}

// ToMarkdownlintFormat returns the violation in markdownlint-compatible format.
func (v *Violation) ToMarkdownlintFormat() map[string]interface{} {
	result := map[string]interface{}{
		"lineNumber":      v.LineNumber,
		"ruleNames":       v.RuleNames,
		"ruleDescription": v.RuleDescription,
		"errorDetail":     v.ErrorDetail.UnwrapOr(""),
		"errorContext":    v.ErrorContext.UnwrapOr(""),
		"errorRange":      nil,
	}

	if v.RuleInformation != nil {
		result["ruleInformation"] = v.RuleInformation.String()
	}

	if v.ErrorRange.IsSome() {
		errorRange := v.ErrorRange.Unwrap()
		result["errorRange"] = []int{errorRange.Start.Column, errorRange.End.Column - errorRange.Start.Column}
	}

	if v.FixInfo.IsSome() {
		fixInfo := v.FixInfo.Unwrap()
		fixData := make(map[string]interface{})

		if fixInfo.LineNumber.IsSome() {
			fixData["lineNumber"] = fixInfo.LineNumber.Unwrap()
		}
		if fixInfo.DeleteCount.IsSome() {
			fixData["deleteCount"] = fixInfo.DeleteCount.Unwrap()
		}
		if fixInfo.InsertText.IsSome() {
			fixData["insertText"] = fixInfo.InsertText.Unwrap()
		}
		if fixInfo.EditColumn.IsSome() {
			fixData["editColumn"] = fixInfo.EditColumn.Unwrap()
		}
		if fixInfo.DeleteLength.IsSome() {
			fixData["deleteLength"] = fixInfo.DeleteLength.Unwrap()
		}
		if fixInfo.ReplaceText.IsSome() {
			fixData["replaceText"] = fixInfo.ReplaceText.Unwrap()
		}

		result["fixInfo"] = fixData
	}

	return result
}
