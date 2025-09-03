package utils

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetXDGPaths_WithEnvironmentVariables(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping XDG-specific tests on Windows - Windows uses AppData paths")
	}
	// Save original environment
	originalEnv := map[string]string{
		"XDG_CONFIG_HOME": os.Getenv("XDG_CONFIG_HOME"),
		"XDG_DATA_HOME":   os.Getenv("XDG_DATA_HOME"),
		"XDG_CACHE_HOME":  os.Getenv("XDG_CACHE_HOME"),
		"XDG_CONFIG_DIRS": os.Getenv("XDG_CONFIG_DIRS"),
		"XDG_DATA_DIRS":   os.Getenv("XDG_DATA_DIRS"),
		"HOME":            os.Getenv("HOME"),
	}

	// Restore environment after test
	defer func() {
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	// Set test environment variables
	os.Setenv("XDG_CONFIG_HOME", "/custom/config")
	os.Setenv("XDG_DATA_HOME", "/custom/data")
	os.Setenv("XDG_CACHE_HOME", "/custom/cache")
	os.Setenv("XDG_CONFIG_DIRS", "/etc/xdg:/usr/local/etc")
	os.Setenv("XDG_DATA_DIRS", "/usr/share:/usr/local/share")
	os.Setenv("HOME", "/home/testuser")

	paths := GetXDGPaths("testapp")

	assert.Equal(t, "/custom/config/testapp", paths.ConfigHome)
	assert.Equal(t, "/custom/data/testapp", paths.DataHome)
	assert.Equal(t, "/custom/cache/testapp", paths.CacheHome)
	assert.Equal(t, []string{"/etc/xdg/testapp", "/usr/local/etc/testapp"}, paths.ConfigDirs)
	assert.Equal(t, []string{"/usr/share/testapp", "/usr/local/share/testapp"}, paths.DataDirs)
}

func TestGetXDGPaths_WithDefaults(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping XDG-specific tests on Windows - Windows uses AppData paths")
	}
	// Save original environment
	originalEnv := map[string]string{
		"XDG_CONFIG_HOME": os.Getenv("XDG_CONFIG_HOME"),
		"XDG_DATA_HOME":   os.Getenv("XDG_DATA_HOME"),
		"XDG_CACHE_HOME":  os.Getenv("XDG_CACHE_HOME"),
		"XDG_CONFIG_DIRS": os.Getenv("XDG_CONFIG_DIRS"),
		"XDG_DATA_DIRS":   os.Getenv("XDG_DATA_DIRS"),
		"HOME":            os.Getenv("HOME"),
	}

	// Restore environment after test
	defer func() {
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	// Clear XDG environment variables to use defaults
	os.Unsetenv("XDG_CONFIG_HOME")
	os.Unsetenv("XDG_DATA_HOME")
	os.Unsetenv("XDG_CACHE_HOME")
	os.Unsetenv("XDG_CONFIG_DIRS")
	os.Unsetenv("XDG_DATA_DIRS")
	os.Setenv("HOME", "/home/testuser")

	paths := GetXDGPaths("testapp")

	assert.Equal(t, "/home/testuser/.config/testapp", paths.ConfigHome)
	assert.Equal(t, "/home/testuser/.local/share/testapp", paths.DataHome)
	assert.Equal(t, "/home/testuser/.cache/testapp", paths.CacheHome)
	assert.Equal(t, []string{"/etc/xdg/testapp"}, paths.ConfigDirs)
	assert.Equal(t, []string{"/usr/local/share/testapp", "/usr/share/testapp"}, paths.DataDirs)
}

func TestGetXDGPaths_NoHomeDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping XDG-specific tests on Windows - Windows uses AppData paths")
	}
	// Save original environment
	originalHome := os.Getenv("HOME")
	defer func() {
		if originalHome == "" {
			os.Unsetenv("HOME")
		} else {
			os.Setenv("HOME", originalHome)
		}
	}()

	// Clear HOME environment variable
	os.Unsetenv("HOME")
	os.Unsetenv("XDG_CONFIG_HOME")
	os.Unsetenv("XDG_DATA_HOME")
	os.Unsetenv("XDG_CACHE_HOME")

	paths := GetXDGPaths("testapp")

	// Should handle missing home directory gracefully
	assert.Equal(t, "", paths.ConfigHome)
	assert.Equal(t, "", paths.DataHome)
	assert.Equal(t, "", paths.CacheHome)
}

