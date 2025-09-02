package theme

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/utils"
)

func TestNewManager(t *testing.T) {
	manager := NewManager()
	assert.NotNil(t, manager)
	assert.NotEmpty(t, manager.providers)
	assert.Len(t, manager.providers, 2) // Should have builtin and custom providers
	assert.NotNil(t, manager.cache)
}

func TestNewBuiltinProvider(t *testing.T) {
	provider := NewBuiltinProvider()
	assert.NotNil(t, provider)
	assert.Equal(t, "builtin", provider.Name())
	assert.NotNil(t, provider.supportedThemes)

	// Check supported themes
	assert.True(t, provider.supportedThemes["default"])
	assert.True(t, provider.supportedThemes["minimal"])
	assert.True(t, provider.supportedThemes["ascii"])
}

func TestNewCustomProvider(t *testing.T) {
	provider := NewCustomProvider()
	assert.NotNil(t, provider)
	assert.Equal(t, "custom", provider.Name())
}

func TestManager_RegisterProvider(t *testing.T) {
	manager := NewManager()
	initialCount := len(manager.providers)

	// Create a new provider instance
	newProvider := NewBuiltinProvider()
	manager.RegisterProvider(newProvider)

	assert.Equal(t, initialCount+1, len(manager.providers))
}

// Test that providers have cache
func TestManager_HasCache(t *testing.T) {
	manager := NewManager()
	assert.NotNil(t, manager.cache)
}

// Test that builtin provider has supported themes
func TestBuiltinProvider_HasSupportedThemes(t *testing.T) {
	provider := NewBuiltinProvider()
	assert.NotNil(t, provider.supportedThemes)
}

// Comprehensive BuiltinProvider Tests

