package theme

import (
	"context"
	"fmt"

	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/functional"
	"github.com/gomdlint/gomdlint/internal/shared/utils"
)

// Provider defines the interface for theme providers.
// This follows the provider pattern from go-bootstrapper.
type Provider interface {
	// Name returns the unique name of the theme provider
	Name() string

	// CanHandle returns true if this provider can handle the given theme configuration
	CanHandle(config value.ThemeConfig) bool

	// CreateTheme creates a theme from the given configuration
	CreateTheme(ctx context.Context, config value.ThemeConfig) functional.Result[value.Theme]

	// ValidateConfig validates the theme configuration
	ValidateConfig(config value.ThemeConfig) error
}

// Manager manages theme providers and handles theme creation.
type Manager struct {
	providers []Provider
	cache     map[string]value.Theme // Simple cache for created themes
}

// NewManager creates a new theme manager with default providers.
func NewManager() *Manager {
	return &Manager{
		providers: []Provider{
			NewBuiltinProvider(),
			NewCustomProvider(),
		},
		cache: make(map[string]value.Theme),
	}
}

// RegisterProvider adds a new theme provider to the manager.
func (tm *Manager) RegisterProvider(provider Provider) {
	tm.providers = append(tm.providers, provider)
}

// CreateTheme creates a theme using the appropriate provider.
func (tm *Manager) CreateTheme(ctx context.Context, config value.ThemeConfig) functional.Result[value.Theme] {
	// Check cache first
	cacheKey := tm.getCacheKey(config)
	if cachedTheme, exists := tm.cache[cacheKey]; exists {
		return functional.Ok(cachedTheme)
	}

	// Find a provider that can handle this config
	for _, provider := range tm.providers {
		if provider.CanHandle(config) {
			result := provider.CreateTheme(ctx, config)
			if result.IsOk() {
				theme := result.Unwrap()
				tm.cache[cacheKey] = theme
				return functional.Ok(theme)
			}
		}
	}

	return functional.Err[value.Theme](fmt.Errorf("no provider found for theme: %s", config.ThemeName))
}

// ValidateConfig validates theme configuration using all providers.
func (tm *Manager) ValidateConfig(config value.ThemeConfig) error {
	for _, provider := range tm.providers {
		if provider.CanHandle(config) {
			return provider.ValidateConfig(config)
		}
	}
	return fmt.Errorf("no provider found for theme: %s", config.ThemeName)
}

// CreateThemeFromDefinition creates a theme from a ThemeDefinition loaded from the theme directory.
func (tm *Manager) CreateThemeFromDefinition(ctx context.Context, definition utils.ThemeDefinition, config value.ThemeConfig) functional.Result[value.Theme] {
	// Convert utils.ThemeDefinition to value.Theme
	// Create theme symbols from the definition
	symbols := value.ThemeSymbols{}
	
	// Map symbols from definition to theme symbols structure
	if val, ok := definition.Symbols["success"]; ok { symbols.Success = val }
	if val, ok := definition.Symbols["error"]; ok { symbols.Error = val }
	if val, ok := definition.Symbols["warning"]; ok { symbols.Warning = val }
	if val, ok := definition.Symbols["info"]; ok { symbols.Info = val }
	if val, ok := definition.Symbols["processing"]; ok { symbols.Processing = val }
	if val, ok := definition.Symbols["file_found"]; ok { symbols.FileFound = val }
	if val, ok := definition.Symbols["file_saved"]; ok { symbols.FileSaved = val }
	if val, ok := definition.Symbols["benchmark"]; ok { symbols.Benchmark = val }
	if val, ok := definition.Symbols["results"]; ok { symbols.Results = val }
	if val, ok := definition.Symbols["winner"]; ok { symbols.Winner = val }
	if val, ok := definition.Symbols["search"]; ok { symbols.Search = val }
	if val, ok := definition.Symbols["launch"]; ok { symbols.Launch = val }

	// Apply custom symbol overrides from config
	for key, value := range config.CustomSymbols {
		switch key {
		case "success":
			symbols.Success = value
		case "error":
			symbols.Error = value
		case "warning":
			symbols.Warning = value
		case "info":
			symbols.Info = value
		case "processing":
			symbols.Processing = value
		case "file_found":
			symbols.FileFound = value
		case "file_saved":
			symbols.FileSaved = value
		case "benchmark":
			symbols.Benchmark = value
		case "results":
			symbols.Results = value
		case "winner":
			symbols.Winner = value
		case "search":
			symbols.Search = value
		case "launch":
			symbols.Launch = value
		}
	}

	// Create a theme config with the converted data
	themeConfig := value.ThemeConfig{
		ThemeName:      definition.Name,
		SuppressEmojis: config.SuppressEmojis,
		CustomSymbols:  config.CustomSymbols,
	}

	// Use the existing NewTheme function
	return value.NewTheme(themeConfig)
}

// ListAvailableThemes returns a list of available theme names.
func (tm *Manager) ListAvailableThemes() []string {
	themes := make(map[string]bool)
	for _, provider := range tm.providers {
		// For now, we'll return common theme names
		// In a more sophisticated implementation, providers could expose available themes
		switch provider.Name() {
		case "builtin":
			themes["default"] = true
			themes["minimal"] = true
			themes["ascii"] = true
		case "custom":
			themes["custom"] = true
		}
	}

	result := make([]string, 0, len(themes))
	for theme := range themes {
		result = append(result, theme)
	}
	return result
}

// getCacheKey generates a cache key for theme configuration.
func (tm *Manager) getCacheKey(config value.ThemeConfig) string {
	// Simple cache key based on theme name and suppress emojis flag
	key := fmt.Sprintf("%s_%t", config.ThemeName, config.SuppressEmojis)

	// Add custom symbols to key if present
	if len(config.CustomSymbols) > 0 {
		key += "_custom"
	}

	return key
}

// ClearCache clears the theme cache.
func (tm *Manager) ClearCache() {
	tm.cache = make(map[string]value.Theme)
}
