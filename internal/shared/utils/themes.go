package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ThemeDefinition represents a complete theme definition stored in the theme directory
type ThemeDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Author      string                 `json:"author,omitempty"`
	Version     string                 `json:"version,omitempty"`
	Symbols     map[string]string      `json:"symbols"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
}

// ThemeManager handles theme directory operations
type ThemeManager struct {
	configDir string
	themesDir string
}

// NewThemeManager creates a new theme manager for the given app
func NewThemeManager(appName string) (*ThemeManager, error) {
	xdg := GetXDGPaths(appName)
	if xdg.ConfigHome == "" {
		return nil, fmt.Errorf("XDG config directory not available")
	}
	
	themesDir := filepath.Join(xdg.ConfigHome, "themes")
	
	return &ThemeManager{
		configDir: xdg.ConfigHome,
		themesDir: themesDir,
	}, nil
}

// EnsureThemesDirectory creates the themes directory if it doesn't exist
func (tm *ThemeManager) EnsureThemesDirectory() error {
	return os.MkdirAll(tm.themesDir, 0755)
}

// GetThemesDirectory returns the path to the themes directory
func (tm *ThemeManager) GetThemesDirectory() string {
	return tm.themesDir
}

// ListThemes returns all available themes
func (tm *ThemeManager) ListThemes() ([]ThemeDefinition, error) {
	if err := tm.EnsureThemesDirectory(); err != nil {
		return nil, fmt.Errorf("failed to ensure themes directory: %w", err)
	}
	
	entries, err := os.ReadDir(tm.themesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read themes directory: %w", err)
	}
	
	var themes []ThemeDefinition
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			themeName := strings.TrimSuffix(entry.Name(), ".json")
			theme, err := tm.LoadTheme(themeName)
			if err != nil {
				// Skip invalid themes but don't fail completely
				continue
			}
			themes = append(themes, *theme)
		}
	}
	
	return themes, nil
}

// LoadTheme loads a theme by name from the themes directory
func (tm *ThemeManager) LoadTheme(name string) (*ThemeDefinition, error) {
	if name == "" {
		return nil, fmt.Errorf("theme name cannot be empty")
	}
	
	// Sanitize theme name to prevent directory traversal
	name = filepath.Base(strings.TrimSpace(name))
	if name == "." || name == ".." {
		return nil, fmt.Errorf("invalid theme name")
	}
	
	themePath := filepath.Join(tm.themesDir, name+".json")
	
	data, err := os.ReadFile(themePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("theme '%s' not found", name)
		}
		return nil, fmt.Errorf("failed to read theme file: %w", err)
	}
	
	var theme ThemeDefinition
	if err := json.Unmarshal(data, &theme); err != nil {
		return nil, fmt.Errorf("failed to parse theme '%s': %w", name, err)
	}
	
	// Ensure the theme has a name
	if theme.Name == "" {
		theme.Name = name
	}
	
	return &theme, nil
}

// SaveTheme saves a theme to the themes directory
func (tm *ThemeManager) SaveTheme(theme *ThemeDefinition) error {
	if theme.Name == "" {
		return fmt.Errorf("theme name cannot be empty")
	}
	
	if err := tm.EnsureThemesDirectory(); err != nil {
		return fmt.Errorf("failed to ensure themes directory: %w", err)
	}
	
	// Sanitize theme name
	name := filepath.Base(strings.TrimSpace(theme.Name))
	if name == "." || name == ".." {
		return fmt.Errorf("invalid theme name")
	}
	
	themePath := filepath.Join(tm.themesDir, name+".json")
	
	// Pretty print the JSON
	data, err := json.MarshalIndent(theme, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal theme: %w", err)
	}
	
	if err := os.WriteFile(themePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write theme file: %w", err)
	}
	
	return nil
}

// DeleteTheme removes a theme from the themes directory
func (tm *ThemeManager) DeleteTheme(name string) error {
	if name == "" {
		return fmt.Errorf("theme name cannot be empty")
	}
	
	// Prevent deletion of built-in themes
	if isBuiltinTheme(name) {
		return fmt.Errorf("cannot delete built-in theme '%s'", name)
	}
	
	// Sanitize theme name
	name = filepath.Base(strings.TrimSpace(name))
	if name == "." || name == ".." {
		return fmt.Errorf("invalid theme name")
	}
	
	themePath := filepath.Join(tm.themesDir, name+".json")
	
	if err := os.Remove(themePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("theme '%s' not found", name)
		}
		return fmt.Errorf("failed to delete theme file: %w", err)
	}
	
	return nil
}

// ThemeExists checks if a theme exists in the themes directory
func (tm *ThemeManager) ThemeExists(name string) bool {
	if name == "" {
		return false
	}
	
	// Sanitize theme name
	name = filepath.Base(strings.TrimSpace(name))
	if name == "." || name == ".." {
		return false
	}
	
	themePath := filepath.Join(tm.themesDir, name+".json")
	_, err := os.Stat(themePath)
	return err == nil
}

// InstallBuiltinThemes installs the built-in themes if they don't already exist
func (tm *ThemeManager) InstallBuiltinThemes() error {
	if err := tm.EnsureThemesDirectory(); err != nil {
		return fmt.Errorf("failed to ensure themes directory: %w", err)
	}
	
	builtinThemes := getBuiltinThemes()
	
	for _, theme := range builtinThemes {
		if !tm.ThemeExists(theme.Name) {
			if err := tm.SaveTheme(&theme); err != nil {
				return fmt.Errorf("failed to install built-in theme '%s': %w", theme.Name, err)
			}
		}
	}
	
	return nil
}

// ValidateTheme validates a theme definition
func (tm *ThemeManager) ValidateTheme(theme *ThemeDefinition) error {
	if theme.Name == "" {
		return fmt.Errorf("theme name is required")
	}
	
	// Validate theme name
	name := strings.TrimSpace(theme.Name)
	if name == "" {
		return fmt.Errorf("theme name cannot be empty")
	}
	
	if strings.ContainsAny(name, "/\\:*?\"<>|") {
		return fmt.Errorf("theme name contains invalid characters")
	}
	
	if len(name) > 50 {
		return fmt.Errorf("theme name too long (max 50 characters)")
	}
	
	// Validate symbols
	if theme.Symbols == nil {
		theme.Symbols = make(map[string]string)
	}
	
	for key, value := range theme.Symbols {
		if key == "" {
			return fmt.Errorf("symbol key cannot be empty")
		}
		if len(value) > 10 {
			return fmt.Errorf("symbol '%s' too long (max 10 characters): %s", key, value)
		}
	}
	
	return nil
}

// getBuiltinThemes returns the built-in theme definitions
func getBuiltinThemes() []ThemeDefinition {
	return []ThemeDefinition{
		{
			Name:        "default",
			Description: "Rich emoji theme with full visual feedback",
			Author:      "gomdlint",
			Version:     "1.0.0",
			Symbols: map[string]string{
				"success":     "✅",
				"error":       "❌",
				"warning":     "⚠️",
				"info":        "ℹ️",
				"processing":  "",
				"launch":      "",
				"winner":      "",
				"search":      "",
				"file_found":  "",
				"file_saved":  "",
				"benchmark":   "",
				"results":     "",
			},
			Settings: map[string]interface{}{
				"use_colors": true,
			},
		},
		{
			Name:        "minimal",
			Description: "Simple ASCII symbols for clean output",
			Author:      "gomdlint",
			Version:     "1.0.0",
			Symbols: map[string]string{
				"success":     "[OK]",
				"error":       "[ERROR]",
				"warning":     "[WARN]",
				"info":        "[INFO]",
				"processing":  "[...]",
				"launch":      "[START]",
				"winner":      "[DONE]",
				"search":      "[SEARCH]",
				"file_found":  "[FOUND]",
				"file_saved":  "[SAVED]",
				"benchmark":   "[BENCH]",
				"results":     "[RESULT]",
			},
			Settings: map[string]interface{}{
				"use_colors": true,
			},
		},
		{
			Name:        "ascii",
			Description: "Pure text indicators for scripts and automation",
			Author:      "gomdlint",
			Version:     "1.0.0",
			Symbols: map[string]string{
				"success":     "PASS",
				"error":       "FAIL",
				"warning":     "WARN",
				"info":        "INFO",
				"processing":  "WORK",
				"launch":      "START",
				"winner":      "COMPLETE",
				"search":      "SEARCH",
				"file_found":  "FOUND",
				"file_saved":  "SAVED",
				"benchmark":   "BENCHMARK",
				"results":     "RESULTS",
			},
			Settings: map[string]interface{}{
				"use_colors": false,
			},
		},
	}
}

// isBuiltinTheme checks if a theme name is a built-in theme
func isBuiltinTheme(name string) bool {
	builtinNames := []string{"default", "minimal", "ascii"}
	for _, builtin := range builtinNames {
		if name == builtin {
			return true
		}
	}
	return false
}

// GetBuiltinThemeNames returns the names of all built-in themes
func GetBuiltinThemeNames() []string {
	return []string{"default", "minimal", "ascii"}
}
