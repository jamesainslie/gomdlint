package commands

import (
	"context"

	"github.com/gomdlint/gomdlint/internal/app/service"
	"github.com/gomdlint/gomdlint/internal/domain/value"
)

// ThemedCommandHelper provides themed symbols for commands
type ThemedCommandHelper struct {
	theme value.Theme
}

// NewThemedCommandHelper creates a new themed command helper
func NewThemedCommandHelper() *ThemedCommandHelper {
	// Load theme configuration
	themeConfig := value.ThemeConfig{
		ThemeName:      "default",
		SuppressEmojis: false,
		CustomSymbols:  make(map[string]string),
	}

	// Try to load from config if available
	if configSource, err := loadConfigurationSource(""); err == nil {
		if themeData, exists := configSource.Config["theme"]; exists {
			// Handle both string and object theme formats
			if themeStr, ok := themeData.(string); ok {
				themeConfig.ThemeName = themeStr
			} else if themeMap, ok := themeData.(map[string]interface{}); ok {
				if themeName, exists := themeMap["theme"]; exists {
					if str, ok := themeName.(string); ok {
						themeConfig.ThemeName = str
					}
				}
				if suppressEmojis, exists := themeMap["suppress_emojis"]; exists {
					if b, ok := suppressEmojis.(bool); ok {
						themeConfig.SuppressEmojis = b
					}
				}
				if customSymbols, exists := themeMap["custom_symbols"]; exists {
					if symbolsMap, ok := customSymbols.(map[string]interface{}); ok {
						themeConfig.CustomSymbols = make(map[string]string)
						for k, v := range symbolsMap {
							if str, ok := v.(string); ok {
								themeConfig.CustomSymbols[k] = str
							}
						}
					}
				}
			}
		}
	}

	// Create theme service and load theme
	themeService := service.NewThemeService()
	ctx := context.Background()

	themeResult := themeService.CreateTheme(ctx, themeConfig)
	if themeResult.IsErr() {
		// Fallback to default if theme creation fails
		themeConfig.ThemeName = "default"
		themeResult = themeService.CreateTheme(ctx, themeConfig)
	}

	var theme value.Theme
	if themeResult.IsOk() {
		theme = themeResult.Unwrap()
	} else {
		// Ultimate fallback - create minimal theme
		theme = createFallbackTheme()
	}

	return &ThemedCommandHelper{
		theme: theme,
	}
}

// createFallbackTheme creates a basic theme when all else fails
func createFallbackTheme() value.Theme {
	config := value.ThemeConfig{
		ThemeName:      "default",
		SuppressEmojis: false,
		CustomSymbols:  make(map[string]string),
	}

	result := value.NewTheme(config)
	if result.IsOk() {
		return result.Unwrap()
	}

	// If even that fails, we have bigger problems, but return something
	panic("Unable to create fallback theme")
}

// Symbol access methods - these provide themed symbols that respect user configuration

func (h *ThemedCommandHelper) Success() string {
	return h.theme.Symbol("success")
}

func (h *ThemedCommandHelper) Error() string {
	return h.theme.Symbol("error")
}

func (h *ThemedCommandHelper) Warning() string {
	return h.theme.Symbol("warning")
}

func (h *ThemedCommandHelper) Info() string {
	return h.theme.Symbol("info")
}

func (h *ThemedCommandHelper) Processing() string {
	return h.theme.Symbol("processing")
}

func (h *ThemedCommandHelper) Launch() string {
	return h.theme.Symbol("launch")
}

func (h *ThemedCommandHelper) Winner() string {
	return h.theme.Symbol("winner")
}

func (h *ThemedCommandHelper) Search() string {
	return h.theme.Symbol("search")
}

func (h *ThemedCommandHelper) FileFound() string {
	return h.theme.Symbol("file_found")
}

func (h *ThemedCommandHelper) FileSaved() string {
	return h.theme.Symbol("file_saved")
}

func (h *ThemedCommandHelper) Benchmark() string {
	return h.theme.Symbol("benchmark")
}

func (h *ThemedCommandHelper) Results() string {
	return h.theme.Symbol("results")
}

// Custom symbols for commands not covered by main theme
func (h *ThemedCommandHelper) Question() string {
	// Check if custom symbol exists first
	if symbol := h.theme.Symbol("question"); symbol != "" {
		return symbol
	}
	if h.theme.IsEmojiSuppressed() {
		return "?"
	}
	return ""
}

func (h *ThemedCommandHelper) Settings() string {
	if symbol := h.theme.Symbol("settings"); symbol != "" {
		return symbol
	}
	if h.theme.IsEmojiSuppressed() {
		return "[CFG]"
	}
	return ""
}

func (h *ThemedCommandHelper) List() string {
	if symbol := h.theme.Symbol("list"); symbol != "" {
		return symbol
	}
	if h.theme.IsEmojiSuppressed() {
		return "[LIST]"
	}
	return ""
}

func (h *ThemedCommandHelper) Document() string {
	if symbol := h.theme.Symbol("document"); symbol != "" {
		return symbol
	}
	if h.theme.IsEmojiSuppressed() {
		return "[DOC]"
	}
	return ""
}

func (h *ThemedCommandHelper) Edit() string {
	if symbol := h.theme.Symbol("edit"); symbol != "" {
		return symbol
	}
	if h.theme.IsEmojiSuppressed() {
		return "[EDIT]"
	}
	return ""
}

func (h *ThemedCommandHelper) Location() string {
	if symbol := h.theme.Symbol("location"); symbol != "" {
		return symbol
	}
	if h.theme.IsEmojiSuppressed() {
		return "[PATH]"
	}
	return ""
}

func (h *ThemedCommandHelper) Tip() string {
	if symbol := h.theme.Symbol("tip"); symbol != "" {
		return symbol
	}
	if h.theme.IsEmojiSuppressed() {
		return "[TIP]"
	}
	return ""
}

func (h *ThemedCommandHelper) Theme() string {
	if symbol := h.theme.Symbol("theme"); symbol != "" {
		return symbol
	}
	if h.theme.IsEmojiSuppressed() {
		return "[THEME]"
	}
	return ""
}
