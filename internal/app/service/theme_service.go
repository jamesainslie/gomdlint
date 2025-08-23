package service

import (
	"context"

	"github.com/gomdlint/gomdlint/internal/app/provider/theme"
	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/functional"
	"github.com/gomdlint/gomdlint/internal/shared/utils"
)

// ThemeService provides theming functionality for the application.
// This follows the service pattern from clean architecture.
type ThemeService struct {
	manager     *theme.Manager
	themeManager *utils.ThemeManager
}

// NewThemeService creates a new theme service.
func NewThemeService() *ThemeService {
	themeManager, err := utils.NewThemeManager("gomdlint")
	if err != nil {
		// Fallback to basic manager if XDG setup fails
		return &ThemeService{
			manager:      theme.NewManager(),
			themeManager: nil,
		}
	}

	// Ensure built-in themes are installed
	_ = themeManager.InstallBuiltinThemes()

	return &ThemeService{
		manager:      theme.NewManager(),
		themeManager: themeManager,
	}
}

// NewThemeServiceWithManager creates a theme service with a custom manager.
func NewThemeServiceWithManager(manager *theme.Manager) *ThemeService {
	return &ThemeService{
		manager: manager,
	}
}

// CreateTheme creates a theme from the given configuration.
// Now loads theme definitions from the theme directory.
func (ts *ThemeService) CreateTheme(ctx context.Context, config value.ThemeConfig) functional.Result[value.Theme] {
	// If we have a theme manager, try to load from directory first
	if ts.themeManager != nil {
		if themeDefinition, err := ts.themeManager.LoadTheme(config.ThemeName); err == nil {
			// Convert ThemeDefinition to value.Theme
			return ts.manager.CreateThemeFromDefinition(ctx, *themeDefinition, config)
		}
	}
	
	// Fallback to original behavior
	return ts.manager.CreateTheme(ctx, config)
}

// ValidateConfig validates theme configuration.
func (ts *ThemeService) ValidateConfig(config value.ThemeConfig) error {
	return ts.manager.ValidateConfig(config)
}

// ListAvailableThemes returns available theme names.
func (ts *ThemeService) ListAvailableThemes() []string {
	var themes []string
	
	// Get themes from directory if available
	if ts.themeManager != nil {
		if themeDefinitions, err := ts.themeManager.ListThemes(); err == nil {
			for _, theme := range themeDefinitions {
				themes = append(themes, theme.Name)
			}
			return themes
		}
	}
	
	// Fallback to built-in themes
	return ts.manager.ListAvailableThemes()
}

// GetThemeManager returns the theme directory manager
func (ts *ThemeService) GetThemeManager() *utils.ThemeManager {
	return ts.themeManager
}

// ClearCache clears the theme cache.
func (ts *ThemeService) ClearCache() {
	ts.manager.ClearCache()
}
