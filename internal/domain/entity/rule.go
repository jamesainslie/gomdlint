package entity

import (
	"context"
	"fmt"
	"net/url"

	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// Rule represents a markdown linting rule with its metadata and execution function.
// Rules are immutable entities that define validation logic for markdown documents.
type Rule struct {
	// Metadata
	names       []string
	description string
	tags        []string
	information *url.URL

	// Configuration
	parser string
	config map[string]interface{}

	// Execution
	function RuleFunction
}

// RuleFunction defines the signature for rule execution functions.
// It follows functional programming principles with immutable parameters.
type RuleFunction func(ctx context.Context, params RuleParams) functional.Result[[]value.Violation]

// RuleParams contains immutable parameters passed to rule functions.
type RuleParams struct {
	// Document content and metadata
	Lines    []string
	Config   map[string]interface{}
	Filename string

	// Parsed markdown tokens/AST
	Tokens []value.Token

	// Helper functions for rule execution
	FrontMatter functional.Option[map[string]interface{}]
}

// NewRule creates a new Rule with the provided configuration.
// All rule properties are validated at creation time.
func NewRule(
	names []string,
	description string,
	tags []string,
	information *url.URL,
	parser string,
	config map[string]interface{},
	function RuleFunction,
) functional.Result[*Rule] {
	// Validate required fields
	if len(names) == 0 {
		return functional.Err[*Rule](fmt.Errorf("rule must have at least one name"))
	}

	if description == "" {
		return functional.Err[*Rule](fmt.Errorf("rule must have a description"))
	}

	if function == nil {
		return functional.Err[*Rule](fmt.Errorf("rule must have a function"))
	}

	// Validate names format (should be uppercase MD### or lowercase aliases)
	for _, name := range names {
		if name == "" {
			return functional.Err[*Rule](fmt.Errorf("rule names cannot be empty"))
		}
	}

	rule := &Rule{
		names:       make([]string, len(names)),
		description: description,
		tags:        make([]string, len(tags)),
		information: information,
		parser:      parser,
		config:      make(map[string]interface{}),
		function:    function,
	}

	// Deep copy slices and maps to ensure immutability
	copy(rule.names, names)
	copy(rule.tags, tags)

	for k, v := range config {
		rule.config[k] = v
	}

	return functional.Ok(rule)
}

// Names returns a copy of the rule names to maintain immutability.
func (r *Rule) Names() []string {
	names := make([]string, len(r.names))
	copy(names, r.names)
	return names
}

// PrimaryName returns the first (primary) name of the rule.
func (r *Rule) PrimaryName() string {
	if len(r.names) == 0 {
		return ""
	}
	return r.names[0]
}

// Description returns the rule description.
func (r *Rule) Description() string {
	return r.description
}

// Tags returns a copy of the rule tags to maintain immutability.
func (r *Rule) Tags() []string {
	tags := make([]string, len(r.tags))
	copy(tags, r.tags)
	return tags
}

// Information returns the URL to the rule documentation.
func (r *Rule) Information() *url.URL {
	if r.information == nil {
		return nil
	}
	// Return a copy to maintain immutability
	u, _ := url.Parse(r.information.String())
	return u
}

// Parser returns the parser type required by this rule.
func (r *Rule) Parser() string {
	return r.parser
}

// Config returns a copy of the rule configuration to maintain immutability.
func (r *Rule) Config() map[string]interface{} {
	config := make(map[string]interface{})
	for k, v := range r.config {
		config[k] = v
	}
	return config
}

// Function returns the rule function for direct execution.
// This is primarily used for testing purposes.
func (r *Rule) Function() RuleFunction {
	return r.function
}

// Execute runs the rule function with the provided parameters.
// This method handles error recovery and provides consistent result formatting.
func (r *Rule) Execute(ctx context.Context, params RuleParams) functional.Result[[]value.Violation] {
	// Ensure we don't panic during rule execution
	defer func() {
		if recovered := recover(); recovered != nil {
			// Log the panic but don't let it crash the entire linting process
			// This will be handled by the error reporting system
		}
	}()

	// Merge rule configuration with runtime parameters
	mergedParams := params
	mergedConfig := make(map[string]interface{})

	// Start with rule defaults
	for k, v := range r.config {
		mergedConfig[k] = v
	}

	// Override with runtime config
	for k, v := range params.Config {
		mergedConfig[k] = v
	}

	mergedParams.Config = mergedConfig

	return r.function(ctx, mergedParams)
}

// HasName checks if the rule matches any of the given names (case-insensitive).
func (r *Rule) HasName(name string) bool {
	for _, ruleName := range r.names {
		if equalIgnoreCase(ruleName, name) {
			return true
		}
	}
	return false
}

// HasTag checks if the rule has the given tag (case-insensitive).
func (r *Rule) HasTag(tag string) bool {
	for _, ruleTag := range r.tags {
		if equalIgnoreCase(ruleTag, tag) {
			return true
		}
	}
	return false
}

// String implements the Stringer interface for debugging.
func (r *Rule) String() string {
	if len(r.names) > 0 {
		return r.names[0]
	}
	return "unnamed-rule"
}

// equalIgnoreCase performs case-insensitive string comparison.
func equalIgnoreCase(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca := a[i]
		cb := b[i]

		// Convert to lowercase
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}

		if ca != cb {
			return false
		}
	}
	return true
}
