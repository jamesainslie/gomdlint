package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/gomdlint/gomdlint/internal/app/service"
	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/utils"
	"github.com/spf13/cobra"
)

// ConfigurationSource represents a loaded configuration and its source files.
// Now supports hierarchical configuration merging from multiple sources.
type ConfigurationSource struct {
	Config      map[string]interface{} // The final merged configuration data
	SourceFiles []string               // Paths to all source files used (in merge order)
	Sources     []ConfigSource         // Detailed information about each source
	IsDefault   bool                   // True if using only default configuration
	IsHierarchy bool                   // True if multiple config files were merged
}

// ConfigSource represents a single configuration source in the hierarchy
type ConfigSource struct {
	Path   string                 // Path to the config file (empty for defaults)
	Type   ConfigSourceType       // Type of configuration source
	Config map[string]interface{} // Configuration data from this source
}

// ConfigSourceType represents the type of configuration source
type ConfigSourceType string

const (
	ConfigSourceTypeDefault ConfigSourceType = "default" // Built-in defaults
	ConfigSourceTypeSystem  ConfigSourceType = "system"  // XDG system config
	ConfigSourceTypeUser    ConfigSourceType = "user"    // XDG user config
	ConfigSourceTypeProject ConfigSourceType = "project" // Project directory
	ConfigSourceTypeCustom  ConfigSourceType = "custom"  // Explicitly specified file
)

// NewConfigCommand creates the config command for configuration management.
func NewConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration management",
		Long:  `Manage gomdlint configuration files and settings.`,
	}

	cmd.AddCommand(
		newConfigInitCommand(),
		newConfigValidateCommand(),
		newConfigShowCommand(),
		newConfigWhichCommand(),
		newConfigEditCommand(),
	)

	return cmd
}

func newConfigInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new configuration file",
		Long: `Create a new gomdlint configuration file.

By default, creates a configuration file in the XDG config directory (recommended).
Use --legacy to create in the current directory for backward compatibility.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			legacy, _ := cmd.Flags().GetBool("legacy")
			return initConfig(legacy)
		},
	}

	cmd.Flags().Bool("legacy", false, "Create config in current directory (.markdownlint.json) instead of XDG directory")
	return cmd
}

func newConfigValidateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "validate [config-file]",
		Short: "Validate configuration files",
		Long: `Validate the syntax and content of gomdlint configuration files.

By default, validates the same configuration hierarchy that would be used for linting.
Specify a config file path to validate only that specific file.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			configFile := ""
			if len(args) > 0 {
				configFile = args[0]
			}
			return validateConfig(configFile)
		},
	}
}

func newConfigShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show [config-file]",
		Short: "Show effective configuration",
		Long:  `Display the effective configuration that would be used for linting.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			configFile := ""
			if len(args) > 0 {
				configFile = args[0]
			}
			return showConfig(configFile)
		},
	}
}

func newConfigWhichCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "which",
		Short: "Show which configuration files are being used",
		Long: `Display the configuration files that would be used for linting.

By default, shows a simple tree of loaded configuration files.
Use --verbose to see detailed information including search paths, file sizes, and merge behavior.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")
			return whichConfig("", verbose)
		},
	}

	cmd.Flags().BoolP("verbose", "v", false, "Show detailed information including search paths and file metadata")
	return cmd
}

func newConfigEditCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit [config-file]",
		Short: "Edit configuration file",
		Long: `Edit a gomdlint configuration file in your default editor.

By default, opens the primary configuration file from the hierarchy.
Specify a config file path to edit a specific file.

If no configuration exists, creates a new one in the XDG config directory.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			configFile := ""
			if len(args) > 0 {
				configFile = args[0]
			}
			return editConfig(configFile)
		},
	}

	return cmd
}

func initConfig(legacy bool) error {
	const appName = "gomdlint"

	defaultConfig := map[string]interface{}{
		"default": true,
		"MD013": map[string]interface{}{
			"line_length": 120,
		},
		"MD033": false,     // Allow HTML
		"MD041": false,     // First line doesn't need to be h1
		"theme": "default", // Simple theme name selection
	}

	var configFile string
	var err error

	if legacy {
		// Create in current directory (legacy behavior)
		configFile = ".markdownlint.json"
	} else {
		// Create in XDG config directory (recommended)
		configFile, err = utils.GetDefaultConfigPath(appName)
		if err != nil {
			return fmt.Errorf("failed to determine config path: %w", err)
		}
	}

	// Check if file already exists
	if _, err := os.Stat(configFile); err == nil {
		return fmt.Errorf("configuration file %s already exists", configFile)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configFile), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write configuration file
	data, err := json.MarshalIndent(defaultConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	err = os.WriteFile(configFile, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	absPath, _ := filepath.Abs(configFile)
	fmt.Printf("Configuration file created: %s\n", absPath)

	if !legacy {
		fmt.Println("Created in XDG config directory (recommended)")
		fmt.Println("Use --legacy flag to create in current directory instead")
	} else {
		fmt.Println("Created in current directory (legacy location)")
		fmt.Println("Consider migrating to XDG config directory with 'gomdlint config init'")
	}

	return nil
}

func validateConfig(configFile string) error {
	configSource, err := loadConfigurationSource(configFile)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if configSource.IsDefault {
		fmt.Println("No configuration found - will use defaults")
		return nil
	}

	// Validate theme configuration if present
	if themeData, exists := configSource.Config["theme"]; exists {
		if err := validateThemeConfig(themeData); err != nil {
			return fmt.Errorf("invalid theme configuration: %w", err)
		}
		fmt.Println("Theme configuration is valid")
	}

	// Display validation results
	if configSource.IsDefault {
		fmt.Println("Default configuration is valid")
	} else if configSource.IsHierarchy {
		fmt.Printf("Hierarchical configuration is valid (%d sources merged)\n", len(configSource.Sources))
		for _, source := range configSource.Sources {
			fmt.Printf("  - %s (%s)\n", source.Path, source.Type)
		}
	} else {
		fmt.Printf("Configuration file %s is valid\n", configSource.Sources[0].Path)
	}
	fmt.Printf("Found %d configuration entries\n", len(configSource.Config))

	return nil
}

// validateThemeConfig validates theme configuration
func validateThemeConfig(themeData interface{}) error {
	// Handle new format: "theme": "theme_name"
	if themeStr, ok := themeData.(string); ok {
		// Simple theme name - validate that the theme exists
		themeConfig := value.ThemeConfig{
			ThemeName:      themeStr,
			SuppressEmojis: false,
			CustomSymbols:  make(map[string]string),
		}

		themeService := service.NewThemeService()
		return themeService.ValidateConfig(themeConfig)
	}

	// Handle legacy format: "theme": { "theme": "name", ... }
	themeMap, ok := themeData.(map[string]interface{})
	if !ok {
		return fmt.Errorf("theme configuration must be a string or object")
	}

	// Extract theme config from object
	themeConfig := value.ThemeConfig{
		ThemeName:      "default",
		SuppressEmojis: false,
		CustomSymbols:  make(map[string]string),
	}

	if themeName, exists := themeMap["theme"]; exists {
		if str, ok := themeName.(string); ok {
			themeConfig.ThemeName = str
		} else {
			return fmt.Errorf("theme name must be a string")
		}
	}

	if suppressEmojis, exists := themeMap["suppress_emojis"]; exists {
		if b, ok := suppressEmojis.(bool); ok {
			themeConfig.SuppressEmojis = b
		} else {
			return fmt.Errorf("suppress_emojis must be a boolean")
		}
	}

	if customSymbols, exists := themeMap["custom_symbols"]; exists {
		if symbolsMap, ok := customSymbols.(map[string]interface{}); ok {
			themeConfig.CustomSymbols = make(map[string]string)
			for k, v := range symbolsMap {
				if str, ok := v.(string); ok {
					themeConfig.CustomSymbols[k] = str
				} else {
					return fmt.Errorf("custom symbol %s must be a string", k)
				}
			}
		} else {
			return fmt.Errorf("custom_symbols must be an object")
		}
	}

	// Use theme service to validate
	themeService := service.NewThemeService()
	return themeService.ValidateConfig(themeConfig)
}

func showConfig(configFile string) error {
	configSource, err := loadConfigurationSource(configFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Show configuration sources and hierarchy information
	if configSource.IsDefault {
		fmt.Println("# No configuration files found - using built-in defaults")
		fmt.Println("# Use 'gomdlint config which' to see all search paths")
		fmt.Println("# Use 'gomdlint config init' to create a configuration file")
	} else if configSource.IsHierarchy {
		fmt.Println("# Hierarchical configuration merged from multiple sources:")
		for i, source := range configSource.Sources {
			absPath, err := filepath.Abs(source.Path)
			if err != nil {
				absPath = source.Path
			}
			fmt.Printf("#   %d. %s (%s)\n", i+1, absPath, source.Type)
		}
		fmt.Println("#")
		fmt.Println("# Higher-numbered sources override lower-numbered sources")
	} else {
		// Single source
		source := configSource.Sources[0]
		absPath, err := filepath.Abs(source.Path)
		if err != nil {
			absPath = source.Path
		}
		fmt.Printf("# Configuration loaded from: %s (%s)\n", absPath, source.Type)
	}
	fmt.Println()

	// Pretty print the configuration
	data, err := json.MarshalIndent(configSource.Config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format configuration: %w", err)
	}

	fmt.Println(string(data))
	return nil
}

func whichConfig(configFile string, verbose bool) error {
	const appName = "gomdlint"
	configSource, err := loadConfigurationSource(configFile)
	if err != nil {
		return fmt.Errorf("failed to determine configuration source: %w", err)
	}

	if configSource.IsDefault {
		return fmt.Errorf("no configuration files found - using built-in defaults")
	}

	if verbose {
		return whichConfigVerbose(configSource, appName)
	}

	return whichConfigSimple(configSource)
}

// configStyles holds the styling for configuration output
type configStyles struct {
	header        lipgloss.Style
	treeBranch    lipgloss.Style
	path          lipgloss.Style
	sourceSystem  lipgloss.Style
	sourceUser    lipgloss.Style
	sourceProject lipgloss.Style
	sourceCustom  lipgloss.Style
	sourceDefault lipgloss.Style
	icon          lipgloss.Style
}

// newConfigStyles creates styled output for configuration display
func newConfigStyles(enableColors bool) configStyles {
	if !enableColors {
		// Return unstyled versions when colors are disabled
		return configStyles{
			header:        lipgloss.NewStyle(),
			treeBranch:    lipgloss.NewStyle(),
			path:          lipgloss.NewStyle(),
			sourceSystem:  lipgloss.NewStyle(),
			sourceUser:    lipgloss.NewStyle(),
			sourceProject: lipgloss.NewStyle(),
			sourceCustom:  lipgloss.NewStyle(),
			sourceDefault: lipgloss.NewStyle(),
			icon:          lipgloss.NewStyle(),
		}
	}

	return configStyles{
		header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12")), // Bright blue

		treeBranch: lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")), // Gray

		path: lipgloss.NewStyle().
			Foreground(lipgloss.Color("14")), // Cyan

		sourceSystem: lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")). // Red
			Bold(true),

		sourceUser: lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")). // Green
			Bold(true),

		sourceProject: lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")). // Yellow
			Bold(true),

		sourceCustom: lipgloss.NewStyle().
			Foreground(lipgloss.Color("13")). // Magenta
			Bold(true),

		sourceDefault: lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")), // Gray

		icon: lipgloss.NewStyle().
			Foreground(lipgloss.Color("6")), // Cyan
	}
}

// whichConfigSimple shows a clean tree of actually loaded configuration files
func whichConfigSimple(configSource *ConfigurationSource) error {
	// Check if colors are enabled via global flag or environment
	enableColors := shouldUseColors()
	styles := newConfigStyles(enableColors)

	// Get themed symbols
	helper := NewThemedCommandHelper()

	if configSource.IsHierarchy {
		// Header with icon
		header := fmt.Sprintf("%s Configuration hierarchy (%d files merged):", helper.List(), len(configSource.Sources))
		fmt.Println(styles.header.Render(header))
		fmt.Println()

		for i, source := range configSource.Sources {
			renderConfigTreeItem(source, i, len(configSource.Sources), styles)
		}
	} else {
		// Single configuration source
		source := configSource.Sources[0]
		sourceName := getSourceTypeStyled(source.Type, styles)

		fmt.Printf("%s %s %s %s\n",
			helper.Document(),
			styles.header.Render("Configuration:"),
			renderPath(source.Path, styles),
			sourceName)
	}

	return nil
}

// renderConfigTreeItem renders a single item in the configuration tree
func renderConfigTreeItem(source ConfigSource, index, total int, styles configStyles) {
	// Tree structure
	var treeChar string
	if index == total-1 {
		treeChar = "└─"
	} else {
		treeChar = "├─"
	}

	// Source icon and styling
	icon := getSourceIcon(source.Type)
	sourceName := getSourceTypeStyled(source.Type, styles)
	path := renderPath(source.Path, styles)

	// Priority indicator
	priority := fmt.Sprintf("[%d]", index+1)
	priorityStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Faint(true)

	fmt.Printf("%s %s %s %s %s\n",
		styles.treeBranch.Render(treeChar),
		styles.icon.Render(icon),
		path,
		sourceName,
		priorityStyle.Render(priority))
}

// renderPath formats the file path nicely, shortening home directory
func renderPath(filePath string, styles configStyles) string {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	// Shorten home directory path
	if homeDir, err := os.UserHomeDir(); err == nil {
		if strings.HasPrefix(absPath, homeDir) {
			absPath = "~" + strings.TrimPrefix(absPath, homeDir)
		}
	}

	return styles.path.Render(absPath)
}

// getSourceIcon returns an icon/emoji for the source type
func getSourceIcon(sourceType ConfigSourceType) string {
	// Get themed symbols
	helper := NewThemedCommandHelper()

	switch sourceType {
	case ConfigSourceTypeSystem:
		return helper.Settings() // Building for system-wide
	case ConfigSourceTypeUser:
		return helper.Info() // Person for user
	case ConfigSourceTypeProject:
		return helper.FileFound() // Folder for project
	case ConfigSourceTypeCustom:
		return helper.Settings() // Gear for custom
	case ConfigSourceTypeDefault:
		return helper.List() // Clipboard for defaults
	default:
		return helper.Document() // Document for unknown
	}
}

// getSourceTypeStyled returns a styled source type description
func getSourceTypeStyled(sourceType ConfigSourceType, styles configStyles) string {
	var style lipgloss.Style
	var text string

	switch sourceType {
	case ConfigSourceTypeSystem:
		style = styles.sourceSystem
		text = "system"
	case ConfigSourceTypeUser:
		style = styles.sourceUser
		text = "user"
	case ConfigSourceTypeProject:
		style = styles.sourceProject
		text = "project"
	case ConfigSourceTypeCustom:
		style = styles.sourceCustom
		text = "custom"
	case ConfigSourceTypeDefault:
		style = styles.sourceDefault
		text = "default"
	default:
		style = styles.sourceDefault
		text = "unknown"
	}

	return style.Render(text)
}

// shouldUseColors determines if colored output should be used
func shouldUseColors() bool {
	// Check if NO_COLOR environment variable is set (universal standard)
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// For now, default to true - in a full implementation this would check
	// the global --color flag, terminal capabilities, etc.
	return true
}

// whichConfigVerbose shows detailed information (original behavior)
func whichConfigVerbose(configSource *ConfigurationSource, appName string) error {
	if configSource.IsHierarchy {
		fmt.Printf("Hierarchical configuration active (%d sources merged):\n\n", len(configSource.Sources))

		for i, source := range configSource.Sources {
			absPath, err := filepath.Abs(source.Path)
			if err != nil {
				absPath = source.Path
			}

			fmt.Printf("%d. %s\n", i+1, absPath)
			fmt.Printf("   Type: %s\n", getSourceTypeDescription(source.Type))

			if info, err := os.Stat(source.Path); err == nil {
				fmt.Printf("   Size: %d bytes\n", info.Size())
				fmt.Printf("   Modified: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))
			}
			fmt.Println()
		}

		fmt.Println("Configuration merge order: system < user < project")
		fmt.Println("Higher priority sources override settings from lower priority sources.")
	} else {
		// Single configuration source
		source := configSource.Sources[0]
		absPath, err := filepath.Abs(source.Path)
		if err != nil {
			absPath = source.Path
		}

		fmt.Printf("Configuration file: %s\n", absPath)
		fmt.Printf("Type: %s\n", getSourceTypeDescription(source.Type))

		if info, err := os.Stat(source.Path); err == nil {
			fmt.Printf("File size: %d bytes\n", info.Size())
			fmt.Printf("Last modified: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))
		}
	}

	fmt.Println()
	fmt.Println("Search paths and detailed information:")
	displaySearchPaths(appName)

	return nil
}

// getSourceTypeShort returns a short description for configuration source types
func getSourceTypeShort(sourceType ConfigSourceType) string {
	switch sourceType {
	case ConfigSourceTypeDefault:
		return "default"
	case ConfigSourceTypeSystem:
		return "system"
	case ConfigSourceTypeUser:
		return "user"
	case ConfigSourceTypeProject:
		return "project"
	case ConfigSourceTypeCustom:
		return "custom"
	default:
		return "unknown"
	}
}

// getSourceTypeDescription returns a human-readable description for configuration source types
func getSourceTypeDescription(sourceType ConfigSourceType) string {
	switch sourceType {
	case ConfigSourceTypeDefault:
		return "Built-in defaults"
	case ConfigSourceTypeSystem:
		return "XDG system config (organization-wide)"
	case ConfigSourceTypeUser:
		return "XDG user config (personal preferences)"
	case ConfigSourceTypeProject:
		return "Project directory (team/project settings)"
	case ConfigSourceTypeCustom:
		return "Custom location (explicitly specified)"
	default:
		return "Unknown configuration source"
	}
}

// displaySearchPaths shows all the paths that are searched for configuration files.
func displaySearchPaths(appName string) {
	fmt.Println("Search order:")
	xdg := utils.GetXDGPaths(appName)
	searchPaths := xdg.GetConfigSearchPaths()
	filenames := utils.GetConfigFilenames()

	for i, path := range searchPaths {
		var pathType string
		switch i {
		case 0:
			pathType = " (current directory - legacy)"
		case 1:
			if xdg.ConfigHome != "" {
				pathType = " (XDG user config)"
			}
		default:
			pathType = " (XDG system config)"
		}

		fmt.Printf("\n%d. %s%s\n", i+1, path, pathType)
		for _, filename := range filenames {
			configPath := filepath.Join(path, filename)
			exists := ""
			if _, err := os.Stat(configPath); err != nil {
				exists = " (not found)"
			} else {
				exists = " ✓"
			}
			fmt.Printf("   - %s%s\n", filename, exists)
		}
	}
	fmt.Println()
}

// loadConfigurationSource loads configuration using hierarchical XDG-aware merging.
// This function merges configurations from multiple sources in priority order:
// system < user < project < explicit file
func loadConfigurationSource(configFile string) (*ConfigurationSource, error) {
	const appName = "gomdlint"

	if configFile != "" {
		// Explicit config file specified - load only that file (no hierarchy)
		return loadSingleConfigurationFile(configFile, ConfigSourceTypeCustom)
	}

	// Load hierarchical configuration
	return loadHierarchicalConfiguration(appName)
}

// loadHierarchicalConfiguration loads and merges configuration from the XDG hierarchy
func loadHierarchicalConfiguration(appName string) (*ConfigurationSource, error) {
	// Find all config files in hierarchy
	configFiles, err := utils.FindAllConfigFiles(appName)
	if err != nil {
		return nil, fmt.Errorf("error finding config files: %w", err)
	}

	if len(configFiles) == 0 {
		// No config files found, return defaults
		return &ConfigurationSource{
			Config:      getDefaultConfiguration(),
			SourceFiles: []string{},
			Sources: []ConfigSource{{
				Path:   "",
				Type:   ConfigSourceTypeDefault,
				Config: getDefaultConfiguration(),
			}},
			IsDefault:   true,
			IsHierarchy: false,
		}, nil
	}

	// Create merger and add configurations in reverse priority order (lowest to highest)
	merger := utils.NewConfigurationMerger()
	sources := make([]ConfigSource, 0)
	sourceFiles := make([]string, 0)

	// Add configurations in priority order (system -> user -> project)
	// We need to reverse the order since FindAllConfigFiles returns highest priority first
	for i := len(configFiles) - 1; i >= 0; i-- {
		configLoc := configFiles[i]

		config, err := loadConfigurationFile(configLoc.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to load config file %s: %w", configLoc.Path, err)
		}

		// Convert utils.ConfigurationType to our ConfigSourceType
		sourceType := convertConfigurationType(configLoc.Type)

		// Add to merger
		mergerSourceType := convertToMergerSourceType(sourceType)
		merger.AddSource(config, configLoc.Path, mergerSourceType)

		// Track sources
		sources = append(sources, ConfigSource{
			Path:   configLoc.Path,
			Type:   sourceType,
			Config: config,
		})
		sourceFiles = append(sourceFiles, configLoc.Path)
	}

	// Perform the merge
	mergedConfig := merger.Merge()

	return &ConfigurationSource{
		Config:      mergedConfig,
		SourceFiles: sourceFiles,
		Sources:     sources,
		IsDefault:   false,
		IsHierarchy: len(configFiles) > 1,
	}, nil
}

// loadSingleConfigurationFile loads a single configuration file without hierarchy
func loadSingleConfigurationFile(configFile string, sourceType ConfigSourceType) (*ConfigurationSource, error) {
	config, err := loadConfigurationFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load config file %s: %w", configFile, err)
	}

	return &ConfigurationSource{
		Config:      config,
		SourceFiles: []string{configFile},
		Sources: []ConfigSource{{
			Path:   configFile,
			Type:   sourceType,
			Config: config,
		}},
		IsDefault:   false,
		IsHierarchy: false,
	}, nil
}

// loadConfigurationFile loads and parses a single configuration file
func loadConfigurationFile(configPath string) (map[string]interface{}, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config map[string]interface{}

	// Try JSON first
	if err := json.Unmarshal(data, &config); err != nil {
		// TODO: Try YAML if JSON fails
		return nil, fmt.Errorf("failed to parse config file as JSON: %w", err)
	}

	return config, nil
}

// getDefaultConfiguration returns the built-in default configuration
func getDefaultConfiguration() map[string]interface{} {
	return map[string]interface{}{
		"default": true,
	}
}

// convertConfigurationType converts utils.ConfigurationType to our ConfigSourceType
func convertConfigurationType(configType utils.ConfigurationType) ConfigSourceType {
	switch configType {
	case utils.ConfigTypeProject:
		return ConfigSourceTypeProject
	case utils.ConfigTypeUser:
		return ConfigSourceTypeUser
	case utils.ConfigTypeSystem:
		return ConfigSourceTypeSystem
	default:
		return ConfigSourceTypeCustom
	}
}

// convertToMergerSourceType converts our ConfigSourceType to merger's ConfigSourceType
func convertToMergerSourceType(sourceType ConfigSourceType) utils.ConfigSourceType {
	switch sourceType {
	case ConfigSourceTypeSystem:
		return utils.ConfigSourceSystem
	case ConfigSourceTypeUser:
		return utils.ConfigSourceUser
	case ConfigSourceTypeProject:
		return utils.ConfigSourceProject
	default:
		return utils.ConfigSourceCLI
	}
}

// editConfig opens a configuration file for editing
func editConfig(configFile string) error {
	const appName = "gomdlint"

	// Get themed symbols
	helper := NewThemedCommandHelper()

	var targetFile string
	var isExistingFile bool

	if configFile != "" {
		// Specific file specified
		targetFile = configFile
		if _, err := os.Stat(targetFile); err == nil {
			isExistingFile = true
		}
	} else {
		// Find the primary configuration file from hierarchy
		configSource, err := loadConfigurationSource("")
		if err != nil || configSource.IsDefault {
			// No config exists, create new one in XDG directory
			xdgPaths := utils.GetXDGPaths(appName)
			if xdgPaths.ConfigHome == "" {
				return fmt.Errorf("unable to determine config directory. Set XDG_CONFIG_HOME or create ~/.config")
			}

			// Ensure config directory exists
			if err := os.MkdirAll(xdgPaths.ConfigHome, 0755); err != nil {
				return fmt.Errorf("failed to create config directory: %w", err)
			}

			targetFile = filepath.Join(xdgPaths.ConfigHome, "config.json")
			isExistingFile = false

			// Create a basic config file if it doesn't exist
			if _, err := os.Stat(targetFile); os.IsNotExist(err) {
				defaultConfig := map[string]interface{}{
					"default": true,
					"MD013": map[string]interface{}{
						"line_length": 120,
					},
					"MD033": false, // Allow HTML
					"MD041": false, // First line doesn't need to be h1
					"theme": "default",
				}

				data, err := json.MarshalIndent(defaultConfig, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to create default config: %w", err)
				}

				if err := os.WriteFile(targetFile, data, 0644); err != nil {
					return fmt.Errorf("failed to write default config: %w", err)
				}

				fmt.Printf("%s Created new configuration file\n", helper.Success())
			}
		} else {
			// Use the primary config file from hierarchy
			if configSource.IsHierarchy {
				// For hierarchical configs, edit the user-specific file (highest priority for user changes)
				for i := len(configSource.Sources) - 1; i >= 0; i-- {
					source := configSource.Sources[i]
					if source.Type == ConfigSourceTypeUser || source.Type == ConfigSourceTypeProject {
						targetFile = source.Path
						isExistingFile = true
						break
					}
				}
				// Fallback to first source if no user/project config found
				if targetFile == "" {
					targetFile = configSource.Sources[0].Path
					isExistingFile = true
				}
			} else {
				targetFile = configSource.Sources[0].Path
				isExistingFile = true
			}
		}
	}

	// Get editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		// Try common editors as fallbacks
		editors := []string{"nano", "vim", "vi"}
		for _, e := range editors {
			if _, err := exec.LookPath(e); err == nil {
				editor = e
				break
			}
		}
		if editor == "" {
			return fmt.Errorf("no editor found. Set EDITOR environment variable or install nano, vim, or vi")
		}
	}

	// Show what we're doing
	if isExistingFile {
		fmt.Printf("%s Opening configuration file in %s...\n", helper.Settings(), editor)
	} else {
		fmt.Printf("%s Creating and opening new configuration file in %s...\n", helper.Settings(), editor)
	}
	fmt.Printf("%s File: %s\n", helper.Location(), targetFile)

	// Execute the editor
	cmd := exec.Command(editor, targetFile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open editor '%s': %w\nTip: Set EDITOR environment variable to your preferred editor", editor, err)
	}

	fmt.Printf("%s Configuration editing completed\n", helper.Success())
	fmt.Printf("%s Validate with: gomdlint config validate\n", helper.Tip())
	fmt.Printf("%s View with: gomdlint config show\n", helper.Tip())

	return nil
}

// loadConfigurationSourceShared is a shared configuration loading function
// that can be used by all commands to maintain consistency.
func loadConfigurationSourceShared(configFile string) (*ConfigurationSource, error) {
	return loadConfigurationSource(configFile)
}
