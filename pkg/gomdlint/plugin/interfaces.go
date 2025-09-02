package plugin

import (
	"context"
	"net/url"

	"github.com/gomdlint/gomdlint/internal/domain/entity"
	"github.com/gomdlint/gomdlint/internal/domain/value"
	"github.com/gomdlint/gomdlint/internal/shared/functional"
)

// Plugin represents a loadable plugin that can provide custom rules
type Plugin interface {
	// Metadata
	Name() string
	Version() string
	Description() string
	Author() string

	// Lifecycle
	Initialize(ctx context.Context, config PluginConfig) error
	Shutdown(ctx context.Context) error

	// Rule provision
	Rules() []CustomRule

	// Health check
	HealthCheck(ctx context.Context) error
}

// CustomRule defines the interface for plugin-provided rules
type CustomRule interface {
	// Metadata (same as built-in rules)
	Names() []string
	Description() string
	Tags() []string
	Information() *url.URL
	Parser() ParserType

	// Configuration
	DefaultConfig() map[string]interface{}
	ValidateConfig(config map[string]interface{}) error

	// Execution
	Execute(ctx context.Context, params entity.RuleParams) functional.Result[[]value.Violation]

	// Async support
	IsAsync() bool
	ExecuteAsync(ctx context.Context, params entity.RuleParams) <-chan RuleResult
}

// ParserType defines supported parsers
type ParserType string

const (
	ParserCommonMark  ParserType = "commonmark"
	ParserGoldmark    ParserType = "goldmark"
	ParserBlackfriday ParserType = "blackfriday"
	ParserNone        ParserType = "none"
)

// RuleResult for async execution
type RuleResult struct {
	Violations []value.Violation
	Error      error
	Metadata   map[string]interface{}
}

// PluginConfig for plugin initialization
type PluginConfig struct {
	DataDir     string
	ConfigDir   string
	CacheDir    string
	LogLevel    string
	Environment map[string]string
}

// PluginInfo contains metadata about a plugin
type PluginInfo struct {
	Name        string
	Version     string
	Description string
	Author      string
	Homepage    string
	License     string
	RuleCount   int
}

// AsyncCapable indicates if a rule supports async execution
type AsyncCapable interface {
	IsAsync() bool
	ExecuteAsync(ctx context.Context, params entity.RuleParams) <-chan RuleResult
}

// Configurable indicates if a rule has configuration options
type Configurable interface {
	DefaultConfig() map[string]interface{}
	ValidateConfig(config map[string]interface{}) error
	ApplyConfig(config map[string]interface{}) error
}

// PluginManager interface for managing plugins
type PluginManager interface {
	// Plugin lifecycle
	LoadPlugin(ctx context.Context, path string) error
	UnloadPlugin(ctx context.Context, name string) error
	ReloadPlugin(ctx context.Context, name string) error

	// Plugin discovery
	LoadPluginsFromDirectory(ctx context.Context, dir string) error
	ScanForPlugins(ctx context.Context, paths []string) ([]string, error)

	// Plugin access
	GetPlugin(name string) (Plugin, error)
	GetAllPlugins() map[string]Plugin
	GetAllCustomRules() []CustomRule

	// Plugin information
	ListPlugins() []PluginInfo
	GetPluginInfo(name string) (*PluginInfo, error)

	// Health and status
	HealthCheckAll(ctx context.Context) map[string]error
	GetPluginStatus(name string) PluginStatus
}

// PluginStatus represents the current state of a plugin
type PluginStatus struct {
	Name        string
	Loaded      bool
	Initialized bool
	Error       error
	LoadTime    int64
	RuleCount   int
}

// PluginBuilder helps build plugins with a fluent interface
type PluginBuilder interface {
	WithName(name string) PluginBuilder
	WithVersion(version string) PluginBuilder
	WithDescription(description string) PluginBuilder
	WithAuthor(author string) PluginBuilder
	AddRule(rule CustomRule) PluginBuilder
	Build() Plugin
}