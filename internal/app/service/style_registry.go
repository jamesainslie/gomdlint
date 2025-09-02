package service

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/gomdlint/gomdlint/internal/domain/value"
)

// Built-in styles will be created programmatically for now
// In future versions, these could be loaded from embedded files

// StyleRegistry manages predefined configuration styles
type StyleRegistry struct {
	styles map[string]*value.Config
	mutex  sync.RWMutex
}

// NewStyleRegistry creates a new style registry with built-in styles
func NewStyleRegistry() *StyleRegistry {
	registry := &StyleRegistry{
		styles: make(map[string]*value.Config),
	}

	// Load built-in styles
	registry.loadBuiltInStyles()

	return registry
}

// loadBuiltInStyles loads all embedded style configurations
func (sr *StyleRegistry) loadBuiltInStyles() {
	styles := []string{
		"all",
		"relaxed",
		"strict",
		"prettier",
		"documentation",
		"blog",
		"academic",
		"minimal",
	}

	for _, style := range styles {
		// Create programmatic style for now
		// In future versions, this could load from embedded files
		sr.createProgrammaticStyle(style)
	}

	// Ensure we have at least basic styles programmatically
	sr.ensureBasicStyles()
}

// createProgrammaticStyle creates styles programmatically when files don't exist
func (sr *StyleRegistry) createProgrammaticStyle(styleName string) {
	var config *value.Config

	switch styleName {
	case "relaxed":
		config = sr.createRelaxedStyle()
	case "strict":
		config = sr.createStrictStyle()
	case "minimal":
		config = sr.createMinimalStyle()
	case "all":
		config = sr.createAllEnabledStyle()
	default:
		return // Unknown style
	}

	if config != nil {
		sr.styles[styleName] = config
	}
}

// createRelaxedStyle creates a relaxed configuration
func (sr *StyleRegistry) createRelaxedStyle() *value.Config {
	config := value.NewConfig()
	config.Schema = "https://raw.githubusercontent.com/gomdlint/gomdlint/main/schema/config.json"
	config.Version = "1.0.0"

	// Relaxed settings
	config.Rules = map[string]value.ExtendedRuleConfiguration{
		"MD013": {
			Enabled: true,
			Options: map[string]interface{}{
				"line_length": 120,
				"code_blocks": false,
				"tables":      false,
			},
		},
		"MD033": {Enabled: false, Options: make(map[string]interface{})}, // Allow inline HTML
		"MD041": {Enabled: false, Options: make(map[string]interface{})}, // Don't require first line heading
		"MD046": {
			Enabled: true,
			Options: map[string]interface{}{
				"style": "consistent",
			},
		},
	}

	return config
}

// createStrictStyle creates a strict configuration
func (sr *StyleRegistry) createStrictStyle() *value.Config {
	config := value.NewConfig()
	config.Schema = "https://raw.githubusercontent.com/gomdlint/gomdlint/main/schema/config.json"
	config.Version = "1.0.0"

	// Strict settings - most rules enabled with strict parameters
	config.Rules = map[string]value.ExtendedRuleConfiguration{
		"MD013": {
			Enabled: true,
			Options: map[string]interface{}{
				"line_length": 80,
				"code_blocks": true,
				"tables":      true,
			},
		},
		"MD022": {
			Enabled: true,
			Options: map[string]interface{}{
				"lines_above": 1,
				"lines_below": 1,
			},
		},
		"MD025": {
			Enabled: true,
			Options: map[string]interface{}{
				"level": 1,
			},
		},
		"MD041": {Enabled: true, Options: make(map[string]interface{})},  // Require first line heading
		"MD046": {
			Enabled: true,
			Options: map[string]interface{}{
				"style": "fenced",
			},
		},
	}

	return config
}

// createMinimalStyle creates a minimal configuration with only essential rules
func (sr *StyleRegistry) createMinimalStyle() *value.Config {
	config := value.NewConfig()
	config.Schema = "https://raw.githubusercontent.com/gomdlint/gomdlint/main/schema/config.json"
	config.Version = "1.0.0"
	config.Default = false // Disable all rules by default

	// Enable only essential rules
	config.Rules = map[string]value.ExtendedRuleConfiguration{
		"MD001": {Enabled: true, Options: make(map[string]interface{})}, // Heading increment
		"MD018": {Enabled: true, Options: make(map[string]interface{})}, // No space after ATX heading hash
		"MD019": {Enabled: true, Options: make(map[string]interface{})}, // Multiple spaces after ATX heading hash
	}

	return config
}