func TestGetXDGPaths_EmptyAppName(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping XDG-specific tests on Windows - Windows uses AppData paths")
	}
	// Save original environment
	originalEnv := map[string]string{
		"XDG_CONFIG_HOME": os.Getenv("XDG_CONFIG_HOME"),
		"XDG_DATA_HOME":   os.Getenv("XDG_DATA_HOME"),
		"XDG_CACHE_HOME":  os.Getenv("XDG_CACHE_HOME"),
		"XDG_CONFIG_DIRS": os.Getenv("XDG_CONFIG_DIRS"),
		"XDG_DATA_DIRS":   os.Getenv("XDG_DATA_DIRS"),
		"HOME":            os.Getenv("HOME"),
	}

	// Restore environment after test
	defer func() {
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	// Clear XDG environment variables to use defaults
	os.Unsetenv("XDG_CONFIG_HOME")
	os.Unsetenv("XDG_DATA_HOME")
	os.Unsetenv("XDG_CACHE_HOME")
	os.Unsetenv("XDG_CONFIG_DIRS")
	os.Unsetenv("XDG_DATA_DIRS")
	os.Setenv("HOME", "/home/testuser")

	paths := GetXDGPaths("")

	assert.Equal(t, "/home/testuser/.config", paths.ConfigHome)
	assert.Equal(t, "/home/testuser/.local/share", paths.DataHome)
	assert.Equal(t, "/home/testuser/.cache", paths.CacheHome)
	assert.Equal(t, []string{"/etc/xdg"}, paths.ConfigDirs)
	assert.Equal(t, []string{"/usr/local/share", "/usr/share"}, paths.DataDirs)
}

func TestGetXDGPaths_ComplexAppName(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping XDG-specific tests on Windows - Windows uses AppData paths")
	}
	// Save original environment
	originalHome := os.Getenv("HOME")
	defer func() {
		if originalHome == "" {
			os.Unsetenv("HOME")
		} else {
			os.Setenv("HOME", originalHome)
		}
	}()

	os.Setenv("HOME", "/home/testuser")

	paths := GetXDGPaths("my-complex-app-name_v2")

	assert.Equal(t, "/home/testuser/.config/my-complex-app-name_v2", paths.ConfigHome)
	assert.Equal(t, "/home/testuser/.local/share/my-complex-app-name_v2", paths.DataHome)
	assert.Equal(t, "/home/testuser/.cache/my-complex-app-name_v2", paths.CacheHome)
}

func TestGetXDGPaths_WindowsStyle(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping XDG Windows-style test on Windows - Windows uses native AppData paths")
	}

	// Save original environment
	originalEnv := map[string]string{
		"XDG_CONFIG_HOME": os.Getenv("XDG_CONFIG_HOME"),
		"HOME":            os.Getenv("HOME"),
	}
	defer func() {
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	// Test with Windows-style paths (using backslashes)
	os.Setenv("XDG_CONFIG_HOME", "C:\\Users\\TestUser\\Config")
	os.Setenv("HOME", "C:\\Users\\TestUser")

	paths := GetXDGPaths("testapp")

	// On Unix systems, filepath.Join will use forward slashes even with backslash input
	expected := filepath.Join("C:\\Users\\TestUser\\Config", "testapp")
	assert.Equal(t, expected, paths.ConfigHome)
}

func TestGetAppConfigDir_ExistingDirectory(t *testing.T) {
	tempDir := t.TempDir()

	// Create a config directory
	configDir := filepath.Join(tempDir, "testapp")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	// Mock XDGPaths
	paths := &XDGPaths{
		ConfigHome: configDir,
		ConfigDirs: []string{"/etc/xdg/testapp", "/usr/local/etc/testapp"},
	}

	result := GetAppConfigDir(paths)
	assert.Equal(t, configDir, result)
}

func TestGetAppConfigDir_FallbackToSystemDir(t *testing.T) {
	tempDir := t.TempDir()

	// Create system config directory
	systemConfigDir := filepath.Join(tempDir, "system", "testapp")
	err := os.MkdirAll(systemConfigDir, 0755)
	require.NoError(t, err)

	// Mock XDGPaths with non-existent user config dir
	paths := &XDGPaths{
		ConfigHome: filepath.Join(tempDir, "nonexistent", "testapp"),
		ConfigDirs: []string{systemConfigDir, "/another/nonexistent"},
	}

	result := GetAppConfigDir(paths)
	assert.Equal(t, systemConfigDir, result)
}

func TestGetAppConfigDir_NoExistingDirectories(t *testing.T) {
	tempDir := t.TempDir()

	// Mock XDGPaths with all non-existent directories
	paths := &XDGPaths{
		ConfigHome: filepath.Join(tempDir, "nonexistent", "testapp"),
		ConfigDirs: []string{
			filepath.Join(tempDir, "also", "nonexistent"),
			filepath.Join(tempDir, "another", "nonexistent"),
		},
	}

	result := GetAppConfigDir(paths)
	// Should return empty string if no directory exists (as documented)
	assert.Equal(t, "", result)
}

