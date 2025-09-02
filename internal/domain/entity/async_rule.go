package entity

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// AsyncRuleFunction for rules that need async execution
type AsyncRuleFunction func(ctx context.Context, params RuleParams) <-chan AsyncRuleResult

// AsyncRuleResult contains the result of async rule execution
type AsyncRuleResult struct {
	Violations []value.Violation
	Error      error
	Metadata   map[string]interface{} // For timing, cache info, etc.
}

// AsyncRule wraps a rule with async capabilities
type AsyncRule struct {
	*Rule
	asyncFunction AsyncRuleFunction
	timeout       time.Duration
}

// NewAsyncRule creates a rule that can execute asynchronously
func NewAsyncRule(
	names []string,
	description string,
	tags []string,
	information *url.URL,
	parser string,
	config map[string]interface{},
	function AsyncRuleFunction,
	timeout time.Duration,
) functional.Result[*AsyncRule] {
	// Create base rule with a sync wrapper
	syncWrapper := func(ctx context.Context, params RuleParams) functional.Result[[]value.Violation] {
		resultChan := function(ctx, params)

		select {
		case result := <-resultChan:
			if result.Error != nil {
				return functional.Err[[]value.Violation](result.Error)
			}
			return functional.Ok(result.Violations)
		case <-ctx.Done():
			return functional.Err[[]value.Violation](ctx.Err())
		case <-time.After(timeout):
			return functional.Err[[]value.Violation](fmt.Errorf("rule execution timeout after %v", timeout))
		}
	}

	baseRule := NewRule(names, description, tags, information, parser, config, syncWrapper)
	if baseRule.IsErr() {
		return functional.Err[*AsyncRule](baseRule.Error())
	}

	return functional.Ok(&AsyncRule{
		Rule:          baseRule.Unwrap(),
		asyncFunction: function,
		timeout:       timeout,
	})
}

// ExecuteAsync runs the rule asynchronously
func (ar *AsyncRule) ExecuteAsync(ctx context.Context, params RuleParams) <-chan AsyncRuleResult {
	return ar.asyncFunction(ctx, params)
}

// GetTimeout returns the execution timeout
func (ar *AsyncRule) GetTimeout() time.Duration {
	return ar.timeout
}

// SetTimeout updates the execution timeout
func (ar *AsyncRule) SetTimeout(timeout time.Duration) {
	ar.timeout = timeout
}

// IsAsync returns true since this is an async rule
func (ar *AsyncRule) IsAsync() bool {
	return true
}

// AsyncRuleBuilder helps build async rules with a fluent interface
type AsyncRuleBuilder struct {
	names       []string
	description string
	tags        []string
	information *url.URL
	parser      string
	config      map[string]interface{}
	function    AsyncRuleFunction
	timeout     time.Duration
}

// NewAsyncRuleBuilder creates a new async rule builder
func NewAsyncRuleBuilder() *AsyncRuleBuilder {
	return &AsyncRuleBuilder{
		names:   make([]string, 0),
		tags:    make([]string, 0),
		config:  make(map[string]interface{}),
		timeout: 30 * time.Second, // Default timeout
	}
}

// WithNames sets the rule names
func (arb *AsyncRuleBuilder) WithNames(names ...string) *AsyncRuleBuilder {
	arb.names = names
	return arb
}

// WithDescription sets the rule description
func (arb *AsyncRuleBuilder) WithDescription(description string) *AsyncRuleBuilder {
	arb.description = description
	return arb
}

// WithTags sets the rule tags
func (arb *AsyncRuleBuilder) WithTags(tags ...string) *AsyncRuleBuilder {
	arb.tags = tags
	return arb
}

// WithInformation sets the rule information URL
func (arb *AsyncRuleBuilder) WithInformation(information *url.URL) *AsyncRuleBuilder {
	arb.information = information
	return arb
}

// WithParser sets the required parser
func (arb *AsyncRuleBuilder) WithParser(parser string) *AsyncRuleBuilder {
	arb.parser = parser
	return arb
}

// WithConfig sets the rule configuration
func (arb *AsyncRuleBuilder) WithConfig(config map[string]interface{}) *AsyncRuleBuilder {
	arb.config = config
	return arb
}

// WithFunction sets the async rule function
func (arb *AsyncRuleBuilder) WithFunction(function AsyncRuleFunction) *AsyncRuleBuilder {
	arb.function = function
	return arb
}

// WithTimeout sets the execution timeout
func (arb *AsyncRuleBuilder) WithTimeout(timeout time.Duration) *AsyncRuleBuilder {
	arb.timeout = timeout
	return arb
}

// Build creates the async rule
func (arb *AsyncRuleBuilder) Build() functional.Result[*AsyncRule] {
	if len(arb.names) == 0 {
		return functional.Err[*AsyncRule](fmt.Errorf("rule must have at least one name"))
	}

	if arb.description == "" {
		return functional.Err[*AsyncRule](fmt.Errorf("rule must have a description"))
	}

	if arb.function == nil {
		return functional.Err[*AsyncRule](fmt.Errorf("rule must have an async function"))
	}

	return NewAsyncRule(
		arb.names,
		arb.description,
		arb.tags,
		arb.information,
		arb.parser,
		arb.config,
		arb.function,
		arb.timeout,
	)
}

// AsyncRuleRegistry manages async rules
type AsyncRuleRegistry struct {
	rules map[string]*AsyncRule
}

// NewAsyncRuleRegistry creates a new async rule registry
func NewAsyncRuleRegistry() *AsyncRuleRegistry {
	return &AsyncRuleRegistry{
		rules: make(map[string]*AsyncRule),
	}
}

// RegisterRule registers an async rule
func (arr *AsyncRuleRegistry) RegisterRule(rule *AsyncRule) error {
	primaryName := rule.PrimaryName()
	if _, exists := arr.rules[primaryName]; exists {
		return fmt.Errorf("async rule %s already registered", primaryName)
	}

	arr.rules[primaryName] = rule
	return nil
}

// GetRule retrieves an async rule by name
func (arr *AsyncRuleRegistry) GetRule(name string) (*AsyncRule, error) {
	rule, exists := arr.rules[name]
	if !exists {
		return nil, fmt.Errorf("async rule %s not found", name)
	}

	return rule, nil
}

// GetAllRules returns all registered async rules
func (arr *AsyncRuleRegistry) GetAllRules() []*AsyncRule {
	rules := make([]*AsyncRule, 0, len(arr.rules))
	for _, rule := range arr.rules {
		rules = append(rules, rule)
	}
	return rules
}

// ListRuleNames returns all rule names
func (arr *AsyncRuleRegistry) ListRuleNames() []string {
	names := make([]string, 0, len(arr.rules))
	for name := range arr.rules {
		names = append(names, name)
	}
	return names
}
