package theme

import (
	"context"
	"fmt"

	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// BuiltinProvider provides built-in themes.
type BuiltinProvider struct {
	supportedThemes map[string]bool
}

// NewBuiltinProvider creates a new builtin theme provider.
func NewBuiltinProvider() *BuiltinProvider {
	return &BuiltinProvider{
		supportedThemes: map[string]bool{
			"default": true,
			"minimal": true,
			"ascii":   true,
		},
	}
}

// Name returns the provider name.
func (bp *BuiltinProvider) Name() string {
	return "builtin"
}

// CanHandle returns true if this provider can handle the theme.
func (bp *BuiltinProvider) CanHandle(config value.ThemeConfig) bool {
	return bp.supportedThemes[config.ThemeName]
}

// CreateTheme creates a builtin theme.
func (bp *BuiltinProvider) CreateTheme(ctx context.Context, config value.ThemeConfig) functional.Result[value.Theme] {
	if !bp.CanHandle(config) {
		return functional.Err[value.Theme](fmt.Errorf("theme %s not supported by builtin provider", config.ThemeName))
	}

	// Validate the configuration first
	if err := bp.ValidateConfig(config); err != nil {
		return functional.Err[value.Theme](err)
	}

	// Create the theme using the domain logic
	return value.NewTheme(config)
}

// ValidateConfig validates the configuration for builtin themes.
func (bp *BuiltinProvider) ValidateConfig(config value.ThemeConfig) error {
	if !bp.supportedThemes[config.ThemeName] {
		return fmt.Errorf("unsupported builtin theme: %s", config.ThemeName)
	}

	// Validate custom symbols if present
	if err := bp.validateCustomSymbols(config.CustomSymbols); err != nil {
		return fmt.Errorf("invalid custom symbols: %w", err)
	}

	return nil
}

// validateCustomSymbols validates custom symbol overrides.
func (bp *BuiltinProvider) validateCustomSymbols(symbols map[string]string) error {
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
	}

	return nil
}
