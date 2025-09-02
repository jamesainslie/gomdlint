package service

import (
	"testing"

	"github.com/gomdlint/gomdlint/internal/domain/value"
)

// Helper function to get int value from interface{}
func getIntValue(val interface{}) int {
	switch v := val.(type) {
	case int:
		return v
	case float64:
		return int(v)
	default:
		return -1 // Invalid value
	}
}

// Helper function to get bool value from interface{}
func getBoolValue(val interface{}) bool {
	if b, ok := val.(bool); ok {
		return b
	}
	return false
}

func TestStyleRegistry_NewStyleRegistry(t *testing.T) {
	registry := NewStyleRegistry()

	if registry == nil {
		t.Fatal("expected non-nil style registry")
	}

	styles := registry.ListStyles()
	if len(styles) == 0 {
		t.Error("expected built-in styles to be loaded")
	}

	expectedStyles := []string{"all", "relaxed", "strict", "prettier", "documentation", "blog", "academic"}

	for _, expected := range expectedStyles {
		found := false
		for _, style := range styles {
			if style == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected built-in style %s not found", expected)
		}
	}
}

func TestStyleRegistry_GetStyle(t *testing.T) {
	registry := NewStyleRegistry()

	// Test getting valid style
	style, err := registry.GetStyle("strict")
	if err != nil {
		t.Errorf("failed to get strict style: %v", err)
	}

	if style == nil {
		t.Error("expected non-nil style config")
	}

	if !style.Default {
		t.Error("expected strict style to have default=true")
	}

	// Verify strict style has expected rules
	if !style.IsRuleEnabled("MD013") {
		t.Error("expected MD013 to be enabled in strict style")
	}

	md013Config := style.GetRuleConfig("MD013")
	if lineLength, exists := md013Config["line_length"]; !exists {
		t.Error("expected line_length to be configured in strict style")
	} else if getIntValue(lineLength) != 80 {
		t.Errorf("expected line_length 80 in strict style, got %v", lineLength)
	}

	// Test getting non-existent style
	_, err = registry.GetStyle("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent style")
	}
}

func TestStyleRegistry_RegisterStyle(t *testing.T) {
	registry := NewStyleRegistry()

	// Create custom style
	customStyle := value.NewEnhancedConfig()
	customStyle.SetRule("MD001", true, map[string]interface{}{"increment": 1})
	customStyle.SetRule("MD013", false, nil)

	// Register custom style
	err := registry.RegisterStyle("custom", customStyle)
	if err != nil {
		t.Errorf("failed to register custom style: %v", err)
	}

	// Verify style was registered
	styles := registry.ListStyles()
	found := false
	for _, style := range styles {
		if style == "custom" {
			found = true
			break
		}
	}

	if !found {
		t.Error("custom style not found in registry")
	}

	// Verify we can retrieve it
	retrieved, err := registry.GetStyle("custom")
	if err != nil {
		t.Errorf("failed to retrieve custom style: %v", err)
	}

	if !retrieved.IsRuleEnabled("MD001") {
		t.Error("expected MD001 to be enabled in custom style")
	}

	if retrieved.IsRuleEnabled("MD013") {
		t.Error("expected MD013 to be disabled in custom style")
	}

	// Test registering invalid style
	err = registry.RegisterStyle("invalid", nil)
	if err == nil {
		t.Error("expected error for nil config")
	}
}

func TestStyleRegistry_UnregisterStyle(t *testing.T) {
	registry := NewStyleRegistry()

	// Create and register custom style
	customStyle := value.NewEnhancedConfig()
	customStyle.SetRule("MD001", true, nil)
	registry.RegisterStyle("temp", customStyle)

	// Verify it's registered
	_, err := registry.GetStyle("temp")
	if err != nil {
		t.Error("custom style should be registered")
	}

	// Unregister it
	err = registry.UnregisterStyle("temp")
	if err != nil {
		t.Errorf("failed to unregister custom style: %v", err)
	}

	// Verify it's gone
	_, err = registry.GetStyle("temp")
	if err == nil {
		t.Error("custom style should be unregistered")
	}

	// Test unregistering built-in style (should fail)
	err = registry.UnregisterStyle("strict")
	if err == nil {
		t.Error("expected error when trying to unregister built-in style")
	}

	// Test unregistering non-existent style
	err = registry.UnregisterStyle("nonexistent")
	if err == nil {
		t.Error("expected error when trying to unregister non-existent style")
	}
}

func TestStyleRegistry_GetStyleInfo(t *testing.T) {
	registry := NewStyleRegistry()

	info := registry.GetStyleInfo()
	if len(info) == 0 {
		t.Error("expected style info for built-in styles")
	}

	// Find strict style info
	var strictInfo *StyleInfo
	for i, si := range info {
		if si.Name == "strict" {
			strictInfo = &info[i]
			break
		}
	}

	if strictInfo == nil {
		t.Error("expected to find strict style info")
	}

	if !strictInfo.BuiltIn {
		t.Error("strict style should be marked as built-in")
	}

	if strictInfo.RuleCount == 0 {
		t.Error("strict style should have rules")
	}

	if strictInfo.Description == "" {
		t.Error("strict style should have description")
	}
}

