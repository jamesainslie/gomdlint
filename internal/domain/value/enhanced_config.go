package value

import (
	"fmt"
)

// Config represents the enhanced configuration with extension support
type Config struct {
	// Base configuration
	Default bool                             `json:"default" yaml:"default"`
	Rules   map[string]ExtendedRuleConfiguration     `json:",inline" yaml:",inline"`

	// Extension support
	Extends []string                         `json:"extends,omitempty" yaml:"extends,omitempty"`

	// Plugin configuration
	Plugins map[string]PluginConfiguration   `json:"plugins,omitempty" yaml:"plugins,omitempty"`

	// Parser configuration
	Parsers map[string]ParserConfiguration   `json:"parsers,omitempty" yaml:"parsers,omitempty"`

	// Advanced features
	Profiles map[string]ProfileConfiguration `json:"profiles,omitempty" yaml:"profiles,omitempty"`

	// Metadata
	Schema  string `json:"$schema,omitempty" yaml:"$schema,omitempty"`
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
}

// PluginConfiguration for plugin-specific settings
type PluginConfiguration struct {
	Enabled bool                   `json:"enabled"`
	Path    string                 `json:"path,omitempty"`
	Config  map[string]interface{} `json:"config,omitempty"`
}

// ParserConfiguration for parser-specific settings
type ParserConfiguration struct {
	Type    string                 `json:"type"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// ProfileConfiguration for configuration profiles
type ProfileConfiguration struct {
	Description string                        `json:"description,omitempty"`
	Extends     []string                      `json:"extends,omitempty"`
	Rules       map[string]ExtendedRuleConfiguration  `json:"rules,omitempty"`
	Plugins     map[string]PluginConfiguration `json:"plugins,omitempty"`
}

// ExtendedRuleConfiguration for enhanced rule configuration
type ExtendedRuleConfiguration struct {
	Enabled bool                   `json:"enabled"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// NewConfig creates a new configuration with default values
func NewConfig() *Config {
	return &Config{
		Default:  true,
		Rules:    make(map[string]ExtendedRuleConfiguration),
		Plugins:  make(map[string]PluginConfiguration),
		Parsers:  make(map[string]ParserConfiguration),
		Profiles: make(map[string]ProfileConfiguration),
		Extends:  make([]string, 0),
	}
}

// Merge merges another configuration into this one (other takes precedence)
func (c *Config) Merge(other *Config) *Config {
	merged := NewConfig()
	
	// Copy current config
	merged.Default = c.Default
	merged.Schema = c.Schema
	merged.Version = c.Version
	
	for name, rule := range c.Rules {
		merged.Rules[name] = rule
	}
	for name, plugin := range c.Plugins {
		merged.Plugins[name] = plugin
	}
	for name, parser := range c.Parsers {
		merged.Parsers[name] = parser
	}
	for name, profile := range c.Profiles {
		merged.Profiles[name] = profile
	}
	
	// Apply other config (takes precedence)
	if other.Default != c.Default {
		merged.Default = other.Default
	}
	if other.Schema != "" {
		merged.Schema = other.Schema
	}
	if other.Version != "" {
		merged.Version = other.Version
	}
	
	for name, rule := range other.Rules {
		merged.Rules[name] = rule
	}
	for name, plugin := range other.Plugins {
		merged.Plugins[name] = plugin
	}
	for name, parser := range other.Parsers {
		merged.Parsers[name] = parser
	}
	for name, profile := range other.Profiles {
		merged.Profiles[name] = profile
	}
	
	return merged
}

// GetRuleConfig returns the configuration for a specific rule
func (c *Config) GetRuleConfig(ruleName string) (ExtendedRuleConfiguration, bool) {
	config, exists := c.Rules[ruleName]
	return config, exists
}

// IsRuleEnabled checks if a rule is enabled
func (c *Config) IsRuleEnabled(ruleName string) bool {
	if config, exists := c.Rules[ruleName]; exists {
		return config.Enabled
	}
	return c.Default // Use default if rule not explicitly configured
}

// GetPluginConfig returns the configuration for a specific plugin
func (c *Config) GetPluginConfig(pluginName string) (PluginConfiguration, bool) {
	config, exists := c.Plugins[pluginName]
	return config, exists
}

// IsPluginEnabled checks if a plugin is enabled
func (c *Config) IsPluginEnabled(pluginName string) bool {
	if config, exists := c.Plugins[pluginName]; exists {
		return config.Enabled
	}
	return false // Plugins are disabled by default
}

// GetParserConfig returns the configuration for a specific parser
func (c *Config) GetParserConfig(parserName string) (ParserConfiguration, bool) {
	config, exists := c.Parsers[parserName]
	return config, exists
}

// GetProfile returns a configuration profile
func (c *Config) GetProfile(profileName string) (ProfileConfiguration, bool) {
	profile, exists := c.Profiles[profileName]
	return profile, exists
}

// Validate validates the configuration for correctness
func (c *Config) Validate() error {
	// Validate extends paths don't create circular references
	if err := c.validateExtendsChain(); err != nil {
		return err
	}

	// Validate rule configurations
	for ruleName, ruleConfig := range c.Rules {
		if err := c.validateRuleConfig(ruleName, ruleConfig); err != nil {
			return fmt.Errorf("invalid rule config for %s: %w", ruleName, err)
		}
	}

	// Validate plugin configurations
	for pluginName, pluginConfig := range c.Plugins {
		if err := c.validatePluginConfig(pluginName, pluginConfig); err != nil {
			return fmt.Errorf("invalid plugin config for %s: %w", pluginName, err)
		}
	}

	return nil
}

func (c *Config) validateExtendsChain() error {
	// This is a simplified validation - full implementation would check actual files
	return nil
}

func (c *Config) validateRuleConfig(ruleName string, config ExtendedRuleConfiguration) error {
	// Basic validation - could be enhanced with rule-specific validation
	if config.Options == nil {
		return fmt.Errorf("rule options cannot be nil")
	}
	return nil
}

func (c *Config) validatePluginConfig(pluginName string, config PluginConfiguration) error {
	if config.Enabled && config.Path == "" {
		return fmt.Errorf("enabled plugin must have a path")
	}
	return nil
}

// Clone creates a deep copy of the configuration
func (c *Config) Clone() *Config {
	clone := NewConfig()
	
	clone.Default = c.Default
	clone.Schema = c.Schema
	clone.Version = c.Version
	
	// Deep copy extends
	clone.Extends = make([]string, len(c.Extends))
	copy(clone.Extends, c.Extends)
	
	// Deep copy rules
	for name, rule := range c.Rules {
		clone.Rules[name] = ExtendedRuleConfiguration{
			Enabled: rule.Enabled,
			Options: cloneMap(rule.Options),
		}
	}
	
	// Deep copy plugins
	for name, plugin := range c.Plugins {
		clone.Plugins[name] = PluginConfiguration{
			Enabled: plugin.Enabled,
			Path:    plugin.Path,
			Config:  cloneMap(plugin.Config),
		}
	}
	
	// Deep copy parsers
	for name, parser := range c.Parsers {
		clone.Parsers[name] = ParserConfiguration{
			Type:    parser.Type,
			Options: cloneMap(parser.Options),
		}
	}
	
	// Deep copy profiles
	for name, profile := range c.Profiles {
		clonedProfile := ProfileConfiguration{
			Description: profile.Description,
			Extends:     make([]string, len(profile.Extends)),
			Rules:       make(map[string]ExtendedRuleConfiguration),
			Plugins:     make(map[string]PluginConfiguration),
		}
		
		copy(clonedProfile.Extends, profile.Extends)
		
		for ruleName, rule := range profile.Rules {
			clonedProfile.Rules[ruleName] = ExtendedRuleConfiguration{
				Enabled: rule.Enabled,
				Options: cloneMap(rule.Options),
			}
		}
		
		for pluginName, plugin := range profile.Plugins {
			clonedProfile.Plugins[pluginName] = PluginConfiguration{
				Enabled: plugin.Enabled,
				Path:    plugin.Path,
				Config:  cloneMap(plugin.Config),
			}
		}
		
		clone.Profiles[name] = clonedProfile
	}
	
	return clone
}

// cloneMap creates a deep copy of a map[string]interface{}
func cloneMap(original map[string]interface{}) map[string]interface{} {
	if original == nil {
		return make(map[string]interface{})
	}
	
	clone := make(map[string]interface{})
	for key, value := range original {
		// For simplicity, we'll do a shallow copy of the values
		// In a production system, you'd want deep cloning for nested maps/slices
		clone[key] = value
	}
	return clone
}