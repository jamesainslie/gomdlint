package utils

import (
	"os"
	"path/filepath"
	"strings"
)

// XDGPaths contains the XDG Base Directory paths for the application.
type XDGPaths struct {
	ConfigHome string   // User-specific configuration directory
	DataHome   string   // User-specific data directory
	CacheHome  string   // User-specific cache directory
	ConfigDirs []string // System-wide configuration directories
	DataDirs   []string // System-wide data directories
}

// GetXDGPaths returns the XDG Base Directory paths for the application.
// It follows the XDG Base Directory Specification:
// https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html
func GetXDGPaths(appName string) *XDGPaths {
	homeDir, _ := os.UserHomeDir()

	// XDG_CONFIG_HOME - defaults to ~/.config
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" && homeDir != "" {
		configHome = filepath.Join(homeDir, ".config")
	}
	if configHome != "" {
		configHome = filepath.Join(configHome, appName)
	}

	// XDG_DATA_HOME - defaults to ~/.local/share
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" && homeDir != "" {
		dataHome = filepath.Join(homeDir, ".local", "share")
	}
	if dataHome != "" {
		dataHome = filepath.Join(dataHome, appName)
	}

	// XDG_CACHE_HOME - defaults to ~/.cache
	cacheHome := os.Getenv("XDG_CACHE_HOME")
	if cacheHome == "" && homeDir != "" {
		cacheHome = filepath.Join(homeDir, ".cache")
	}
	if cacheHome != "" {
		cacheHome = filepath.Join(cacheHome, appName)
	}

	// XDG_CONFIG_DIRS - defaults to /etc/xdg
	configDirsEnv := os.Getenv("XDG_CONFIG_DIRS")
	if configDirsEnv == "" {
		configDirsEnv = "/etc/xdg"
	}

	var configDirs []string
	for _, dir := range strings.Split(configDirsEnv, ":") {
		if dir != "" {
			configDirs = append(configDirs, filepath.Join(dir, appName))
		}
	}

	// XDG_DATA_DIRS - defaults to /usr/local/share:/usr/share
	dataDirsEnv := os.Getenv("XDG_DATA_DIRS")
	if dataDirsEnv == "" {
		dataDirsEnv = "/usr/local/share:/usr/share"
	}

	var dataDirs []string
	for _, dir := range strings.Split(dataDirsEnv, ":") {
		if dir != "" {
			dataDirs = append(dataDirs, filepath.Join(dir, appName))
		}
	}

	return &XDGPaths{
		ConfigHome: configHome,
		DataHome:   dataHome,
		CacheHome:  cacheHome,
		ConfigDirs: configDirs,
		DataDirs:   dataDirs,
	}
}

// GetConfigSearchPaths returns all directories where configuration files should be searched.
// It returns paths in priority order: current directory first (for backward compatibility),
// then XDG user config, then XDG system config directories.
func (x *XDGPaths) GetConfigSearchPaths() []string {
	var paths []string

	// Current directory first for backward compatibility
	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths, cwd)
	}

	// User config directory
	if x.ConfigHome != "" {
		paths = append(paths, x.ConfigHome)
	}

	// System config directories
	paths = append(paths, x.ConfigDirs...)

	return paths
}

// GetConfigFilenames returns the configuration filenames to search for, in priority order.
func GetConfigFilenames() []string {
	return []string{
		"config.json",
		"config.yaml",
		"config.yml",
		".gomdlint.json",
		".gomdlint.yaml",
		".gomdlint.yml",
		// Legacy markdownlint compatibility
		".markdownlint.json",
		".markdownlint.yaml",
		".markdownlint.yml",
		"markdownlint.json",
		"markdownlint.yaml",
		"markdownlint.yml",
	}
}

// FindConfigFile searches for a configuration file using XDG paths and returns the first found.
// It returns the full path to the config file, or empty string if none found.
//
// Deprecated: Use FindAllConfigFiles for hierarchical configuration support.
func FindConfigFile(appName string) (string, error) {
	xdg := GetXDGPaths(appName)
	searchPaths := xdg.GetConfigSearchPaths()
	filenames := GetConfigFilenames()

	for _, path := range searchPaths {
		for _, filename := range filenames {
			configPath := filepath.Join(path, filename)
			if info, err := os.Stat(configPath); err == nil && !info.IsDir() {
				return configPath, nil
			}
		}
	}

	return "", nil
}

// ConfigFileLocation represents a found configuration file with metadata
type ConfigFileLocation struct {
	Path   string            // Full path to the config file
	Type   ConfigurationType // Type of configuration location
	Source string            // Human-readable description of the source
}

// ConfigurationType represents the type of configuration location
type ConfigurationType string

const (
	ConfigTypeProject ConfigurationType = "project" // Current directory (legacy)
	ConfigTypeUser    ConfigurationType = "user"    // XDG user directory
	ConfigTypeSystem  ConfigurationType = "system"  // XDG system directory
)

// FindAllConfigFiles searches for all configuration files in the XDG hierarchy.
// Returns configs in priority order: project -> user -> system (highest to lowest priority)
func FindAllConfigFiles(appName string) ([]ConfigFileLocation, error) {
	xdg := GetXDGPaths(appName)
	filenames := GetConfigFilenames()
	var found []ConfigFileLocation

	// 1. Check current directory (project config - highest priority)
	if cwd, err := os.Getwd(); err == nil {
		if projectConfig := findConfigInDirectory(cwd, filenames); projectConfig != "" {
			found = append(found, ConfigFileLocation{
				Path:   projectConfig,
				Type:   ConfigTypeProject,
				Source: "project directory (legacy)",
			})
		}
	}

	// 2. Check XDG user config directory
	if xdg.ConfigHome != "" {
		if userConfig := findConfigInDirectory(xdg.ConfigHome, filenames); userConfig != "" {
			found = append(found, ConfigFileLocation{
				Path:   userConfig,
				Type:   ConfigTypeUser,
				Source: "XDG user config",
			})
		}
	}

	// 3. Check XDG system config directories
	for _, systemDir := range xdg.ConfigDirs {
		if systemConfig := findConfigInDirectory(systemDir, filenames); systemConfig != "" {
			found = append(found, ConfigFileLocation{
				Path:   systemConfig,
				Type:   ConfigTypeSystem,
				Source: "XDG system config",
			})
			break // Only use first system config found
		}
	}

	return found, nil
}

