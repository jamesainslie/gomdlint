package service

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/gomdlint/gomdlint/internal/app/service/rules"
	"github.com/gomdlint/gomdlint/internal/domain/entity"
	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// RuleEngine manages and executes markdown linting rules.
// It provides a plugin architecture for built-in and custom rules.
type RuleEngine struct {
	// Rule registry
	rules     []*entity.Rule
	ruleIndex map[string]*entity.Rule   // Index by name/alias for fast lookup
	tagIndex  map[string][]*entity.Rule // Index by tag for bulk operations

	// Configuration
	enabledRules map[string]bool
	ruleConfigs  map[string]map[string]interface{}

	// Performance
	mutex sync.RWMutex
}

// NewRuleEngine creates a new rule engine with all built-in rules registered.
func NewRuleEngine() (*RuleEngine, error) {
	engine := &RuleEngine{
		rules:        make([]*entity.Rule, 0),
		ruleIndex:    make(map[string]*entity.Rule),
		tagIndex:     make(map[string][]*entity.Rule),
		enabledRules: make(map[string]bool),
		ruleConfigs:  make(map[string]map[string]interface{}),
	}

	// Register all built-in rules
	if err := engine.registerBuiltInRules(); err != nil {
		return nil, fmt.Errorf("failed to register built-in rules: %w", err)
	}

	return engine, nil
}

// registerBuiltInRules registers all the built-in markdown linting rules.
func (re *RuleEngine) registerBuiltInRules() error {
	// Define core built-in rule constructors (reduced set for better performance)
	// Full compatibility with markdownlint's default enabled rules
	ruleConstructors := []func() functional.Result[*entity.Rule]{
		rules.NewMD001Rule, // heading-increment
		rules.NewMD003Rule, // heading-style
		rules.NewMD004Rule, // ul-style
		rules.NewMD005Rule, // list-indent
		rules.NewMD007Rule, // ul-indent
		rules.NewMD009Rule, // no-trailing-spaces
		rules.NewMD010Rule, // no-hard-tabs
		rules.NewMD011Rule, // no-reversed-links
		rules.NewMD012Rule, // no-multiple-blanks
		rules.NewMD013Rule, // line-length
		rules.NewMD018Rule, // no-missing-space-atx
		rules.NewMD019Rule, // no-multiple-space-atx
		rules.NewMD020Rule, // no-missing-space-closed-atx
		rules.NewMD021Rule, // no-multiple-space-closed-atx
		rules.NewMD022Rule, // blanks-around-headings
		rules.NewMD023Rule, // heading-start-left
		rules.NewMD024Rule, // no-duplicate-heading
		rules.NewMD025Rule, // single-h1
		rules.NewMD026Rule, // no-trailing-punctuation
		rules.NewMD027Rule, // no-multiple-space-blockquote
		rules.NewMD028Rule, // no-blanks-blockquote
		rules.NewMD029Rule, // ol-prefix
		rules.NewMD030Rule, // list-marker-space
		rules.NewMD031Rule, // blanks-around-fences
		rules.NewMD032Rule, // blanks-around-lists
		rules.NewMD033Rule, // no-inline-html
		rules.NewMD034Rule, // no-bare-urls
		rules.NewMD035Rule, // hr-style
		rules.NewMD036Rule, // no-emphasis-as-heading
		rules.NewMD037Rule, // no-space-in-emphasis
		rules.NewMD038Rule, // no-space-in-code
		rules.NewMD039Rule, // no-space-in-links
		rules.NewMD040Rule, // fenced-code-language
		rules.NewMD041Rule, // first-line-h1
		rules.NewMD042Rule, // no-empty-links
		rules.NewMD043Rule, // required-headings (disabled by default)
		rules.NewMD044Rule, // proper-names (disabled by default)
		rules.NewMD045Rule, // no-alt-text
		rules.NewMD046Rule, // code-block-style
		rules.NewMD047Rule, // single-trailing-newline
		rules.NewMD048Rule, // code-fence-style
		rules.NewMD049Rule, // emphasis-style
		rules.NewMD050Rule, // strong-style
	}

	// Optional rules (disabled by default to match markdownlint behavior)
	optionalRuleConstructors := []func() functional.Result[*entity.Rule]{
		rules.NewMD014Rule, // commands-show-output
		rules.NewMD051Rule, // link-fragments
		rules.NewMD052Rule, // reference-links-images
		rules.NewMD053Rule, // link-image-reference-definitions
		rules.NewMD054Rule, // link-image-style
		rules.NewMD055Rule, // table-pipe-style
		rules.NewMD056Rule, // table-column-count
		rules.NewMD058Rule, // blanks-around-tables
		rules.NewMD059Rule, // descriptive-link-text
	}

	// Register core rules (enabled by default)
	for _, constructor := range ruleConstructors {
		ruleResult := constructor()
		if ruleResult.IsErr() {
			return fmt.Errorf("failed to create rule: %w", ruleResult.Error())
		}

		rule := ruleResult.Unwrap()
		if err := re.RegisterRule(rule); err != nil {
			return fmt.Errorf("failed to register rule %s: %w", rule.PrimaryName(), err)
		}
	}

	// Register optional rules (disabled by default)
	for _, constructor := range optionalRuleConstructors {
		ruleResult := constructor()
		if ruleResult.IsErr() {
			return fmt.Errorf("failed to create rule: %w", ruleResult.Error())
		}

		rule := ruleResult.Unwrap()
		if err := re.RegisterRuleDisabled(rule); err != nil {
			return fmt.Errorf("failed to register optional rule %s: %w", rule.PrimaryName(), err)
		}
	}

	return nil
}

