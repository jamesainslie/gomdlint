package value

import (
	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// ThemeConfig represents the configuration for theming.
// This follows the functional programming principles from go-bootstrapper.
type ThemeConfig struct {
	// Theme selection
	ThemeName string `json:"theme" yaml:"theme" mapstructure:"theme"`

	// Global emoji control
	SuppressEmojis bool `json:"suppress_emojis" yaml:"suppress_emojis" mapstructure:"suppress_emojis"`

	// Custom theme overrides
	CustomSymbols map[string]string `json:"custom_symbols" yaml:"custom_symbols" mapstructure:"custom_symbols"`
}

// Theme represents a complete set of visual symbols and formatting.
// Immutable by design following functional programming principles.
type Theme struct {
	name        string
	symbols     ThemeSymbols
	colors      ThemeColors
	suppressAll bool
}

// ThemeSymbols defines all visual symbols used in output.
type ThemeSymbols struct {
	// Status indicators
	Success    string `json:"success"`
	Error      string `json:"error"`
	Warning    string `json:"warning"`
	Info       string `json:"info"`
	Processing string `json:"processing"`

	// File operations
	FileFound string `json:"file_found"`
	FileSaved string `json:"file_saved"`

	// Performance indicators
	Benchmark   string `json:"benchmark"`
	Performance string `json:"performance"`
	Winner      string `json:"winner"`
	Results     string `json:"results"`

	// Actions
	Search string `json:"search"`
	Launch string `json:"launch"`

	// Bullet points and separators
	Bullet    string `json:"bullet"`
	Arrow     string `json:"arrow"`
	Separator string `json:"separator"`
}

// ThemeColors defines color codes for different message types.
type ThemeColors struct {
	Reset   string
	Red     string
	Green   string
	Yellow  string
	Blue    string
	Magenta string
	Cyan    string
	White   string
}

// NewThemeConfig creates a new ThemeConfig with sensible defaults.
func NewThemeConfig() ThemeConfig {
	return ThemeConfig{
		ThemeName:      "default",
		SuppressEmojis: false,
		CustomSymbols:  make(map[string]string),
	}
}

// NewTheme creates a new Theme with the specified configuration.
func NewTheme(config ThemeConfig) functional.Result[Theme] {
	theme, err := createThemeFromConfig(config)
	if err != nil {
		return functional.Err[Theme](err)
	}
	return functional.Ok(theme)
}

// createThemeFromConfig creates a theme based on configuration.
func createThemeFromConfig(config ThemeConfig) (Theme, error) {
	var baseSymbols ThemeSymbols

	switch config.ThemeName {
	case "default":
		baseSymbols = defaultThemeSymbols()
	case "minimal":
		baseSymbols = minimalThemeSymbols()
	case "ascii":
		baseSymbols = asciiThemeSymbols()
	default:
		baseSymbols = defaultThemeSymbols()
	}

	// Apply custom symbol overrides
	finalSymbols := applyCustomSymbols(baseSymbols, config.CustomSymbols)

	// Apply emoji suppression if requested
	if config.SuppressEmojis {
		finalSymbols = suppressEmojis(finalSymbols)
	}

	return Theme{
		name:        config.ThemeName,
		symbols:     finalSymbols,
		colors:      standardColors(),
		suppressAll: config.SuppressEmojis,
	}, nil
}

// defaultThemeSymbols returns the default emoji-rich theme.
func defaultThemeSymbols() ThemeSymbols {
	return ThemeSymbols{
		Success:     "✅",
		Error:       "❌",
		Warning:     "⚠️",
		Info:        "ℹ️",
		Processing:  "🔍",
		FileFound:   "📁",
		FileSaved:   "📁",
		Benchmark:   "🚀",
		Performance: "📊",
		Winner:      "🏆",
		Results:     "📈",
		Search:      "🔍",
		Launch:      "🚀",
		Bullet:      "•",
		Arrow:       "→",
		Separator:   "│",
	}
}

// minimalThemeSymbols returns a minimal theme with subtle symbols.
func minimalThemeSymbols() ThemeSymbols {
	return ThemeSymbols{
		Success:     "✓",
		Error:       "✗",
		Warning:     "!",
		Info:        "i",
		Processing:  "...",
		FileFound:   "*",
		FileSaved:   "*",
		Benchmark:   ">",
		Performance: "#",
		Winner:      "*",
		Results:     "#",
		Search:      "?",
		Launch:      ">",
		Bullet:      "•",
		Arrow:       "->",
		Separator:   "|",
	}
}

// asciiThemeSymbols returns a pure ASCII theme.
func asciiThemeSymbols() ThemeSymbols {
	return ThemeSymbols{
		Success:     "[OK]",
		Error:       "[ERROR]",
		Warning:     "[WARN]",
		Info:        "[INFO]",
		Processing:  "[...]",
		FileFound:   "[FILE]",
		FileSaved:   "[SAVED]",
		Benchmark:   "[BENCH]",
		Performance: "[PERF]",
		Winner:      "[BEST]",
		Results:     "[RESULTS]",
		Search:      "[SEARCH]",
		Launch:      "[START]",
		Bullet:      "*",
		Arrow:       "=>",
		Separator:   "|",
	}
}

// applyCustomSymbols applies custom symbol overrides to a base theme.
func applyCustomSymbols(base ThemeSymbols, custom map[string]string) ThemeSymbols {
	result := base // Create a copy

	if val, exists := custom["success"]; exists {
		result.Success = val
	}
	if val, exists := custom["error"]; exists {
		result.Error = val
	}
	if val, exists := custom["warning"]; exists {
		result.Warning = val
	}
	if val, exists := custom["info"]; exists {
		result.Info = val
	}
	if val, exists := custom["processing"]; exists {
		result.Processing = val
	}
	if val, exists := custom["file_found"]; exists {
		result.FileFound = val
	}
	if val, exists := custom["file_saved"]; exists {
		result.FileSaved = val
	}
	if val, exists := custom["benchmark"]; exists {
		result.Benchmark = val
	}
	if val, exists := custom["performance"]; exists {
		result.Performance = val
	}
	if val, exists := custom["winner"]; exists {
		result.Winner = val
	}
	if val, exists := custom["results"]; exists {
		result.Results = val
	}
	if val, exists := custom["search"]; exists {
		result.Search = val
	}
	if val, exists := custom["launch"]; exists {
		result.Launch = val
	}

	return result
}

// suppressEmojis removes all emoji-like symbols from a theme.
func suppressEmojis(symbols ThemeSymbols) ThemeSymbols {
	return ThemeSymbols{
		Success:     "",
		Error:       "",
		Warning:     "",
		Info:        "",
		Processing:  "",
		FileFound:   "",
		FileSaved:   "",
		Benchmark:   "",
		Performance: "",
		Winner:      "",
		Results:     "",
		Search:      "",
		Launch:      "",
		Bullet:      symbols.Bullet, // Keep text symbols
		Arrow:       symbols.Arrow,
		Separator:   symbols.Separator,
	}
}

// standardColors returns standard ANSI color codes.
func standardColors() ThemeColors {
	return ThemeColors{
		Reset:   "\033[0m",
		Red:     "\033[31m",
		Green:   "\033[32m",
		Yellow:  "\033[33m",
		Blue:    "\033[34m",
		Magenta: "\033[35m",
		Cyan:    "\033[36m",
		White:   "\033[37m",
	}
}

// Methods for accessing theme properties (immutable)

// Name returns the theme name.
func (t Theme) Name() string {
	return t.name
}

// Symbols returns the theme symbols.
func (t Theme) Symbols() ThemeSymbols {
	return t.symbols
}

// Colors returns the theme colors.
func (t Theme) Colors() ThemeColors {
	return t.colors
}

// IsEmojiSuppressed returns whether emojis are suppressed.
func (t Theme) IsEmojiSuppressed() bool {
	return t.suppressAll
}

// Symbol returns a specific symbol by name, with fallback.
func (t Theme) Symbol(name string) string {
	switch name {
	case "success":
		return t.symbols.Success
	case "error":
		return t.symbols.Error
	case "warning":
		return t.symbols.Warning
	case "info":
		return t.symbols.Info
	case "processing":
		return t.symbols.Processing
	case "file_found":
		return t.symbols.FileFound
	case "file_saved":
		return t.symbols.FileSaved
	case "benchmark":
		return t.symbols.Benchmark
	case "performance":
		return t.symbols.Performance
	case "winner":
		return t.symbols.Winner
	case "results":
		return t.symbols.Results
	case "search":
		return t.symbols.Search
	case "launch":
		return t.symbols.Launch
	case "bullet":
		return t.symbols.Bullet
	case "arrow":
		return t.symbols.Arrow
	case "separator":
		return t.symbols.Separator
	default:
		return ""
	}
}