func TestGetAppDataDir_ExistingDirectory(t *testing.T) {
	tempDir := t.TempDir()

	// Create a data directory
	dataDir := filepath.Join(tempDir, "testapp")
	err := os.MkdirAll(dataDir, 0755)
	require.NoError(t, err)

	// Mock XDGPaths
	paths := &XDGPaths{
		DataHome: dataDir,
		DataDirs: []string{"/usr/share/testapp", "/usr/local/share/testapp"},
	}

	result := GetAppDataDir(paths)
	assert.Equal(t, dataDir, result)
}

func TestGetAppDataDir_FallbackToSystemDir(t *testing.T) {
	tempDir := t.TempDir()

	// Create system data directory
	systemDataDir := filepath.Join(tempDir, "system", "testapp")
	err := os.MkdirAll(systemDataDir, 0755)
	require.NoError(t, err)

	// Mock XDGPaths with non-existent user data dir
	paths := &XDGPaths{
		DataHome: filepath.Join(tempDir, "nonexistent", "testapp"),
		DataDirs: []string{systemDataDir, "/another/nonexistent"},
	}

	result := GetAppDataDir(paths)
	assert.Equal(t, systemDataDir, result)
}

func TestGetAppCacheDir(t *testing.T) {
	tempDir := t.TempDir()

	// Create cache directory
	cacheDir := filepath.Join(tempDir, "testapp")
	err := os.MkdirAll(cacheDir, 0755)
	require.NoError(t, err)

	// Mock XDGPaths
	paths := &XDGPaths{
		CacheHome: cacheDir,
	}

	result := GetAppCacheDir(paths)
	assert.Equal(t, cacheDir, result)
}

func TestGetAppCacheDir_NonExistent(t *testing.T) {
	tempDir := t.TempDir()

	// Mock XDGPaths with non-existent cache directory
	paths := &XDGPaths{
		CacheHome: filepath.Join(tempDir, "nonexistent", "testapp"),
	}

	result := GetAppCacheDir(paths)
	// Should return empty string if directory doesn't exist (as documented)
	assert.Equal(t, "", result)
}

func TestFindConfigFile_InUserConfigDir(t *testing.T) {
	tempDir := t.TempDir()

	// Create user config directory with config file
	userConfigDir := filepath.Join(tempDir, "user", "testapp")
	err := os.MkdirAll(userConfigDir, 0755)
	require.NoError(t, err)

	configFile := filepath.Join(userConfigDir, "config.json")
	err = os.WriteFile(configFile, []byte(`{"test": true}`), 0644)
	require.NoError(t, err)

	// Mock XDGPaths
	paths := &XDGPaths{
		ConfigHome: userConfigDir,
		ConfigDirs: []string{filepath.Join(tempDir, "system", "testapp")},
	}

	result := FindConfigFile(paths, "config.json")
	assert.Equal(t, configFile, result)
}

func TestFindConfigFile_InSystemConfigDir(t *testing.T) {
	tempDir := t.TempDir()

	// Create system config directory with config file
	systemConfigDir := filepath.Join(tempDir, "system", "testapp")
	err := os.MkdirAll(systemConfigDir, 0755)
	require.NoError(t, err)

	configFile := filepath.Join(systemConfigDir, "config.json")
	err = os.WriteFile(configFile, []byte(`{"test": true}`), 0644)
	require.NoError(t, err)

	// Mock XDGPaths with non-existent user config dir
	paths := &XDGPaths{
		ConfigHome: filepath.Join(tempDir, "nonexistent", "testapp"),
		ConfigDirs: []string{systemConfigDir},
	}

	result := FindConfigFile(paths, "config.json")
	assert.Equal(t, configFile, result)
}

func TestFindConfigFile_NotFound(t *testing.T) {
	tempDir := t.TempDir()

	// Mock XDGPaths with non-existent directories
	paths := &XDGPaths{
		ConfigHome: filepath.Join(tempDir, "nonexistent", "testapp"),
		ConfigDirs: []string{filepath.Join(tempDir, "also", "nonexistent")},
	}

	result := FindConfigFile(paths, "config.json")
	assert.Equal(t, "", result)
}

