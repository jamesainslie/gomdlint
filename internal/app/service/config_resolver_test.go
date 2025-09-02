package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// MockConfigLoader for testing
type MockConfigLoader struct {
	configs map[string]*value.EnhancedConfig
	errors  map[string]error
}

func NewMockConfigLoader() *MockConfigLoader {
	return &MockConfigLoader{
		configs: make(map[string]*value.EnhancedConfig),
		errors:  make(map[string]error),
	}
}

func (mcl *MockConfigLoader) LoadConfig(ctx context.Context, path string) functional.Result[*value.EnhancedConfig] {
	if err, exists := mcl.errors[path]; exists {
		return functional.Err[*value.EnhancedConfig](err)
	}

	if config, exists := mcl.configs[path]; exists {
		cloned := config.Clone()
		cloned.Metadata.Source = path
		cloned.Metadata.LoadedFrom = []string{path}
		cloned.Metadata.ResolvedAt = time.Now().Format(time.RFC3339)
		return functional.Ok(cloned)
	}

	return functional.Err[*value.EnhancedConfig](fmt.Errorf("config not found: %s", path))
}

func (mcl *MockConfigLoader) ExistsConfig(path string) bool {
	_, exists := mcl.configs[path]
	return exists
}

func (mcl *MockConfigLoader) SetConfig(path string, config *value.EnhancedConfig) {
	mcl.configs[path] = config
}

func (mcl *MockConfigLoader) SetError(path string, err error) {
	mcl.errors[path] = err
}

func TestEnhancedConfig_NewEnhancedConfig(t *testing.T) {
	config := value.NewEnhancedConfig()

	if config == nil {
		t.Fatal("expected non-nil config")
	}

	if !config.Default {
		t.Error("expected default to be true")
	}

	if config.GlobalOptions.MaxConcurrency != 4 {
		t.Errorf("expected default max concurrency 4, got %d", config.GlobalOptions.MaxConcurrency)
	}

	if !config.GlobalOptions.EnableCaching {
		t.Error("expected caching to be enabled by default")
	}

	if config.Schema == "" {
		t.Error("expected schema to be set")
	}
}

func TestEnhancedConfig_SetRule(t *testing.T) {
	config := value.NewEnhancedConfig()

	// Set rule enabled with options
	options := map[string]interface{}{
		"line_length": 120,
		"code_blocks": false,
	}
	config.SetRule("MD013", true, options)

	if !config.IsRuleEnabled("MD013") {
		t.Error("expected MD013 to be enabled")
	}

	ruleConfig := config.GetRuleConfig("MD013")
	if ruleConfig["line_length"] != 120 {
		t.Errorf("expected line_length 120, got %v", ruleConfig["line_length"])
	}

	// Set rule disabled
	config.SetRule("MD001", false, nil)
	if config.IsRuleEnabled("MD001") {
		t.Error("expected MD001 to be disabled")
	}
}

func TestEnhancedConfig_Plugins(t *testing.T) {
	config := value.NewEnhancedConfig()

	// Add plugin
	pluginConfig := value.PluginConfiguration{
		Enabled: true,
		Path:    "/path/to/plugin.so",
		Version: "1.0.0",
		Config: map[string]interface{}{
			"option1": "value1",
		},
	}

	config.AddPlugin("test-plugin", pluginConfig)

	if _, exists := config.Plugins["test-plugin"]; !exists {
		t.Error("expected plugin to be added")
	}

	if !config.Plugins["test-plugin"].Enabled {
		t.Error("expected plugin to be enabled")
	}

	// Remove plugin
	config.RemovePlugin("test-plugin")
	if _, exists := config.Plugins["test-plugin"]; exists {
		t.Error("expected plugin to be removed")
	}
}

func TestEnhancedConfig_Validation(t *testing.T) {
	config := value.NewEnhancedConfig()

	// Valid config should pass
	err := config.Validate()
	if err != nil {
		t.Errorf("valid config should pass validation: %v", err)
	}

	// Invalid severity should fail
	severity := "invalid"
	config.Rules["TEST001"] = value.RuleConfiguration{
		Severity: &severity,
	}

	err = config.Validate()
	if err == nil {
		t.Error("invalid severity should fail validation")
	}

	// Reset for next test
	config = value.NewEnhancedConfig()

	// Plugin enabled without path should fail
	config.Plugins["broken-plugin"] = value.PluginConfiguration{
		Enabled: true,
		Path:    "",
	}

	err = config.Validate()
	if err == nil {
		t.Error("plugin without path should fail validation")
	}
}

func TestEnhancedConfig_Clone(t *testing.T) {
	config := value.NewEnhancedConfig()
	config.SetRule("MD001", true, map[string]interface{}{"option": "value"})
	config.AddPlugin("test", value.PluginConfiguration{Enabled: true, Path: "/test"})

	cloned := config.Clone()

	// Verify they're separate instances
	if config == cloned {
		t.Error("clone should return different instance")
	}

	// Verify content is the same
	if config.Default != cloned.Default {
		t.Error("cloned config should have same default value")
	}

	if len(config.Rules) != len(cloned.Rules) {
		t.Error("cloned config should have same number of rules")
	}

	// Verify deep copy (modifying original shouldn't affect clone)
	config.SetRule("MD002", false, nil)
	if len(config.Rules) == len(cloned.Rules) {
		t.Error("modifying original should not affect clone")
	}
}