func TestStyleRegistry_ApplyStyle(t *testing.T) {
	registry := NewStyleRegistry()

	// Create base config
	config := value.NewEnhancedConfig()
	config.SetRule("MD001", false, nil) // Will be overridden
	config.SetRule("MD002", true, nil)  // Will be preserved

	// Apply relaxed style
	err := registry.ApplyStyle(config, "relaxed")
	if err != nil {
		t.Errorf("failed to apply relaxed style: %v", err)
	}

	// Check that relaxed style rules were applied
	if !config.IsRuleEnabled("MD013") {
		t.Error("expected MD013 to be enabled after applying relaxed style")
	}

	md013Config := config.GetRuleConfig("MD013")
	if lineLength, exists := md013Config["line_length"]; !exists || getIntValue(lineLength) != 120 {
		t.Errorf("expected line_length 120 from relaxed style, got %v", lineLength)
	}

	// Check that MD033 (inline HTML) is disabled in relaxed style
	if config.IsRuleEnabled("MD033") {
		t.Error("expected MD033 to be disabled in relaxed style")
	}

	// Check that extends was updated
	found := false
	for _, extend := range config.Extends {
		if extend == "style:relaxed" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected extends to include applied style")
	}

	// Test applying non-existent style
	err = registry.ApplyStyle(config, "nonexistent")
	if err == nil {
		t.Error("expected error when applying non-existent style")
	}
}

func TestStyleRegistry_CreateStyleFromConfig(t *testing.T) {
	registry := NewStyleRegistry()

	// Create config with some rules
	sourceConfig := value.NewEnhancedConfig()
	sourceConfig.SetRule("MD001", true, map[string]interface{}{"increment": 1})
	sourceConfig.SetRule("MD013", true, map[string]interface{}{"line_length": 100})
	sourceConfig.Extends = []string{"some-extension"} // Should be removed

	// Create style from config
	err := registry.CreateStyleFromConfig("derived", sourceConfig)
	if err != nil {
		t.Errorf("failed to create style from config: %v", err)
	}

	// Verify style was created
	style, err := registry.GetStyle("derived")
	if err != nil {
		t.Errorf("failed to get derived style: %v", err)
	}

	if !style.IsRuleEnabled("MD001") {
		t.Error("derived style should have MD001 enabled")
	}

	md013Config := style.GetRuleConfig("MD013")
	if lineLength, exists := md013Config["line_length"]; !exists || getIntValue(lineLength) != 100 {
		t.Errorf("expected line_length 100 in derived style, got %v", lineLength)
	}

	// Verify extends was cleared
	if len(style.Extends) != 0 {
		t.Error("derived style should not have extends")
	}

	// Verify metadata was set
	if style.Metadata.Source != "custom:derived" {
		t.Errorf("expected source to be 'custom:derived', got %s", style.Metadata.Source)
	}
}

func TestBuiltInStyles_Validation(t *testing.T) {
	registry := NewStyleRegistry()

	builtInStyles := []string{"all", "relaxed", "strict", "prettier", "documentation", "blog", "academic"}

	for _, styleName := range builtInStyles {
		t.Run(styleName, func(t *testing.T) {
			style, err := registry.GetStyle(styleName)
			if err != nil {
				t.Errorf("failed to get style %s: %v", styleName, err)
				return
			}

			// Validate the style
			err = style.Validate()
			if err != nil {
				t.Errorf("style %s failed validation: %v", styleName, err)
			}

			// All built-in styles should have default=true
			if !style.Default {
				t.Errorf("built-in style %s should have default=true", styleName)
			}

			// Should have schema set
			if style.Schema == "" {
				t.Errorf("built-in style %s should have schema set", styleName)
			}

			// Should have metadata source
			if style.Metadata.Source == "" {
				t.Errorf("built-in style %s should have metadata source", styleName)
			}
		})
	}
}

func TestStyleSpecificConfigurations(t *testing.T) {
	registry := NewStyleRegistry()

	// Test strict style specifics
	strict, _ := registry.GetStyle("strict")
	strictMD013 := strict.GetRuleConfig("MD013")
	if int(strictMD013["line_length"].(float64)) != 80 {
		t.Error("strict style should have line_length 80")
	}
	if strictMD013["code_blocks"].(bool) != true {
		t.Error("strict style should have code_blocks enforcement")
	}

	// Test relaxed style specifics
	relaxed, _ := registry.GetStyle("relaxed")
	relaxedMD013 := relaxed.GetRuleConfig("MD013")
	if int(relaxedMD013["line_length"].(float64)) != 120 {
		t.Error("relaxed style should have line_length 120")
	}
	if relaxedMD013["code_blocks"].(bool) != false {
		t.Error("relaxed style should not enforce code_blocks")
	}

	// Verify MD033 is disabled in relaxed (allow inline HTML)
	if relaxed.IsRuleEnabled("MD033") {
		t.Error("relaxed style should allow inline HTML (MD033 disabled)")
	}

	// Test documentation style specifics
	documentation, _ := registry.GetStyle("documentation")
	docMD013 := documentation.GetRuleConfig("MD013")
	if int(docMD013["line_length"].(float64)) != 100 {
		t.Error("documentation style should have line_length 100")
	}

	// Should require first line H1
	if !documentation.IsRuleEnabled("MD041") {
		t.Error("documentation style should require first line H1")
	}
}
