package service

import (
	"context"
	"fmt"
	"net/url"
	"testing"

	"github.com/gomdlint/gomdlint/internal/domain/entity"
	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/functional"
	pluginpkg "github.com/gomdlint/gomdlint/pkg/gomdlint/plugin"
)

// Mock plugin implementation for testing
type mockPlugin struct {
	name        string
	version     string
	description string
	author      string
	rules       []pluginpkg.CustomRule
	initialized bool
	healthOK    bool
}

func (m *mockPlugin) Name() string        { return m.name }
func (m *mockPlugin) Version() string     { return m.version }
func (m *mockPlugin) Description() string { return m.description }
func (m *mockPlugin) Author() string      { return m.author }
func (m *mockPlugin) Rules() []pluginpkg.CustomRule { return m.rules }

func (m *mockPlugin) Initialize(ctx context.Context, config pluginpkg.PluginConfig) error {
	m.initialized = true
	return nil
}

func (m *mockPlugin) Shutdown(ctx context.Context) error {
	m.initialized = false
	return nil
}

func (m *mockPlugin) HealthCheck(ctx context.Context) error {
	if !m.healthOK {
		return fmt.Errorf("health check failed")
	}
	return nil
}

// Mock custom rule implementation
type mockCustomRule struct {
	names       []string
	description string
	tags        []string
	info        *url.URL
	parser      string
	config      map[string]interface{}
}

func (m *mockCustomRule) Names() []string                            { return m.names }
func (m *mockCustomRule) Description() string                       { return m.description }
func (m *mockCustomRule) Tags() []string                            { return m.tags }
func (m *mockCustomRule) Information() *url.URL                     { return m.info }
func (m *mockCustomRule) Parser() string                            { return m.parser }
func (m *mockCustomRule) DefaultConfig() map[string]interface{}     { return m.config }
func (m *mockCustomRule) ValidateConfig(config map[string]interface{}) error { return nil }
func (m *mockCustomRule) IsAsync() bool                             { return false }
func (m *mockCustomRule) Metadata() map[string]interface{}          { return map[string]interface{}{} }

func (m *mockCustomRule) Execute(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation] {
	// Mock rule that always passes
	return functional.Ok([]value.Violation{})
}

func TestPluginManager_NewPluginManager(t *testing.T) {
	config := pluginpkg.PluginConfig{
		DataDir:   "/tmp/test",
		ConfigDir: "/tmp/config",
		CacheDir:  "/tmp/cache",
		LogLevel:  "info",
	}

	pm := NewPluginManager(config)

	if pm == nil {
		t.Fatal("expected non-nil plugin manager")
	}

	if len(pm.plugins) != 0 {
		t.Errorf("expected empty plugins map, got %d plugins", len(pm.plugins))
	}
}

func TestPluginManager_ManualPluginRegistration(t *testing.T) {
	config := pluginpkg.PluginConfig{
		DataDir: "/tmp/test",
	}

	pm := NewPluginManager(config)
	ctx := context.Background()

	// Create mock plugin with a rule
	infoURL, _ := url.Parse("https://example.com/rule")
	mockRule := &mockCustomRule{
		names:       []string{"TEST001", "test-rule"},
		description: "Test rule",
		tags:        []string{"test"},
		info:        infoURL,
		parser:      "commonmark",
		config:      map[string]interface{}{"enabled": true},
	}

	plugin := &mockPlugin{
		name:        "test-plugin",
		version:     "1.0.0",
		description: "Test plugin",
		author:      "Test Author",
		rules:       []pluginpkg.CustomRule{mockRule},
		healthOK:    true,
	}

	// Manually register plugin (simulating what would happen after loading)
	err := pm.registerPlugin(ctx, plugin)
	if err != nil {
		t.Fatalf("failed to register plugin: %v", err)
	}

	// Verify plugin is registered
	if len(pm.plugins) != 1 {
		t.Errorf("expected 1 plugin, got %d", len(pm.plugins))
	}

	retrievedPlugin, exists := pm.GetPlugin("test-plugin")
	if !exists {
		t.Error("expected plugin to be found")
	}

	if retrievedPlugin.Name() != "test-plugin" {
		t.Errorf("expected plugin name 'test-plugin', got %q", retrievedPlugin.Name())
	}

	// Verify rules are available
	rules := pm.GetAllCustomRules()
	if len(rules) != 1 {
		t.Errorf("expected 1 rule, got %d", len(rules))
	}

	if rules[0].PrimaryName() != "TEST001" {
		t.Errorf("expected rule name 'TEST001', got %q", rules[0].PrimaryName())
	}
}

func TestPluginManager_PluginLifecycle(t *testing.T) {
	config := pluginpkg.PluginConfig{
		DataDir: "/tmp/test",
	}

	pm := NewPluginManager(config)
	ctx := context.Background()

	plugin := &mockPlugin{
		name:        "lifecycle-test",
		version:     "1.0.0",
		description: "Lifecycle test plugin",
		author:      "Test Author",
		rules:       []pluginpkg.CustomRule{},
		healthOK:    true,
	}

	// Register plugin
	err := pm.registerPlugin(ctx, plugin)
	if err != nil {
		t.Fatalf("failed to register plugin: %v", err)
	}

	if !plugin.initialized {
		t.Error("expected plugin to be initialized")
	}

	// Unload plugin
	err = pm.UnloadPlugin(ctx, "lifecycle-test")
	if err != nil {
		t.Fatalf("failed to unload plugin: %v", err)
	}

	if plugin.initialized {
		t.Error("expected plugin to be shutdown")
	}

	// Verify plugin is removed
	_, exists := pm.GetPlugin("lifecycle-test")
	if exists {
		t.Error("expected plugin to be removed")
	}
}