// findConfigInDirectory searches for a config file in a specific directory
func findConfigInDirectory(dir string, filenames []string) string {
	for _, filename := range filenames {
		configPath := filepath.Join(dir, filename)
		if info, err := os.Stat(configPath); err == nil && !info.IsDir() {
			return configPath
		}
	}
	return ""
}

// GetConfigHierarchy returns configuration file hierarchy with search information.
// This includes both found and not-found locations for debugging.
func GetConfigHierarchy(appName string) (*ConfigHierarchy, error) {
	xdg := GetXDGPaths(appName)
	filenames := GetConfigFilenames()

	hierarchy := &ConfigHierarchy{
		SearchPaths: make([]SearchPath, 0),
		FoundFiles:  make([]ConfigFileLocation, 0),
	}

	// Build search paths with found files
	searchPaths := []struct {
		path    string
		cfgType ConfigurationType
		source  string
	}{
		{getWorkingDirectory(), ConfigTypeProject, "project directory (legacy)"},
	}

	if xdg.ConfigHome != "" {
		searchPaths = append(searchPaths, struct {
			path    string
			cfgType ConfigurationType
			source  string
		}{xdg.ConfigHome, ConfigTypeUser, "XDG user config"})
	}

	for _, systemDir := range xdg.ConfigDirs {
		searchPaths = append(searchPaths, struct {
			path    string
			cfgType ConfigurationType
			source  string
		}{systemDir, ConfigTypeSystem, "XDG system config"})
	}

	// Process each search path
	for _, sp := range searchPaths {
		searchPath := SearchPath{
			Path:      sp.path,
			Type:      sp.cfgType,
			Source:    sp.source,
			Filenames: make([]SearchFile, 0),
		}

		// Check each possible filename
		for _, filename := range filenames {
			configPath := filepath.Join(sp.path, filename)
			exists := false

			if info, err := os.Stat(configPath); err == nil && !info.IsDir() {
				exists = true
				// Add to found files if this is the first file found in this directory
				if len(searchPath.Filenames) == 0 || !hasFoundFile(searchPath.Filenames) {
					hierarchy.FoundFiles = append(hierarchy.FoundFiles, ConfigFileLocation{
						Path:   configPath,
						Type:   sp.cfgType,
						Source: sp.source,
					})
				}
			}

			searchPath.Filenames = append(searchPath.Filenames, SearchFile{
				Name:   filename,
				Path:   configPath,
				Exists: exists,
			})
		}

		hierarchy.SearchPaths = append(hierarchy.SearchPaths, searchPath)
	}

	return hierarchy, nil
}

// ConfigHierarchy represents the complete configuration hierarchy with search information
type ConfigHierarchy struct {
	SearchPaths []SearchPath         // All searched paths with details
	FoundFiles  []ConfigFileLocation // Actually found config files
}

// SearchPath represents a directory that was searched for config files
type SearchPath struct {
	Path      string            // Directory path
	Type      ConfigurationType // Type of config location
	Source    string            // Human-readable description
	Filenames []SearchFile      // Files that were checked
}

// SearchFile represents a specific config file that was checked
type SearchFile struct {
	Name   string // Filename
	Path   string // Full path
	Exists bool   // Whether the file exists
}

// getWorkingDirectory returns the current working directory or empty string on error
func getWorkingDirectory() string {
	if cwd, err := os.Getwd(); err == nil {
		return cwd
	}
	return ""
}

// hasFoundFile checks if any file in the search files list exists
func hasFoundFile(files []SearchFile) bool {
	for _, file := range files {
		if file.Exists {
			return true
		}
	}
	return false
}

// EnsureConfigDir creates the XDG config directory if it doesn't exist.
// Returns the path to the config directory.
func EnsureConfigDir(appName string) (string, error) {
	xdg := GetXDGPaths(appName)
	if xdg.ConfigHome == "" {
		return "", os.ErrNotExist
	}

	if err := os.MkdirAll(xdg.ConfigHome, 0755); err != nil {
		return "", err
	}

	return xdg.ConfigHome, nil
}

// GetDefaultConfigPath returns the default path where a new config file should be created.
// It prefers the XDG config directory over the current directory.
func GetDefaultConfigPath(appName string) (string, error) {
	configDir, err := EnsureConfigDir(appName)
	if err != nil {
		// Fall back to current directory
		if cwd, err := os.Getwd(); err == nil {
			return filepath.Join(cwd, ".markdownlint.json"), nil
		}
		return "", err
	}

	return filepath.Join(configDir, "config.json"), nil
}

// IsLegacyConfigFile returns true if the config file is in a legacy location.
func IsLegacyConfigFile(configPath string) bool {
	filename := filepath.Base(configPath)
	legacyFiles := []string{
		".markdownlint.json",
		".markdownlint.yaml",
		".markdownlint.yml",
		"markdownlint.json",
		"markdownlint.yaml",
		"markdownlint.yml",
	}

	for _, legacy := range legacyFiles {
		if filename == legacy {
			return true
		}
	}

	return false
}
