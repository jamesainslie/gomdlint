package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// ConfigLoader defines the interface for loading configuration files
type ConfigLoader interface {
	LoadConfig(ctx context.Context, configPath string) functional.Result[*value.Config]
	SupportsPath(path string) bool
}

// ConfigResolver handles configuration extension and resolution
type ConfigResolver struct {
	loader       ConfigLoader
	cache        map[string]*value.Config
	resolveStack []string // For circular dependency detection
}

// NewConfigResolver creates a new configuration resolver
func NewConfigResolver(loader ConfigLoader) *ConfigResolver {
	return &ConfigResolver{
		loader: loader,
		cache:  make(map[string]*value.Config),
	}
}

// ResolveConfig resolves a configuration with all its extensions
func (cr *ConfigResolver) ResolveConfig(ctx context.Context, configPath string) functional.Result[*value.Config] {
	// Check for circular dependencies
	for _, path := range cr.resolveStack {
		if path == configPath {
			return functional.Err[*value.Config](
				fmt.Errorf("circular dependency detected: %s", strings.Join(cr.resolveStack, " -> ")),
			)
		}
	}

	// Check cache
	if cached, exists := cr.cache[configPath]; exists {
		return functional.Ok(cached)
	}

	// Add to resolve stack
	cr.resolveStack = append(cr.resolveStack, configPath)
	defer func() {
		if len(cr.resolveStack) > 0 {
			cr.resolveStack = cr.resolveStack[:len(cr.resolveStack)-1]
		}
	}()

	// Load base configuration
	baseResult := cr.loader.LoadConfig(ctx, configPath)
	if baseResult.IsErr() {
		return baseResult
	}

	baseConfig := baseResult.Unwrap()

	// If no extensions, return base config
	if len(baseConfig.Extends) == 0 {
		cr.cache[configPath] = baseConfig
		return functional.Ok(baseConfig)
	}

	// Resolve extensions
	resolvedConfig := &value.Config{
		Default:  baseConfig.Default,
		Rules:    make(map[string]value.ExtendedRuleConfiguration),
		Plugins:  make(map[string]value.PluginConfiguration),
		Parsers:  make(map[string]value.ParserConfiguration),
		Profiles: make(map[string]value.ProfileConfiguration),
		Schema:   baseConfig.Schema,
		Version:  baseConfig.Version,
	}

	// Process extensions in order (later ones override earlier ones)
	for _, extendPath := range baseConfig.Extends {
		// Resolve relative paths
		if !filepath.IsAbs(extendPath) {
			extendPath = filepath.Join(filepath.Dir(configPath), extendPath)
		}

		extendedResult := cr.ResolveConfig(ctx, extendPath)
		if extendedResult.IsErr() {
			return functional.Err[*value.Config](
				fmt.Errorf("failed to resolve extension %s: %w", extendPath, extendedResult.Error()),
			)
		}

		extended := extendedResult.Unwrap()

		// Merge configurations
		cr.mergeConfigs(resolvedConfig, extended)
	}

	// Apply base config on top
	cr.mergeConfigs(resolvedConfig, baseConfig)

	// Cache and return
	cr.cache[configPath] = resolvedConfig
	return functional.Ok(resolvedConfig)
}

// mergeConfigs merges source into target (source takes precedence)
func (cr *ConfigResolver) mergeConfigs(target, source *value.Config) {
	// Merge rules
	for name, rule := range source.Rules {
		target.Rules[name] = rule
	}

	// Merge plugins
	for name, plugin := range source.Plugins {
		target.Plugins[name] = plugin
	}

	// Merge parsers
	for name, parser := range source.Parsers {
		target.Parsers[name] = parser
	}

	// Merge profiles
	for name, profile := range source.Profiles {
		target.Profiles[name] = profile
	}

	// Override scalar values if they have meaningful values
	if source.Default != target.Default {
		target.Default = source.Default
	}

	if source.Schema != "" {
		target.Schema = source.Schema
	}

	if source.Version != "" {
		target.Version = source.Version
	}
}

// ClearCache clears the configuration cache
func (cr *ConfigResolver) ClearCache() {
	cr.cache = make(map[string]*value.Config)
}

// GetCacheStats returns cache statistics
func (cr *ConfigResolver) GetCacheStats() map[string]interface{} {
	return map[string]interface{}{
		"cached_configs": len(cr.cache),
		"resolve_depth":  len(cr.resolveStack),
	}
}

// JSONConfigLoader implements ConfigLoader for JSON files
type JSONConfigLoader struct{}