// RegisterRule registers a new rule with the engine.
func (re *RuleEngine) RegisterRule(rule *entity.Rule) error {
	re.mutex.Lock()
	defer re.mutex.Unlock()

	// Validate rule names don't conflict
	for _, name := range rule.Names() {
		if existingRule, exists := re.ruleIndex[strings.ToLower(name)]; exists {
			return fmt.Errorf("rule name %s conflicts with existing rule %s", name, existingRule.PrimaryName())
		}
	}

	// Add to rules list
	re.rules = append(re.rules, rule)

	// Index by all names/aliases
	for _, name := range rule.Names() {
		re.ruleIndex[strings.ToLower(name)] = rule
	}

	// Index by tags
	for _, tag := range rule.Tags() {
		tagLower := strings.ToLower(tag)
		re.tagIndex[tagLower] = append(re.tagIndex[tagLower], rule)
	}

	// Enable by default
	re.enabledRules[rule.PrimaryName()] = true
	re.ruleConfigs[rule.PrimaryName()] = rule.Config()

	return nil
}

// RegisterRuleDisabled registers a new rule with the engine but keeps it disabled by default.
func (re *RuleEngine) RegisterRuleDisabled(rule *entity.Rule) error {
	re.mutex.Lock()
	defer re.mutex.Unlock()

	// Validate rule names don't conflict
	for _, name := range rule.Names() {
		if existingRule, exists := re.ruleIndex[strings.ToLower(name)]; exists {
			return fmt.Errorf("rule name %s conflicts with existing rule %s", name, existingRule.PrimaryName())
		}
	}

	// Add to rules list
	re.rules = append(re.rules, rule)

	// Index by all names/aliases
	for _, name := range rule.Names() {
		re.ruleIndex[strings.ToLower(name)] = rule
	}

	// Index by tags
	for _, tag := range rule.Tags() {
		tagLower := strings.ToLower(tag)
		re.tagIndex[tagLower] = append(re.tagIndex[tagLower], rule)
	}

	// Disable by default for performance
	re.enabledRules[rule.PrimaryName()] = false
	re.ruleConfigs[rule.PrimaryName()] = rule.Config()

	return nil
}

// ConfigureRules configures rules based on the provided configuration map.
// This supports the markdownlint configuration format.
func (re *RuleEngine) ConfigureRules(config map[string]interface{}) error {
	re.mutex.Lock()
	defer re.mutex.Unlock()

	// Handle "default" rule
	defaultEnabled := true
	if defaultValue, exists := config["default"]; exists {
		if enabled, ok := defaultValue.(bool); ok {
			defaultEnabled = enabled
		}
	}

	// Set default state for all rules
	for _, rule := range re.rules {
		re.enabledRules[rule.PrimaryName()] = defaultEnabled
	}

	// Process individual rule configurations
	for key, value := range config {
		if key == "default" {
			continue
		}

		// Find rules by name or tag
		matchingRules := re.findRulesByNameOrTag(key)
		if len(matchingRules) == 0 {
			// Warn about unknown rule/tag, but don't error
			continue
		}

		// Configure each matching rule
		for _, rule := range matchingRules {
			ruleName := rule.PrimaryName()

			switch v := value.(type) {
			case bool:
				// Simple enable/disable
				re.enabledRules[ruleName] = v
			case map[string]interface{}:
				// Rule configuration
				re.enabledRules[ruleName] = true
				re.ruleConfigs[ruleName] = v
			default:
				return fmt.Errorf("invalid configuration value for rule %s: %v", key, value)
			}
		}
	}

	return nil
}

// findRulesByNameOrTag finds rules that match a name or tag (case-insensitive).
func (re *RuleEngine) findRulesByNameOrTag(nameOrTag string) []*entity.Rule {
	keyLower := strings.ToLower(nameOrTag)
	var rules []*entity.Rule

	// Check if it's a specific rule name
	if rule, exists := re.ruleIndex[keyLower]; exists {
		rules = append(rules, rule)
		return rules
	}

	// Check if it's a tag
	if tagRules, exists := re.tagIndex[keyLower]; exists {
		rules = append(rules, tagRules...)
	}

	return rules
}