func TestEnhancedConfig_JSONSerialization(t *testing.T) {
	config := value.NewEnhancedConfig()
	config.SetRule("MD001", true, map[string]interface{}{"level": 1})
	config.Default = false

	// To JSON
	jsonResult := config.ToJSON()
	if jsonResult.IsErr() {
		t.Fatalf("failed to serialize to JSON: %v", jsonResult.Error())
	}

	jsonData := jsonResult.Unwrap()

	// From JSON
	configResult := value.FromJSON(jsonData)
	if configResult.IsErr() {
		t.Fatalf("failed to deserialize from JSON: %v", configResult.Error())
	}

	newConfig := configResult.Unwrap()

	// Verify deserialized config
	if newConfig.Default != false {
		t.Error("deserialized config should have correct default value")
	}

	if !newConfig.IsRuleEnabled("MD001") {
		t.Error("deserialized config should have MD001 enabled")
	}

	ruleConfig := newConfig.GetRuleConfig("MD001")
	if ruleConfig["level"] != 1.0 { // JSON numbers become float64
		t.Errorf("expected level 1.0, got %v", ruleConfig["level"])
	}
}

func TestConfigResolver_SimpleResolution(t *testing.T) {
	loader := NewMockConfigLoader()
	resolver := NewConfigResolver(loader)

	// Create base config
	baseConfig := value.NewEnhancedConfig()
	baseConfig.SetRule("MD001", true, nil)
	baseConfig.SetRule("MD013", false, nil)

	loader.SetConfig("/base.json", baseConfig)

	ctx := context.Background()
	result := resolver.ResolveConfig(ctx, "/base.json")

	if result.IsErr() {
		t.Fatalf("failed to resolve config: %v", result.Error())
	}

	resolved := result.Unwrap()

	if !resolved.IsRuleEnabled("MD001") {
		t.Error("expected MD001 to be enabled")
	}

	if resolved.IsRuleEnabled("MD013") {
		t.Error("expected MD013 to be disabled")
	}
}

func TestConfigResolver_ExtensionResolution(t *testing.T) {
	loader := NewMockConfigLoader()
	resolver := NewConfigResolver(loader)

	// Create parent config
	parentConfig := value.NewEnhancedConfig()
	parentConfig.SetRule("MD001", true, map[string]interface{}{"increment": 1})
	parentConfig.SetRule("MD013", true, map[string]interface{}{"line_length": 80})
	parentConfig.Default = false

	loader.SetConfig("/parent.json", parentConfig)

	// Create child config that extends parent
	childConfig := value.NewEnhancedConfig()
	childConfig.Extends = []string{"/parent.json"}
	childConfig.Default = true               // This will be the merged result since child comes after parent
	childConfig.SetRule("MD013", false, nil) // Override parent
	childConfig.SetRule("MD025", true, map[string]interface{}{"level": 1})

	loader.SetConfig("/child.json", childConfig)

	ctx := context.Background()
	result := resolver.ResolveConfig(ctx, "/child.json")

	if result.IsErr() {
		t.Fatalf("failed to resolve config: %v", result.Error())
	}

	resolved := result.Unwrap()

	// Should inherit from parent
	if !resolved.IsRuleEnabled("MD001") {
		t.Error("expected MD001 to be enabled from parent")
	}

	// Should override parent
	if resolved.IsRuleEnabled("MD013") {
		t.Error("expected MD013 to be disabled (overridden)")
	}

	// Should have child's rule
	if !resolved.IsRuleEnabled("MD025") {
		t.Error("expected MD025 to be enabled from child")
	}

	// Should use child's default (child overrides parent)
	if !resolved.Default {
		t.Errorf("expected default to be true (child overrides parent), got %v", resolved.Default)
	}
}

func TestConfigResolver_CircularDependency(t *testing.T) {
	loader := NewMockConfigLoader()
	resolver := NewConfigResolver(loader)

	// Create config A that extends B
	configA := value.NewEnhancedConfig()
	configA.Extends = []string{"/b.json"}
	loader.SetConfig("/a.json", configA)

	// Create config B that extends A (circular)
	configB := value.NewEnhancedConfig()
	configB.Extends = []string{"/a.json"}
	loader.SetConfig("/b.json", configB)

	ctx := context.Background()
	result := resolver.ResolveConfig(ctx, "/a.json")

	if result.IsOk() {
		t.Error("expected circular dependency error")
	}

	if !strings.Contains(result.Error().Error(), "circular dependency") {
		t.Errorf("expected circular dependency error, got: %v", result.Error())
	}
}