// NewJSONConfigLoader creates a new JSON config loader
func NewJSONConfigLoader() *JSONConfigLoader {
	return &JSONConfigLoader{}
}

// LoadConfig loads a JSON configuration file
func (jcl *JSONConfigLoader) LoadConfig(ctx context.Context, configPath string) functional.Result[*value.Config] {
	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return functional.Err[*value.Config](fmt.Errorf("config file not found: %s", configPath))
	}

	// Read file content
	data, err := os.ReadFile(configPath)
	if err != nil {
		return functional.Err[*value.Config](fmt.Errorf("failed to read config file %s: %w", configPath, err))
	}

	// Parse JSON
	var rawConfig map[string]interface{}
	if err := json.Unmarshal(data, &rawConfig); err != nil {
		return functional.Err[*value.Config](fmt.Errorf("failed to parse JSON config %s: %w", configPath, err))
	}

	// Convert to Config structure
	config := &value.Config{
		Default: true, // Default value
		Rules:   make(map[string]value.ExtendedRuleConfiguration),
		Plugins: make(map[string]value.PluginConfiguration),
		Parsers: make(map[string]value.ParserConfiguration),
		Profiles: make(map[string]value.ProfileConfiguration),
	}

	// Process extends field
	if extends, ok := rawConfig["extends"].([]interface{}); ok {
		config.Extends = make([]string, len(extends))
		for i, ext := range extends {
			if extStr, ok := ext.(string); ok {
				config.Extends[i] = extStr
			}
		}
		delete(rawConfig, "extends")
	}

	// Process schema and version
	if schema, ok := rawConfig["$schema"].(string); ok {
		config.Schema = schema
		delete(rawConfig, "$schema")
	}

	if version, ok := rawConfig["version"].(string); ok {
		config.Version = version
		delete(rawConfig, "version")
	}

	if defaultVal, ok := rawConfig["default"].(bool); ok {
		config.Default = defaultVal
		delete(rawConfig, "default")
	}

	// Process plugins section
	if plugins, ok := rawConfig["plugins"].(map[string]interface{}); ok {
		for name, pluginData := range plugins {
			if pluginConfig, ok := pluginData.(map[string]interface{}); ok {
				config.Plugins[name] = value.PluginConfiguration{
					Enabled: getBoolFromMap(pluginConfig, "enabled", true),
					Path:    getStringFromMap(pluginConfig, "path", ""),
					Config:  getMapFromMap(pluginConfig, "config"),
				}
			}
		}
		delete(rawConfig, "plugins")
	}

	// Process parsers section
	if parsers, ok := rawConfig["parsers"].(map[string]interface{}); ok {
		for name, parserData := range parsers {
			if parserConfig, ok := parserData.(map[string]interface{}); ok {
				config.Parsers[name] = value.ParserConfiguration{
					Type:    getStringFromMap(parserConfig, "type", name),
					Options: getMapFromMap(parserConfig, "options"),
				}
			}
		}
		delete(rawConfig, "parsers")
	}

	// All remaining fields are rule configurations
	for name, ruleData := range rawConfig {
		switch rd := ruleData.(type) {
		case bool:
			config.Rules[name] = value.ExtendedRuleConfiguration{
				Enabled: rd,
				Options: make(map[string]interface{}),
			}
		case map[string]interface{}:
			enabled := getBoolFromMap(rd, "enabled", true)
			options := make(map[string]interface{})
			for k, v := range rd {
				if k != "enabled" {
					options[k] = v
				}
			}
			config.Rules[name] = value.ExtendedRuleConfiguration{
				Enabled: enabled,
				Options: options,
			}
		default:
			// Treat as options for enabled rule
			config.Rules[name] = value.ExtendedRuleConfiguration{
				Enabled: true,
				Options: map[string]interface{}{"value": ruleData},
			}
		}
	}

	return functional.Ok(config)
}

// SupportsPath checks if the loader supports the given file path
func (jcl *JSONConfigLoader) SupportsPath(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".json" || ext == ".jsonc"
}

// Helper functions for map extraction
func getBoolFromMap(m map[string]interface{}, key string, defaultVal bool) bool {
	if val, ok := m[key].(bool); ok {
		return val
	}
	return defaultVal
}

func getStringFromMap(m map[string]interface{}, key string, defaultVal string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return defaultVal
}

func getMapFromMap(m map[string]interface{}, key string) map[string]interface{} {
	if val, ok := m[key].(map[string]interface{}); ok {
		return val
	}
	return make(map[string]interface{})
}