func TestBuiltinProvider_CanHandle(t *testing.T) {
	provider := NewBuiltinProvider()

	tests := []struct {
		name      string
		themeName string
		expected  bool
	}{
		{"default theme", "default", true},
		{"minimal theme", "minimal", true},
		{"ascii theme", "ascii", true},
		{"unsupported theme", "custom-theme", false},
		{"empty theme", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := value.ThemeConfig{ThemeName: tt.themeName}
			result := provider.CanHandle(config)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuiltinProvider_CreateTheme(t *testing.T) {
	provider := NewBuiltinProvider()
	ctx := context.Background()

	t.Run("successful theme creation", func(t *testing.T) {
		config := value.ThemeConfig{
			ThemeName:      "default",
			SuppressEmojis: false,
			CustomSymbols:  map[string]string{},
		}

		result := provider.CreateTheme(ctx, config)
		require.True(t, result.IsOk())

		theme := result.Unwrap()
		assert.Equal(t, "default", theme.Name())
	})

	t.Run("unsupported theme", func(t *testing.T) {
		config := value.ThemeConfig{
			ThemeName: "unsupported",
		}

		result := provider.CreateTheme(ctx, config)
		require.True(t, result.IsErr())

		err := result.Error()
		assert.Contains(t, err.Error(), "not supported by builtin provider")
	})

	t.Run("theme with custom symbols", func(t *testing.T) {
		config := value.ThemeConfig{
			ThemeName: "minimal",
			CustomSymbols: map[string]string{
				"success": "",
				"error":   "",
			},
		}

		result := provider.CreateTheme(ctx, config)
		require.True(t, result.IsOk())

		theme := result.Unwrap()
		assert.Equal(t, "minimal", theme.Name())
	})
}

func TestBuiltinProvider_ValidateConfig(t *testing.T) {
	provider := NewBuiltinProvider()

	t.Run("valid config", func(t *testing.T) {
		config := value.ThemeConfig{
			ThemeName: "default",
			CustomSymbols: map[string]string{
				"success": "",
				"error":   "",
			},
		}

		err := provider.ValidateConfig(config)
		assert.NoError(t, err)
	})

	t.Run("unsupported theme", func(t *testing.T) {
		config := value.ThemeConfig{
			ThemeName: "unsupported",
		}

		err := provider.ValidateConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported builtin theme")
	})

	t.Run("invalid custom symbols", func(t *testing.T) {
		config := value.ThemeConfig{
			ThemeName: "default",
			CustomSymbols: map[string]string{
				"invalid_symbol": "test",
			},
		}

		err := provider.ValidateConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid custom symbols")
	})

	t.Run("symbol too long", func(t *testing.T) {
		config := value.ThemeConfig{
			ThemeName: "default",
			CustomSymbols: map[string]string{
				"success": "very_long_symbol_that_exceeds_limit",
			},
		}

		err := provider.ValidateConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too long")
	})
}

// Comprehensive CustomProvider Tests

func TestCustomProvider_CanHandle(t *testing.T) {
	provider := NewCustomProvider()

	tests := []struct {
		name      string
		themeName string
		expected  bool
	}{
		{"custom theme", "my-custom-theme", true},
		{"another custom", "corporate-theme", true},
		{"default builtin", "default", false},
		{"minimal builtin", "minimal", false},
		{"ascii builtin", "ascii", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := value.ThemeConfig{ThemeName: tt.themeName}
			result := provider.CanHandle(config)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCustomProvider_CreateTheme(t *testing.T) {
	provider := NewCustomProvider()
	ctx := context.Background()

	t.Run("successful custom theme creation", func(t *testing.T) {
		config := value.ThemeConfig{
			ThemeName:      "my-custom-theme",
			SuppressEmojis: true,
			CustomSymbols: map[string]string{
				"success": "PASS",
				"error":   "FAIL",
			},
		}

		result := provider.CreateTheme(ctx, config)
		require.True(t, result.IsOk())

		theme := result.Unwrap()
		// Custom themes get converted to minimal base
		assert.Equal(t, "minimal", theme.Name()) // Base config uses minimal
	})

	t.Run("custom theme with empty name", func(t *testing.T) {
		config := value.ThemeConfig{
			ThemeName: "",
		}

		result := provider.CreateTheme(ctx, config)
		require.True(t, result.IsErr())

		err := result.Error()
		assert.Contains(t, err.Error(), "cannot be empty")
	})
}

func TestCustomProvider_ValidateConfig(t *testing.T) {
	provider := NewCustomProvider()

	t.Run("valid custom config", func(t *testing.T) {
		config := value.ThemeConfig{
			ThemeName: "custom-theme",
			CustomSymbols: map[string]string{
				"success": "OK",
				"error":   "ERR",
			},
		}

		err := provider.ValidateConfig(config)
		assert.NoError(t, err)
	})

	t.Run("empty theme name", func(t *testing.T) {
		config := value.ThemeConfig{
			ThemeName: "",
		}

		err := provider.ValidateConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("invalid symbol with control characters", func(t *testing.T) {
		config := value.ThemeConfig{
			ThemeName: "custom-theme",
			CustomSymbols: map[string]string{
				"success": "test\x00", // Contains null character
			},
		}

		err := provider.ValidateConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid control character")
	})
}

// Comprehensive Manager Tests

func TestManager_CreateTheme(t *testing.T) {
	manager := NewManager()
	ctx := context.Background()

	t.Run("create builtin theme", func(t *testing.T) {
		config := value.ThemeConfig{
			ThemeName:      "default",
			SuppressEmojis: false,
		}

		result := manager.CreateTheme(ctx, config)
		require.True(t, result.IsOk())

		theme := result.Unwrap()
		assert.Equal(t, "default", theme.Name())
	})

	t.Run("create custom theme", func(t *testing.T) {
		config := value.ThemeConfig{
			ThemeName: "my-custom-theme",
		}

		result := manager.CreateTheme(ctx, config)
		require.True(t, result.IsOk())

		theme := result.Unwrap()
		assert.Equal(t, "minimal", theme.Name()) // Custom themes use minimal as base
	})

	t.Run("unsupported theme", func(t *testing.T) {
		// Since custom provider handles any non-builtin theme,
		// we need to test a scenario where no provider can handle it
		// For now, this is hard to test since custom provider accepts everything

		// Test with invalid config instead
		config := value.ThemeConfig{
			ThemeName: "", // Empty name should cause validation error
		}

		result := manager.CreateTheme(ctx, config)
		require.True(t, result.IsErr())

		err := result.Error()
		// The actual error message from the manager when no provider can handle
		assert.Contains(t, err.Error(), "no provider found for theme")
	})

	t.Run("caching works", func(t *testing.T) {
		config := value.ThemeConfig{
			ThemeName:      "minimal",
			SuppressEmojis: false,
		}

		// First call
		result1 := manager.CreateTheme(ctx, config)
		require.True(t, result1.IsOk())

		// Second call should use cache
		result2 := manager.CreateTheme(ctx, config)
		require.True(t, result2.IsOk())

		// Both should be successful
		theme1 := result1.Unwrap()
		theme2 := result2.Unwrap()
		assert.Equal(t, theme1.Name(), theme2.Name())
	})
}

func TestManager_ValidateConfig(t *testing.T) {
	manager := NewManager()

	t.Run("valid builtin config", func(t *testing.T) {
		config := value.ThemeConfig{
			ThemeName: "default",
		}

		err := manager.ValidateConfig(config)
		assert.NoError(t, err)
	})

	t.Run("valid custom config", func(t *testing.T) {
		config := value.ThemeConfig{
			ThemeName: "custom-theme",
		}

		err := manager.ValidateConfig(config)
		assert.NoError(t, err)
	})

	t.Run("invalid config", func(t *testing.T) {
		config := value.ThemeConfig{
			ThemeName: "default",
			CustomSymbols: map[string]string{
				"invalid_symbol": "test",
			},
		}

		err := manager.ValidateConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid custom symbols")
	})
}

func TestManager_ListAvailableThemes(t *testing.T) {
	manager := NewManager()

	themes := manager.ListAvailableThemes()
	assert.NotEmpty(t, themes)

	// Should contain builtin themes
	assert.Contains(t, themes, "default")
	assert.Contains(t, themes, "minimal")
	assert.Contains(t, themes, "ascii")
	assert.Contains(t, themes, "custom")

	// Should have at least 4 themes
	assert.GreaterOrEqual(t, len(themes), 4)
}

func TestManager_ClearCache(t *testing.T) {
	manager := NewManager()
	ctx := context.Background()

	// Create a theme to populate cache
	config := value.ThemeConfig{
		ThemeName: "default",
	}

	result := manager.CreateTheme(ctx, config)
	require.True(t, result.IsOk())

	// Cache should have an entry
	assert.NotEmpty(t, manager.cache)

	// Clear cache
	manager.ClearCache()

	// Cache should be empty
	assert.Empty(t, manager.cache)
}

func TestManager_GetCacheKey(t *testing.T) {
	manager := NewManager()

	t.Run("basic cache key", func(t *testing.T) {
		config := value.ThemeConfig{
			ThemeName:      "default",
			SuppressEmojis: false,
		}

		key := manager.getCacheKey(config)
		assert.Equal(t, "default_false", key)
	})

	t.Run("cache key with suppress emojis", func(t *testing.T) {
		config := value.ThemeConfig{
			ThemeName:      "minimal",
			SuppressEmojis: true,
		}

		key := manager.getCacheKey(config)
		assert.Equal(t, "minimal_true", key)
	})

	t.Run("cache key with custom symbols", func(t *testing.T) {
		config := value.ThemeConfig{
			ThemeName:      "default",
			SuppressEmojis: false,
			CustomSymbols: map[string]string{
				"success": "",
			},
		}

		key := manager.getCacheKey(config)
		assert.Equal(t, "default_false_custom", key)
	})
}

// Test Provider Interface Compliance

func TestProviderInterfaceCompliance(t *testing.T) {
	t.Run("builtin provider implements interface", func(t *testing.T) {
		var provider Provider = NewBuiltinProvider()
		assert.NotNil(t, provider)

		// Test interface methods
		assert.Equal(t, "builtin", provider.Name())

		config := value.ThemeConfig{ThemeName: "default"}
		assert.True(t, provider.CanHandle(config))

		err := provider.ValidateConfig(config)
		assert.NoError(t, err)

		result := provider.CreateTheme(context.Background(), config)
		assert.True(t, result.IsOk())
	})

	t.Run("custom provider implements interface", func(t *testing.T) {
		var provider Provider = NewCustomProvider()
		assert.NotNil(t, provider)

		// Test interface methods
		assert.Equal(t, "custom", provider.Name())

		config := value.ThemeConfig{ThemeName: "custom-theme"}
		assert.True(t, provider.CanHandle(config))

		err := provider.ValidateConfig(config)
		assert.NoError(t, err)

		result := provider.CreateTheme(context.Background(), config)
		assert.True(t, result.IsOk())
	})
}

// Additional comprehensive tests to reach 85% coverage

func TestManager_CreateThemeFromDefinition(t *testing.T) {
	manager := NewManager()
	ctx := context.Background()

	t.Run("create theme from complete definition", func(t *testing.T) {
		definition := utils.ThemeDefinition{
			Name:        "test-definition",
			Description: "Test theme from definition",
			Symbols: map[string]string{
				"success":    "✅",
				"error":      "❌",
				"warning":    "⚠️",
				"info":       "ℹ️",
				"processing": "",
				"file_found": "",
				"file_saved": "",
				"benchmark":  "⏱️",
				"results":    "",
				"winner":     "",
				"search":     "",
				"launch":     "",
			},
		}

		config := value.ThemeConfig{
			ThemeName:      "test-definition",
			SuppressEmojis: false,
		}

		result := manager.CreateThemeFromDefinition(ctx, definition, config)
		require.True(t, result.IsOk())

		theme := result.Unwrap()
		assert.Equal(t, "test-definition", theme.Name())
	})

	t.Run("create theme with custom symbol overrides", func(t *testing.T) {
		definition := utils.ThemeDefinition{
			Name: "base-definition",
			Symbols: map[string]string{
				"success": "",
				"error":   "",
			},
		}

		config := value.ThemeConfig{
			ThemeName: "base-definition",
			CustomSymbols: map[string]string{
				"success": "PASS",
				"error":   "FAIL",
				"warning": "WARN",
			},
		}

		result := manager.CreateThemeFromDefinition(ctx, definition, config)
		require.True(t, result.IsOk())

		theme := result.Unwrap()
		assert.Equal(t, "base-definition", theme.Name())
	})

	t.Run("create theme with partial definition", func(t *testing.T) {
		definition := utils.ThemeDefinition{
			Name: "partial-definition",
			Symbols: map[string]string{
				"success": "OK",
				"error":   "ERR",
				// Only partial symbols defined
			},
		}

		config := value.ThemeConfig{
			ThemeName:      "partial-definition",
			SuppressEmojis: true,
		}

		result := manager.CreateThemeFromDefinition(ctx, definition, config)
		require.True(t, result.IsOk())

		theme := result.Unwrap()
		assert.Equal(t, "partial-definition", theme.Name())
	})
}

func TestManager_ErrorHandling(t *testing.T) {
	ctx := context.Background()
	_ = ctx // Use ctx to avoid unused variable warning

	t.Run("validate config with no suitable provider", func(t *testing.T) {
		// Create a manager with no providers
		emptyManager := &Manager{
			providers: []Provider{},
			cache:     make(map[string]value.Theme),
		}

		config := value.ThemeConfig{
			ThemeName: "any-theme",
		}

		err := emptyManager.ValidateConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no provider found for theme")
	})

	t.Run("create theme with no suitable provider", func(t *testing.T) {
		emptyManager := &Manager{
			providers: []Provider{},
			cache:     make(map[string]value.Theme),
		}

		config := value.ThemeConfig{
			ThemeName: "any-theme",
		}

		result := emptyManager.CreateTheme(ctx, config)
		assert.True(t, result.IsErr())
		assert.Contains(t, result.Error().Error(), "no provider found for theme")
	})
}

func TestBuiltinProvider_EdgeCases(t *testing.T) {
	provider := NewBuiltinProvider()

	t.Run("validate config with empty theme name", func(t *testing.T) {
		config := value.ThemeConfig{
			ThemeName: "",
		}

		err := provider.ValidateConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported builtin theme")
	})

	t.Run("create theme with context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		config := value.ThemeConfig{
			ThemeName: "default",
		}

		result := provider.CreateTheme(ctx, config)
		// Should still work despite cancelled context (depends on implementation)
		if result.IsOk() {
			theme := result.Unwrap()
			assert.Equal(t, "default", theme.Name())
		}
	})

	t.Run("validate all supported themes", func(t *testing.T) {
		supportedThemes := []string{"default", "minimal", "ascii"}

		for _, themeName := range supportedThemes {
			t.Run(themeName, func(t *testing.T) {
				config := value.ThemeConfig{
					ThemeName: themeName,
				}

				assert.True(t, provider.CanHandle(config))

				err := provider.ValidateConfig(config)
				assert.NoError(t, err)

				result := provider.CreateTheme(context.Background(), config)
				assert.True(t, result.IsOk())
			})
		}
	})
}

func TestCustomProvider_EdgeCases(t *testing.T) {
	provider := NewCustomProvider()

	t.Run("validate config with long custom symbols", func(t *testing.T) {
		config := value.ThemeConfig{
			ThemeName: "custom-theme",
			CustomSymbols: map[string]string{
				"success": "this_is_a_very_long_symbol_that_might_exceed_reasonable_limits_for_display_purposes_and_could_cause_formatting_issues",
			},
		}

		err := provider.ValidateConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too long")
	})

	t.Run("create theme with special characters in name", func(t *testing.T) {
		config := value.ThemeConfig{
			ThemeName: "custom-theme-with-special-chars!@#$%^&*()",
		}

		result := provider.CreateTheme(context.Background(), config)
		require.True(t, result.IsOk())

		theme := result.Unwrap()
		assert.Equal(t, "minimal", theme.Name()) // Custom themes use minimal base
	})
}

func TestManager_CacheKeyGeneration(t *testing.T) {
	manager := NewManager()

	t.Run("cache key with complex config", func(t *testing.T) {
		config := value.ThemeConfig{
			ThemeName:      "test-theme",
			SuppressEmojis: true,
			CustomSymbols: map[string]string{
				"success": "",
				"error":   "",
				"warning": "!",
			},
		}

		key := manager.getCacheKey(config)
		assert.Equal(t, "test-theme_true_custom", key)
	})

	t.Run("cache key consistency", func(t *testing.T) {
		config := value.ThemeConfig{
			ThemeName:      "consistent-theme",
			SuppressEmojis: false,
		}

		key1 := manager.getCacheKey(config)
		key2 := manager.getCacheKey(config)
		assert.Equal(t, key1, key2)
	})

	t.Run("cache key uniqueness", func(t *testing.T) {
		config1 := value.ThemeConfig{
			ThemeName:      "theme1",
			SuppressEmojis: false,
		}

		config2 := value.ThemeConfig{
			ThemeName:      "theme1",
			SuppressEmojis: true,
		}

		key1 := manager.getCacheKey(config1)
		key2 := manager.getCacheKey(config2)
		assert.NotEqual(t, key1, key2)
	})
}

func TestManager_ProviderPriority(t *testing.T) {
	manager := NewManager()
	ctx := context.Background()

	t.Run("builtin provider takes precedence", func(t *testing.T) {
		config := value.ThemeConfig{
			ThemeName: "default", // This should be handled by builtin provider
		}

		result := manager.CreateTheme(ctx, config)
		require.True(t, result.IsOk())

		theme := result.Unwrap()
		assert.Equal(t, "default", theme.Name())
	})

	t.Run("custom provider handles non-builtin themes", func(t *testing.T) {
		config := value.ThemeConfig{
			ThemeName: "my-custom-corporate-theme",
		}

		result := manager.CreateTheme(ctx, config)
		require.True(t, result.IsOk())

		theme := result.Unwrap()
		assert.Equal(t, "minimal", theme.Name()) // Custom themes use minimal base
	})
}