func TestConfigResolver_Caching(t *testing.T) {
	loader := NewMockConfigLoader()
	resolver := NewConfigResolver(loader)

	config := value.NewEnhancedConfig()
	config.SetRule("MD001", true, nil)

	loader.SetConfig("/test.json", config)

	ctx := context.Background()

	// First resolution
	result1 := resolver.ResolveConfig(ctx, "/test.json")
	if result1.IsErr() {
		t.Fatalf("first resolution failed: %v", result1.Error())
	}

	// Second resolution (should be cached)
	result2 := resolver.ResolveConfig(ctx, "/test.json")
	if result2.IsErr() {
		t.Fatalf("second resolution failed: %v", result2.Error())
	}

	// Check cache stats
	stats := resolver.GetCacheStats()
	if stats["size"].(int) != 1 {
		t.Errorf("expected cache size 1, got %d", stats["size"].(int))
	}

	cachedFiles := stats["cached_files"].([]string)
	if len(cachedFiles) != 1 || cachedFiles[0] != "/test.json" {
		t.Errorf("expected cached file /test.json, got %v", cachedFiles)
	}
}

func TestConfigResolver_ValidationErrors(t *testing.T) {
	loader := NewMockConfigLoader()
	resolver := NewConfigResolver(loader)

	// Create config with issues
	config := value.NewEnhancedConfig()

	// Add plugin with missing path
	config.Plugins["broken-plugin"] = value.PluginConfiguration{
		Enabled: true,
		Path:    "",
	}

	// Add invalid severity
	severity := "invalid"
	config.Rules["TEST001"] = value.RuleConfiguration{
		Severity: &severity,
	}

	loader.SetConfig("/broken.json", config)

	ctx := context.Background()
	result := resolver.ValidateConfig(ctx, "/broken.json")

	if result.IsErr() {
		t.Fatalf("validation should return errors, not fail: %v", result.Error())
	}

	errors := result.Unwrap()
	if len(errors) == 0 {
		t.Error("expected validation errors")
	}

	// Should have validation errors
	hasValidationError := false
	for _, err := range errors {
		if err.Type == "validation" {
			hasValidationError = true
			break
		}
	}

	if !hasValidationError {
		t.Error("expected validation errors for invalid configuration")
	}
}

func TestFileConfigLoader_LoadConfig(t *testing.T) {
	// Create temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test.json")

	config := value.NewEnhancedConfig()
	config.SetRule("MD001", true, map[string]interface{}{"increment": 1})

	jsonData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}

	err = os.WriteFile(configPath, jsonData, 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Test loading
	loader := NewFileConfigLoader()
	ctx := context.Background()

	result := loader.LoadConfig(ctx, configPath)
	if result.IsErr() {
		t.Fatalf("failed to load config: %v", result.Error())
	}

	loadedConfig := result.Unwrap()

	if !loadedConfig.IsRuleEnabled("MD001") {
		t.Error("expected MD001 to be enabled in loaded config")
	}

	if loadedConfig.Metadata.Source != configPath {
		t.Errorf("expected source to be %s, got %s", configPath, loadedConfig.Metadata.Source)
	}

	// Test existence check
	if !loader.ExistsConfig(configPath) {
		t.Error("config file should exist")
	}

	if loader.ExistsConfig(filepath.Join(tempDir, "nonexistent.json")) {
		t.Error("nonexistent file should not exist")
	}
}

func TestEnhancedConfig_LegacyCompatibility(t *testing.T) {
	// Test conversion to LintOptions
	config := value.NewEnhancedConfig()
	config.SetRule("MD001", true, map[string]interface{}{"increment": 1})
	config.SetRule("MD013", false, nil)
	config.Default = false

	lintOptions := config.ToLintOptions()

	// Check default
	if defaultVal, exists := lintOptions.Config["default"]; !exists || defaultVal != false {
		t.Error("expected default to be false in legacy format")
	}

	// Check enabled rule with options
	if md001, exists := lintOptions.Config["MD001"]; !exists {
		t.Error("expected MD001 in legacy config")
	} else if opts, ok := md001.(map[string]interface{}); !ok {
		t.Error("expected MD001 to have options")
	} else if opts["increment"] != 1 {
		t.Error("expected increment option to be preserved")
	}

	// Check disabled rule
	if md013, exists := lintOptions.Config["MD013"]; !exists || md013 != false {
		t.Error("expected MD013 to be false in legacy config")
	}

	// Test conversion from LintOptions
	legacyConfig := map[string]interface{}{
		"default": true,
		"MD001":   map[string]interface{}{"increment": 2},
		"MD013":   false,
	}

	lintOpts := value.NewLintOptions().WithConfig(legacyConfig)
	enhancedConfig := value.FromLintOptions(lintOpts)

	if !enhancedConfig.Default {
		t.Error("expected default to be true from legacy config")
	}

	if !enhancedConfig.IsRuleEnabled("MD001") {
		t.Error("expected MD001 to be enabled from legacy config")
	}

	if enhancedConfig.IsRuleEnabled("MD013") {
		t.Error("expected MD013 to be disabled from legacy config")
	}

	md001Config := enhancedConfig.GetRuleConfig("MD001")
	if md001Config["increment"] != 2 {
		t.Errorf("expected increment 2, got %v", md001Config["increment"])
	}
}