func TestPluginManager_GetPluginInfo(t *testing.T) {
	config := pluginpkg.PluginConfig{
		DataDir: "/tmp/test",
	}

	pm := NewPluginManager(config)
	ctx := context.Background()

	plugin := &mockPlugin{
		name:        "info-test",
		version:     "2.1.0",
		description: "Info test plugin",
		author:      "Info Author",
		rules:       []pluginpkg.CustomRule{},
		healthOK:    true,
	}

	err := pm.registerPlugin(ctx, plugin)
	if err != nil {
		t.Fatalf("failed to register plugin: %v", err)
	}

	info := pm.GetPluginInfo()
	if len(info) != 1 {
		t.Errorf("expected 1 plugin info, got %d", len(info))
	}

	pluginInfo := info[0]
	if pluginInfo.Name != "info-test" {
		t.Errorf("expected name 'info-test', got %q", pluginInfo.Name)
	}

	if pluginInfo.Version != "2.1.0" {
		t.Errorf("expected version '2.1.0', got %q", pluginInfo.Version)
	}

	if pluginInfo.Author != "Info Author" {
		t.Errorf("expected author 'Info Author', got %q", pluginInfo.Author)
	}

	if !pluginInfo.Loaded {
		t.Error("expected plugin to be loaded")
	}
}

func TestPluginManager_Shutdown(t *testing.T) {
	config := pluginpkg.PluginConfig{
		DataDir: "/tmp/test",
	}

	pm := NewPluginManager(config)
	ctx := context.Background()

	// Register multiple plugins
	for i := 0; i < 3; i++ {
		plugin := &mockPlugin{
			name:        fmt.Sprintf("shutdown-test-%d", i),
			version:     "1.0.0",
			description: "Shutdown test plugin",
			author:      "Test Author",
			rules:       []pluginpkg.CustomRule{},
			healthOK:    true,
		}

		err := pm.registerPlugin(ctx, plugin)
		if err != nil {
			t.Fatalf("failed to register plugin %d: %v", i, err)
		}
	}

	// Verify plugins are loaded
	if len(pm.plugins) != 3 {
		t.Errorf("expected 3 plugins, got %d", len(pm.plugins))
	}

	// Shutdown all plugins
	err := pm.Shutdown(ctx)
	if err != nil {
		t.Fatalf("failed to shutdown plugins: %v", err)
	}

	// Verify all plugins are unloaded
	if len(pm.plugins) != 0 {
		t.Errorf("expected 0 plugins after shutdown, got %d", len(pm.plugins))
	}

	if len(pm.adapters) != 0 {
		t.Errorf("expected 0 adapters after shutdown, got %d", len(pm.adapters))
	}
}

// Helper method for manual plugin registration (for testing)
func (pm *PluginManager) registerPlugin(ctx context.Context, plugin pluginpkg.Plugin) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// Initialize plugin
	if err := plugin.Initialize(ctx, pm.config); err != nil {
		return fmt.Errorf("failed to initialize plugin: %w", err)
	}

	// Health check
	if err := plugin.HealthCheck(ctx); err != nil {
		return fmt.Errorf("plugin failed health check: %w", err)
	}

	pluginName := plugin.Name()

	// Check for name conflicts
	if _, exists := pm.plugins[pluginName]; exists {
		return fmt.Errorf("plugin %s already loaded", pluginName)
	}

	// Store plugin
	pm.plugins[pluginName] = plugin

	// Create rule adapters
	for _, customRule := range plugin.Rules() {
		adapterResult := pluginpkg.NewRuleAdapter(customRule)
		if adapterResult.IsErr() {
			continue
		}
		adapter := adapterResult.Unwrap()

		ruleKey := fmt.Sprintf("%s:%s", pluginName, customRule.Names()[0])
		pm.adapters[ruleKey] = adapter
	}

	return nil
}

func TestRuleAdapter_Integration(t *testing.T) {
	// Create a mock custom rule
	infoURL, _ := url.Parse("https://example.com/test-rule")
	mockRule := &mockCustomRule{
		names:       []string{"CUSTOM001", "custom-test"},
		description: "Custom test rule",
		tags:        []string{"custom", "test"},
		info:        infoURL,
		parser:      "commonmark",
		config:      map[string]interface{}{"threshold": 5},
	}

	// Create adapter
	adapterResult := pluginpkg.NewRuleAdapter(mockRule)
	if adapterResult.IsErr() {
		t.Fatalf("failed to create adapter: %v", adapterResult.Error())
	}

	adapter := adapterResult.Unwrap()

	// Test that adapter creates valid entity.Rule
	rule := adapter.Rule()
	if rule == nil {
		t.Fatal("expected non-nil rule")
	}

	if rule.PrimaryName() != "CUSTOM001" {
		t.Errorf("expected primary name 'CUSTOM001', got %q", rule.PrimaryName())
	}

	if rule.Description() != "Custom test rule" {
		t.Errorf("expected description 'Custom test rule', got %q", rule.Description())
	}

	// Test rule execution
	ctx := context.Background()
	params := entity.RuleParams{
		Lines:    []string{"# Test heading", "Some content"},
		Config:   map[string]interface{}{"threshold": 5},
		Filename: "test.md",
		Tokens:   []value.Token{},
	}

	result := rule.Execute(ctx, params)
	if result.IsErr() {
		t.Errorf("rule execution failed: %v", result.Error())
	}

	violations := result.Unwrap()
	if len(violations) != 0 {
		t.Errorf("expected 0 violations, got %d", len(violations))
	}
}
