package value

import (
	"fmt"
	"regexp"

	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// LintOptions represents the configuration for linting operations.
// This mirrors the markdownlint options structure for compatibility.
type LintOptions struct {
	// Input sources
	Files   []string          // List of files to lint
	Strings map[string]string // Map of identifier to content for string inputs

	// Rule configuration
	Config map[string]interface{} // Rule configuration map

	// Parser configuration
	FrontMatter    functional.Option[*regexp.Regexp] // Front matter detection regex
	NoInlineConfig bool                              // Disable inline config comments
	ResultVersion  int                               // Result format version (default: 3)

	// Rule customization
	CustomRules   []interface{}  // Custom rule definitions
	ConfigParsers []ConfigParser // Config parsers for inline comments

	// Error handling
	HandleRuleFailures bool // Catch and report rule execution errors

	// Theming configuration
	Theme ThemeConfig // Theme configuration for output formatting
}

// ConfigParser is a function type that parses configuration content.
type ConfigParser func(content string) (map[string]interface{}, error)

// RuleConfig represents configuration for a specific rule.
type RuleConfig struct {
	Enabled bool                   // Whether the rule is enabled
	Config  map[string]interface{} // Rule-specific configuration
}

// NewLintOptions creates a new LintOptions with sensible defaults.
func NewLintOptions() *LintOptions {
	return &LintOptions{
		Files:              make([]string, 0),
		Strings:            make(map[string]string),
		Config:             make(map[string]interface{}),
		FrontMatter:        functional.Some(DefaultFrontMatterRegex()),
		NoInlineConfig:     false,
		ResultVersion:      3,
		CustomRules:        make([]interface{}, 0),
		ConfigParsers:      make([]ConfigParser, 0),
		HandleRuleFailures: false,
		Theme:              NewThemeConfig(),
	}
}

// WithFiles sets the files to be linted.
func (o *LintOptions) WithFiles(files []string) *LintOptions {
	newOptions := *o
	newOptions.Files = make([]string, len(files))
	copy(newOptions.Files, files)
	return &newOptions
}

// WithStrings sets the string content to be linted.
func (o *LintOptions) WithStrings(strings map[string]string) *LintOptions {
	newOptions := *o
	newOptions.Strings = make(map[string]string)
	for k, v := range strings {
		newOptions.Strings[k] = v
	}
	return &newOptions
}

// WithConfig sets the rule configuration.
func (o *LintOptions) WithConfig(config map[string]interface{}) *LintOptions {
	newOptions := *o
	newOptions.Config = make(map[string]interface{})
	for k, v := range config {
		newOptions.Config[k] = v
	}
	return &newOptions
}

// WithFrontMatter sets the front matter detection regex.
func (o *LintOptions) WithFrontMatter(regex *regexp.Regexp) *LintOptions {
	newOptions := *o
	if regex == nil {
		newOptions.FrontMatter = functional.None[*regexp.Regexp]()
	} else {
		newOptions.FrontMatter = functional.Some(regex)
	}
	return &newOptions
}

// WithNoInlineConfig disables inline configuration comments.
func (o *LintOptions) WithNoInlineConfig(disable bool) *LintOptions {
	newOptions := *o
	newOptions.NoInlineConfig = disable
	return &newOptions
}

// WithResultVersion sets the result format version.
func (o *LintOptions) WithResultVersion(version int) *LintOptions {
	newOptions := *o
	newOptions.ResultVersion = version
	return &newOptions
}

// WithCustomRules adds custom rules to the configuration.
func (o *LintOptions) WithCustomRules(rules []interface{}) *LintOptions {
	newOptions := *o
	newOptions.CustomRules = make([]interface{}, len(rules))
	copy(newOptions.CustomRules, rules)
	return &newOptions
}

// WithConfigParsers sets the configuration parsers.
func (o *LintOptions) WithConfigParsers(parsers []ConfigParser) *LintOptions {
	newOptions := *o
	newOptions.ConfigParsers = make([]ConfigParser, len(parsers))
	copy(newOptions.ConfigParsers, parsers)
	return &newOptions
}

// WithHandleRuleFailures enables/disables rule failure handling.
func (o *LintOptions) WithHandleRuleFailures(handle bool) *LintOptions {
	newOptions := *o
	newOptions.HandleRuleFailures = handle
	return &newOptions
}

// WithTheme sets the theme configuration.
func (o *LintOptions) WithTheme(theme ThemeConfig) *LintOptions {
	newOptions := *o
	newOptions.Theme = theme
	return &newOptions
}

// WithThemeName sets the theme name.
func (o *LintOptions) WithThemeName(name string) *LintOptions {
	newOptions := *o
	newOptions.Theme.ThemeName = name
	return &newOptions
}

// WithSuppressEmojis enables or disables emoji suppression.
func (o *LintOptions) WithSuppressEmojis(suppress bool) *LintOptions {
	newOptions := *o
	newOptions.Theme.SuppressEmojis = suppress
	return &newOptions
}

// WithCustomSymbols sets custom symbol overrides.
func (o *LintOptions) WithCustomSymbols(symbols map[string]string) *LintOptions {
	newOptions := *o
	newOptions.Theme.CustomSymbols = make(map[string]string)
	for k, v := range symbols {
		newOptions.Theme.CustomSymbols[k] = v
	}
	return &newOptions
}

// HasInput returns true if there are files or strings to lint.
func (o *LintOptions) HasInput() bool {
	return len(o.Files) > 0 || len(o.Strings) > 0
}

// GetRuleConfig returns the configuration for a specific rule.
func (o *LintOptions) GetRuleConfig(ruleName string) RuleConfig {
	// Check if the rule is explicitly configured
	if configValue, exists := o.Config[ruleName]; exists {
		switch v := configValue.(type) {
		case bool:
			return RuleConfig{
				Enabled: v,
				Config:  make(map[string]interface{}),
			}
		case map[string]interface{}:
			return RuleConfig{
				Enabled: true,
				Config:  v,
			}
		}
	}

	// Check for default configuration
	if defaultValue, exists := o.Config["default"]; exists {
		if enabled, ok := defaultValue.(bool); ok {
			return RuleConfig{
				Enabled: enabled,
				Config:  make(map[string]interface{}),
			}
		}
	}

	// Default: enabled with no specific config
	return RuleConfig{
		Enabled: true,
		Config:  make(map[string]interface{}),
	}
}

// DefaultFrontMatterRegex returns the default front matter detection regex.
// This matches YAML, TOML, and JSON front matter patterns.
func DefaultFrontMatterRegex() *regexp.Regexp {
	// This regex matches the default markdownlint front matter pattern
	pattern := `(?m)^((---[^\S\r\n]*$[\s\S]+?^---\s*)|(^\+\+\+[^\S\r\n]*$[\s\S]+?^(\+\+\+|\.\.\.)\s*)|(^\{[^\S\r\n]*$[\s\S]+?^\}\s*))(\r\n|\r|\n|$)`

	regex, err := regexp.Compile(pattern)
	if err != nil {
		// Fallback to a simpler pattern if the complex one fails
		regex = regexp.MustCompile(`(?m)^---[\s\S]*?^---\s*$`)
	}

	return regex
}

// LintResult represents the result of a linting operation.
type LintResult struct {
	// Results by file/string identifier
	Results map[string][]Violation

	// Summary information
	TotalViolations int
	TotalFiles      int
	TotalErrors     int
	TotalWarnings   int
}

// NewLintResult creates a new empty LintResult.
func NewLintResult() *LintResult {
	return &LintResult{
		Results:         make(map[string][]Violation),
		TotalViolations: 0,
		TotalFiles:      0,
		TotalErrors:     0,
		TotalWarnings:   0,
	}
}

// AddViolations adds violations for a specific file/identifier.
func (r *LintResult) AddViolations(identifier string, violations []Violation) {
	r.Results[identifier] = violations
	r.TotalFiles++
	r.TotalViolations += len(violations)

	// Count errors and warnings
	for _, violation := range violations {
		switch violation.Severity {
		case SeverityError:
			r.TotalErrors++
		case SeverityWarning:
			r.TotalWarnings++
		}
	}
}

// HasViolations returns true if there are any violations.
func (r *LintResult) HasViolations() bool {
	return r.TotalViolations > 0
}

// HasErrors returns true if there are any error-level violations.
func (r *LintResult) HasErrors() bool {
	return r.TotalErrors > 0
}

// GetViolations returns violations for a specific identifier.
func (r *LintResult) GetViolations(identifier string) []Violation {
	return r.Results[identifier]
}

// GetAllViolations returns all violations across all files.
func (r *LintResult) GetAllViolations() []Violation {
	var allViolations []Violation
	for _, violations := range r.Results {
		allViolations = append(allViolations, violations...)
	}
	return allViolations
}

// String implements the Stringer interface for formatted output.
func (r *LintResult) String() string {
	return r.ToFormattedString(false)
}

// ToFormattedString returns a formatted string representation.
// If useAliases is true, rule aliases are used instead of MD### names.
func (r *LintResult) ToFormattedString(useAliases bool) string {
	if r.TotalViolations == 0 {
		return ""
	}

	var output string
	for identifier, violations := range r.Results {
		for _, violation := range violations {
			if output != "" {
				output += "\n"
			}

			ruleName := violation.PrimaryRuleName()
			if useAliases && len(violation.RuleNames) > 1 {
				// Use the first alias if available
				for i := 1; i < len(violation.RuleNames); i++ {
					name := violation.RuleNames[i]
					if len(name) < 3 || name[:2] != "MD" {
						ruleName = name
						break
					}
				}
			}

			location := violation.GetLocation()
			description := violation.RuleDescription
			detail := violation.GetDetailString()

			if detail != "" {
				output += fmt.Sprintf("%s: %s: %s %s %s",
					identifier, location, ruleName, description, detail)
			} else {
				output += fmt.Sprintf("%s: %s: %s %s",
					identifier, location, ruleName, description)
			}
		}
	}

	return output
}

// ToJSON returns the result in JSON format compatible with markdownlint.
func (r *LintResult) ToJSON() map[string]interface{} {
	result := make(map[string]interface{})

	for identifier, violations := range r.Results {
		violationData := make([]map[string]interface{}, len(violations))
		for i, violation := range violations {
			violationData[i] = violation.ToMarkdownlintFormat()
		}
		result[identifier] = violationData
	}

	return result
}
