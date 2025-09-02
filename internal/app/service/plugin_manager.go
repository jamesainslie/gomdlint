package service

import (
	"context"
	"fmt"
	"path/filepath"
	"plugin"
	"sync"
	"time"

	gomdlintPlugin "github.com/gomdlint/gomdlint/pkg/gomdlint/plugin"
)

// PluginManager implements the plugin management interface
type PluginManager struct {
	plugins     map[string]gomdlintPlugin.Plugin
	pluginPaths map[string]string
	config      gomdlintPlugin.PluginConfig
	mutex       sync.RWMutex
	status      map[string]gomdlintPlugin.PluginStatus
}

// globalPluginManager is a singleton instance
var (
	globalPluginManager *PluginManager
	pluginManagerOnce   sync.Once
)

// NewPluginManager creates a new plugin manager instance
func NewPluginManager(config gomdlintPlugin.PluginConfig) *PluginManager {
	return &PluginManager{
		plugins:     make(map[string]gomdlintPlugin.Plugin),
		pluginPaths: make(map[string]string),
		config:      config,
		status:      make(map[string]gomdlintPlugin.PluginStatus),
	}
}

// GetGlobalPluginManager returns the global plugin manager instance
func GetGlobalPluginManager() *PluginManager {
	pluginManagerOnce.Do(func() {
		config := gomdlintPlugin.PluginConfig{
			DataDir:     "/tmp/gomdlint/data",
			ConfigDir:   "/tmp/gomdlint/config",
			CacheDir:    "/tmp/gomdlint/cache",
			LogLevel:    "info",
			Environment: make(map[string]string),
		}
		globalPluginManager = NewPluginManager(config)
	})
	return globalPluginManager
}

// LoadPlugin loads a plugin from a .so file (Go plugin)
func (pm *PluginManager) LoadPlugin(ctx context.Context, path string) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	startTime := time.Now()

	// Initialize status
	status := gomdlintPlugin.PluginStatus{
		Loaded:      false,
		Initialized: false,
		LoadTime:    startTime.Unix(),
	}

	defer func() {
		pm.status[filepath.Base(path)] = status
	}()

	// Load the plugin
	p, err := plugin.Open(path)
	if err != nil {
		status.Error = fmt.Errorf("failed to open plugin %s: %w", path, err)
		return status.Error
	}

	// Look for the required symbol
	sym, err := p.Lookup("NewPlugin")
	if err != nil {
		status.Error = fmt.Errorf("plugin %s missing NewPlugin function: %w", path, err)
		return status.Error
	}

	// Type assert to plugin constructor
	constructor, ok := sym.(func() gomdlintPlugin.Plugin)
	if !ok {
		status.Error = fmt.Errorf("plugin %s NewPlugin has wrong signature", path)
		return status.Error
	}

	// Create plugin instance
	pluginInstance := constructor()
	status.Name = pluginInstance.Name()
	status.Loaded = true

	// Initialize plugin
	if err := pluginInstance.Initialize(ctx, pm.config); err != nil {
		status.Error = fmt.Errorf("failed to initialize plugin %s: %w", path, err)
		return status.Error
	}

	status.Initialized = true
	status.RuleCount = len(pluginInstance.Rules())

	pm.plugins[pluginInstance.Name()] = pluginInstance
	pm.pluginPaths[pluginInstance.Name()] = path

	return nil
}

// UnloadPlugin removes a plugin from the manager
func (pm *PluginManager) UnloadPlugin(ctx context.Context, name string) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	// Shutdown plugin
	if err := plugin.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown plugin %s: %w", name, err)
	}

	// Remove from collections
	delete(pm.plugins, name)
	delete(pm.pluginPaths, name)
	delete(pm.status, name)

	return nil
}

// ReloadPlugin reloads a plugin
func (pm *PluginManager) ReloadPlugin(ctx context.Context, name string) error {
	pm.mutex.RLock()
	path, exists := pm.pluginPaths[name]
	pm.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	// Unload first
	if err := pm.UnloadPlugin(ctx, name); err != nil {
		return fmt.Errorf("failed to unload plugin during reload: %w", err)
	}

	// Reload
	return pm.LoadPlugin(ctx, path)
}

