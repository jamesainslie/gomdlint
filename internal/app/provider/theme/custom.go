package theme

import (
	"context"
	"fmt"

	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// CustomProvider provides custom themes from user configuration.
type CustomProvider struct{}

// NewCustomProvider creates a new custom theme provider.
func NewCustomProvider() *CustomProvider {
	return &CustomProvider{}
}

// Name returns the provider name.
func (cp *CustomProvider) Name() string {
	return "custom"
}

// CanHandle returns true if this provider can handle the theme.
// Custom provider handles any theme that's not a builtin theme.
func (cp *CustomProvider) CanHandle(config value.ThemeConfig) bool {
	builtinThemes := map[string]bool{
		"default": true,
		"minimal": true,
		"ascii":   true,
	}

	return !builtinThemes[config.ThemeName]
}

// CreateTheme creates a custom theme.
func (cp *CustomProvider) CreateTheme(ctx context.Context, config value.ThemeConfig) functional.Result[value.Theme] {
	// Validate the configuration first
	if err := cp.ValidateConfig(config); err != nil {
		return functional.Err[value.Theme](err)
	}

	// For custom themes, we fall back to a base theme and apply customizations
	baseConfig := cp.createBaseConfig(config)

	// Create the theme using the domain logic
	return value.NewTheme(baseConfig)
}

// ValidateConfig validates the configuration for custom themes.
func (cp *CustomProvider) ValidateConfig(config value.ThemeConfig) error {
	// For custom themes, we mainly validate the custom symbols
	if err := cp.validateCustomSymbols(config.CustomSymbols); err != nil {
		return fmt.Errorf("invalid custom symbols: %w", err)
	}

	// Ensure theme name is not empty
	if config.ThemeName == "" {
		return fmt.Errorf("theme name cannot be empty")
	}

	return nil
}

// createBaseConfig creates a base configuration for custom themes.
func (cp *CustomProvider) createBaseConfig(config value.ThemeConfig) value.ThemeConfig {
	// For unknown custom themes, we use minimal as the base
	baseConfig := value.ThemeConfig{
		ThemeName:      "minimal", // Use minimal as base for custom themes
		SuppressEmojis: config.SuppressEmojis,
		CustomSymbols:  make(map[string]string),
	}

	// Copy all custom symbols
	for k, v := range config.CustomSymbols {
		baseConfig.CustomSymbols[k] = v
	}

	return baseConfig
}

// validateCustomSymbols validates custom symbol overrides.
func (cp *CustomProvider) validateCustomSymbols(symbols map[string]string) error {
	validSymbolNames := map[string]bool{
		"success":     true,
		"error":       true,
		"warning":     true,
		"info":        true,
		"processing":  true,
		"file_found":  true,
		"file_saved":  true,
		"benchmark":   true,
		"performance": true,
		"winner":      true,
		"results":     true,
		"search":      true,
		"launch":      true,
		"bullet":      true,
		"arrow":       true,
		"separator":   true,
	}

	for name, value := range symbols {
		if !validSymbolNames[name] {
			return fmt.Errorf("unknown symbol name: %s", name)
		}

		// Validate symbol value (basic checks)
		if len(value) > 10 { // Reasonable limit for symbols
			return fmt.Errorf("symbol %s too long (max 10 characters): %s", name, value)
		}

		// Don't allow control characters
		for _, r := range value {
			if r < 32 && r != 9 && r != 10 && r != 13 { // Allow tab, newline, carriage return
				return fmt.Errorf("symbol %s contains invalid control character", name)
			}
		}
	}

	return nil
}