func TestFindConfigFile_MultipleFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Create both user and system config directories
	userConfigDir := filepath.Join(tempDir, "user", "testapp")
	systemConfigDir := filepath.Join(tempDir, "system", "testapp")

	err := os.MkdirAll(userConfigDir, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(systemConfigDir, 0755)
	require.NoError(t, err)

	// Create config files in both locations
	userConfigFile := filepath.Join(userConfigDir, "config.json")
	systemConfigFile := filepath.Join(systemConfigDir, "config.json")

	err = os.WriteFile(userConfigFile, []byte(`{"user": true}`), 0644)
	require.NoError(t, err)
	err = os.WriteFile(systemConfigFile, []byte(`{"system": true}`), 0644)
	require.NoError(t, err)

	// Mock XDGPaths
	paths := &XDGPaths{
		ConfigHome: userConfigDir,
		ConfigDirs: []string{systemConfigDir},
	}

	// Should prefer user config file
	result := FindConfigFile(paths, "config.json")
	assert.Equal(t, userConfigFile, result)
}

func TestParseXDGDirs_EmptyString(t *testing.T) {
	result := parseXDGDirs("")
	assert.Equal(t, []string{}, result)
}

func TestParseXDGDirs_SinglePath(t *testing.T) {
	result := parseXDGDirs("/etc/xdg")
	assert.Equal(t, []string{"/etc/xdg"}, result)
}

func TestParseXDGDirs_MultiplePaths(t *testing.T) {
	result := parseXDGDirs("/etc/xdg:/usr/local/etc:/opt/local/etc")
	expected := []string{"/etc/xdg", "/usr/local/etc", "/opt/local/etc"}
	assert.Equal(t, expected, result)
}

func TestParseXDGDirs_EmptyPathsInList(t *testing.T) {
	result := parseXDGDirs("/etc/xdg::/usr/local/etc:")
	expected := []string{"/etc/xdg", "/usr/local/etc"}
	assert.Equal(t, expected, result)
}

func TestParseXDGDirs_PathsWithSpaces(t *testing.T) {
	result := parseXDGDirs("/path with spaces:/another path:/normal")
	expected := []string{"/path with spaces", "/another path", "/normal"}
	assert.Equal(t, expected, result)
}

// TestGetWindowsPaths verifies Windows-specific path handling
func TestGetWindowsPaths(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific tests on non-Windows platforms")
	}

	paths := GetXDGPaths("testapp")

	// On Windows, we expect AppData paths
	assert.NotEmpty(t, paths.ConfigHome)
	assert.NotEmpty(t, paths.DataHome)
	assert.NotEmpty(t, paths.CacheHome)

	// Verify paths contain expected Windows patterns
	assert.Contains(t, paths.ConfigHome, "AppData")
	assert.Contains(t, paths.ConfigHome, "testapp")
	assert.Contains(t, paths.CacheHome, "Local")

	// Config and data should be the same on Windows
	assert.Equal(t, paths.ConfigHome, paths.DataHome)
}

// Test edge cases and error conditions
func TestXDGPaths_EdgeCases(t *testing.T) {
	t.Run("very_long_paths", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Skipping XDG-specific path tests on Windows - Windows uses AppData paths")
		}
		longPath := strings.Repeat("a", 1000)

		// Save original environment
		originalHome := os.Getenv("HOME")
		defer func() {
			if originalHome == "" {
				os.Unsetenv("HOME")
			} else {
				os.Setenv("HOME", originalHome)
			}
		}()

		os.Setenv("HOME", "/home/"+longPath)

		paths := GetXDGPaths("testapp")
		assert.Contains(t, paths.ConfigHome, longPath)
	})

	t.Run("special_characters_in_app_name", func(t *testing.T) {
		specialChars := "app-name_with.special~chars@#$%"

		// Save original environment
		originalHome := os.Getenv("HOME")
		defer func() {
			if originalHome == "" {
				os.Unsetenv("HOME")
			} else {
				os.Setenv("HOME", originalHome)
			}
		}()

		os.Setenv("HOME", "/home/testuser")

		paths := GetXDGPaths(specialChars)
		assert.Contains(t, paths.ConfigHome, specialChars)
	})
}

// Benchmark tests
func BenchmarkGetXDGPaths(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetXDGPaths("testapp")
	}
}

func BenchmarkFindConfigFile(b *testing.B) {
	tempDir := b.TempDir()

	// Create config directory with file
	configDir := filepath.Join(tempDir, "testapp")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(b, err)

	configFile := filepath.Join(configDir, "config.json")
	err = os.WriteFile(configFile, []byte(`{"test": true}`), 0644)
	require.NoError(b, err)

	paths := &XDGPaths{
		ConfigHome: configDir,
		ConfigDirs: []string{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FindConfigFile(paths, "config.json")
	}
}