// LoadPluginsFromDirectory scans directory for .so files
func (pm *PluginManager) LoadPluginsFromDirectory(ctx context.Context, dir string) error {
	matches, err := filepath.Glob(filepath.Join(dir, "*.so"))
	if err != nil {
		return fmt.Errorf("failed to scan plugin directory %s: %w", dir, err)
	}

	var errors []error
	for _, match := range matches {
		if err := pm.LoadPlugin(ctx, match); err != nil {
			errors = append(errors, err)
			// Continue loading other plugins
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to load %d plugins: %v", len(errors), errors)
	}

	return nil
}

// ScanForPlugins scans multiple paths for plugin files
func (pm *PluginManager) ScanForPlugins(ctx context.Context, paths []string) ([]string, error) {
	var pluginFiles []string

	for _, path := range paths {
		matches, err := filepath.Glob(filepath.Join(path, "*.so"))
		if err != nil {
			continue // Skip invalid paths
		}
		pluginFiles = append(pluginFiles, matches...)
	}

	return pluginFiles, nil
}

// GetPlugin returns a specific plugin by name
func (pm *PluginManager) GetPlugin(name string) (gomdlintPlugin.Plugin, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
	}

	return plugin, nil
}

// GetAllPlugins returns all loaded plugins
func (pm *PluginManager) GetAllPlugins() map[string]gomdlintPlugin.Plugin {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	// Return a copy to prevent external modification
	result := make(map[string]gomdlintPlugin.Plugin)
	for name, plugin := range pm.plugins {
		result[name] = plugin
	}

	return result
}

// GetAllCustomRules returns all rules from all loaded plugins
func (pm *PluginManager) GetAllCustomRules() []gomdlintPlugin.CustomRule {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	var rules []gomdlintPlugin.CustomRule
	for _, p := range pm.plugins {
		rules = append(rules, p.Rules()...)
	}

	return rules
}

// ListPlugins returns information about all loaded plugins
func (pm *PluginManager) ListPlugins() []gomdlintPlugin.PluginInfo {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	var infos []gomdlintPlugin.PluginInfo
	for _, p := range pm.plugins {
		info := gomdlintPlugin.PluginInfo{
			Name:        p.Name(),
			Version:     p.Version(),
			Description: p.Description(),
			Author:      p.Author(),
			RuleCount:   len(p.Rules()),
		}
		infos = append(infos, info)
	}

	return infos
}

// GetPluginInfo returns detailed information about a specific plugin
func (pm *PluginManager) GetPluginInfo(name string) (*gomdlintPlugin.PluginInfo, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
	}

	info := &gomdlintPlugin.PluginInfo{
		Name:        plugin.Name(),
		Version:     plugin.Version(),
		Description: plugin.Description(),
		Author:      plugin.Author(),
		RuleCount:   len(plugin.Rules()),
	}

	return info, nil
}

// HealthCheckAll performs health checks on all loaded plugins
func (pm *PluginManager) HealthCheckAll(ctx context.Context) map[string]error {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	results := make(map[string]error)
	for name, plugin := range pm.plugins {
		if err := plugin.HealthCheck(ctx); err != nil {
			results[name] = err
		}
	}

	return results
}

// GetPluginStatus returns the current status of a plugin
func (pm *PluginManager) GetPluginStatus(name string) gomdlintPlugin.PluginStatus {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	if status, exists := pm.status[name]; exists {
		return status
	}

	return gomdlintPlugin.PluginStatus{
		Name:   name,
		Loaded: false,
	}
}

// Configure updates the plugin manager configuration
func (pm *PluginManager) Configure(config gomdlintPlugin.PluginConfig) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	pm.config = config
}

// GetConfig returns the current plugin manager configuration
func (pm *PluginManager) GetConfig() gomdlintPlugin.PluginConfig {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	return pm.config
}