// createAllEnabledStyle creates a configuration with all rules enabled
func (sr *StyleRegistry) createAllEnabledStyle() *value.Config {
	config := value.NewConfig()
	config.Schema = "https://raw.githubusercontent.com/gomdlint/gomdlint/main/schema/config.json"
	config.Version = "1.0.0"
	config.Default = true // Enable all rules by default

	return config
}

// ensureBasicStyles ensures basic styles are available
func (sr *StyleRegistry) ensureBasicStyles() {
	requiredStyles := []string{"relaxed", "strict", "minimal", "all"}

	for _, styleName := range requiredStyles {
		if _, exists := sr.styles[styleName]; !exists {
			sr.createProgrammaticStyle(styleName)
		}
	}
}

// GetStyle retrieves a style by name
func (sr *StyleRegistry) GetStyle(name string) (*value.Config, error) {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	style, exists := sr.styles[name]
	if !exists {
		return nil, fmt.Errorf("style %s not found", name)
	}

	// Return a clone to prevent external modification
	return style.Clone(), nil
}

// ListStyles returns all available style names
func (sr *StyleRegistry) ListStyles() []string {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	names := make([]string, 0, len(sr.styles))
	for name := range sr.styles {
		names = append(names, name)
	}
	return names
}

// RegisterStyle adds a new custom style
func (sr *StyleRegistry) RegisterStyle(name string, config *value.Config) error {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	if name == "" {
		return fmt.Errorf("style name cannot be empty")
	}

	if config == nil {
		return fmt.Errorf("style config cannot be nil")
	}

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid style configuration: %w", err)
	}

	sr.styles[name] = config.Clone()
	return nil
}

// UnregisterStyle removes a style
func (sr *StyleRegistry) UnregisterStyle(name string) error {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	if _, exists := sr.styles[name]; !exists {
		return fmt.Errorf("style %s not found", name)
	}

	// Prevent unregistering built-in styles
	builtInStyles := []string{"all", "relaxed", "strict", "minimal", "prettier", "documentation", "blog", "academic"}
	for _, builtIn := range builtInStyles {
		if name == builtIn {
			return fmt.Errorf("cannot unregister built-in style: %s", name)
		}
	}

	delete(sr.styles, name)
	return nil
}

// GetStyleInfo returns detailed information about a style
func (sr *StyleRegistry) GetStyleInfo(name string) (*StyleInfo, error) {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	style, exists := sr.styles[name]
	if !exists {
		return nil, fmt.Errorf("style %s not found", name)
	}

	info := &StyleInfo{
		Name:        name,
		Description: getStyleDescription(style),
		RuleCount:   len(style.Rules),
		PluginCount: len(style.Plugins),
		ParserCount: len(style.Parsers),
		Extends:     style.Extends,
		Version:     style.Version,
		Schema:      style.Schema,
	}

	return info, nil
}

// StyleInfo contains metadata about a style
type StyleInfo struct {
	Name        string
	Description string
	RuleCount   int
	PluginCount int
	ParserCount int
	Extends     []string
	Version     string
	Schema      string
}

// getStyleDescription extracts description from style metadata
func getStyleDescription(config *value.Config) string {
	// Try to get description from various sources
	if config.Version != "" {
		return fmt.Sprintf("Configuration v%s", config.Version)
	}

	if config.Schema != "" {
		return "Structured configuration"
	}

	return "Custom configuration"
}

// ValidateStyle validates a style configuration
func (sr *StyleRegistry) ValidateStyle(name string) error {
	style, err := sr.GetStyle(name)
	if err != nil {
		return err
	}

	return style.Validate()
}

// ExportStyle exports a style to a file
func (sr *StyleRegistry) ExportStyle(name string, filename string) error {
	style, err := sr.GetStyle(name)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(style, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format style: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write style file: %w", err)
	}

	return nil
}

// globalStyleRegistry is the singleton instance
var (
	globalStyleRegistry *StyleRegistry
	styleRegistryOnce   sync.Once
)

// GetGlobalStyleRegistry returns the global style registry
func GetGlobalStyleRegistry() *StyleRegistry {
	styleRegistryOnce.Do(func() {
		globalStyleRegistry = NewStyleRegistry()
	})
	return globalStyleRegistry
}