// LintDocument runs all enabled rules against a parsed document.
func (re *RuleEngine) LintDocument(ctx context.Context, tokens []value.Token, lines []string, filename string) functional.Result[[]value.Violation] {
	re.mutex.RLock()
	defer re.mutex.RUnlock()

	allViolations := make([]value.Violation, 0)

	// Run each enabled rule
	for _, rule := range re.rules {
		ruleName := rule.PrimaryName()

		// Skip disabled rules
		if !re.enabledRules[ruleName] {
			continue
		}

		// Check for context cancellation
		select {
		case <-ctx.Done():
			return functional.Err[[]value.Violation](ctx.Err())
		default:
		}

		// Prepare rule parameters
		params := entity.RuleParams{
			Lines:       lines,
			Config:      re.ruleConfigs[ruleName],
			Filename:    filename,
			Tokens:      tokens,
			FrontMatter: functional.None[map[string]interface{}](), // TODO: Extract from tokens
		}

		// Execute the rule
		violationResult := rule.Execute(ctx, params)
		if violationResult.IsErr() {
			// Handle rule execution errors
			errorViolation := value.NewViolation(
				rule.Names(),
				"Rule execution error",
				rule.Information(),
				1,
			)
			errorViolation = errorViolation.WithErrorDetail(violationResult.Error().Error())
			errorViolation = errorViolation.WithSeverity(value.SeverityError)

			allViolations = append(allViolations, *errorViolation)
			continue
		}

		// Add rule information to violations
		violations := violationResult.Unwrap()
		for i := range violations {
			// Ensure rule information is set
			if violations[i].RuleInformation == nil {
				violations[i] = *violations[i].WithErrorContext("").WithFixInfo(value.FixInfo{})
				// Copy the struct and set the information
				updatedViolation := violations[i]
				updatedViolation.RuleInformation = rule.Information()
				violations[i] = updatedViolation
			}
		}

		allViolations = append(allViolations, violations...)
	}

	return functional.Ok(allViolations)
}

// GetRuleByName returns a rule by its name or alias (case-insensitive).
func (re *RuleEngine) GetRuleByName(name string) functional.Option[*entity.Rule] {
	re.mutex.RLock()
	defer re.mutex.RUnlock()

	if rule, exists := re.ruleIndex[strings.ToLower(name)]; exists {
		return functional.Some(rule)
	}

	return functional.None[*entity.Rule]()
}

// GetRulesByTag returns all rules with the specified tag (case-insensitive).
func (re *RuleEngine) GetRulesByTag(tag string) []*entity.Rule {
	re.mutex.RLock()
	defer re.mutex.RUnlock()

	if rules, exists := re.tagIndex[strings.ToLower(tag)]; exists {
		// Return a copy to maintain thread safety
		result := make([]*entity.Rule, len(rules))
		copy(result, rules)
		return result
	}

	return []*entity.Rule{}
}

// GetAllRules returns all registered rules.
func (re *RuleEngine) GetAllRules() []*entity.Rule {
	re.mutex.RLock()
	defer re.mutex.RUnlock()

	result := make([]*entity.Rule, len(re.rules))
	copy(result, re.rules)
	return result
}

// GetEnabledRules returns all currently enabled rules.
func (re *RuleEngine) GetEnabledRules() []*entity.Rule {
	re.mutex.RLock()
	defer re.mutex.RUnlock()

	var enabledRules []*entity.Rule
	for _, rule := range re.rules {
		if re.enabledRules[rule.PrimaryName()] {
			enabledRules = append(enabledRules, rule)
		}
	}

	return enabledRules
}

// IsRuleEnabled checks if a rule is enabled by name.
func (re *RuleEngine) IsRuleEnabled(name string) bool {
	re.mutex.RLock()
	defer re.mutex.RUnlock()

	// Find rule by name
	rule, exists := re.ruleIndex[strings.ToLower(name)]
	if !exists {
		return false
	}

	return re.enabledRules[rule.PrimaryName()]
}

// GetRuleConfig returns the configuration for a specific rule.
func (re *RuleEngine) GetRuleConfig(name string) map[string]interface{} {
	re.mutex.RLock()
	defer re.mutex.RUnlock()

	// Find rule by name
	rule, exists := re.ruleIndex[strings.ToLower(name)]
	if !exists {
		return nil
	}

	if config, exists := re.ruleConfigs[rule.PrimaryName()]; exists {
		// Return a copy to maintain immutability
		result := make(map[string]interface{})
		for k, v := range config {
			result[k] = v
		}
		return result
	}

	return nil
}

// Stats returns statistics about the rule engine.
func (re *RuleEngine) Stats() map[string]interface{} {
	re.mutex.RLock()
	defer re.mutex.RUnlock()

	enabledCount := 0
	for _, enabled := range re.enabledRules {
		if enabled {
			enabledCount++
		}
	}

	return map[string]interface{}{
		"total_rules":   len(re.rules),
		"enabled_rules": enabledCount,
		"tags":          len(re.tagIndex),
	}
}
