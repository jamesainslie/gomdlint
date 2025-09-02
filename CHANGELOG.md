# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Future planned features and improvements

### Changed
- Future changes and modifications

## [0.2.1] - 2025-01-09

### Fixed
- Created missing homebrew-tap repository for Homebrew package distribution
- Created missing scoop-bucket repository for Scoop package distribution
- Updated Dockerfile to remove corporate-specific configurations for public use
- Updated Docker image namespaces to use jamesainslie account

### Changed
- Docker builds temporarily disabled due to network connectivity issues

## [0.2.0] - 2025-01-09

### Added
- **Plugin System**: Complete plugin architecture for custom rules and functionality
  - Plugin interfaces with lifecycle management (load, unload, health checks)
  - PluginManager with thread-safe plugin registry and status tracking
  - Support for .so file loading using Go's native plugin system
  - Plugin CLI commands: install, uninstall, list, info, build, health
- **Async Rule Execution**: Concurrent rule processing with performance optimization
  - AsyncRule and AsyncRuleEngine for parallel rule execution
  - Channel-based result collection with timeout support
  - Worker pool pattern for controlled concurrency
  - Performance metadata tracking and timing measurements
- **Multiple Parser Support**: Extensible parser architecture
  - Parser interface with registry pattern for different markdown parsers
  - Built-in parsers: CommonMark, Goldmark (stub), Blackfriday (stub), None
  - Parser-specific configuration and capabilities
  - Front matter extraction and token generation abstraction
- **Enhanced Configuration System**: Advanced configuration management
  - Configuration extension support with 'extends' field and inheritance
  - ConfigResolver with circular dependency detection
  - JSON config loader with comprehensive validation
  - Plugin/parser/profile configuration structures
- **Style Management**: Predefined configuration templates
  - Built-in styles: relaxed, strict, minimal, all
  - Style CLI commands: list, show, apply, create, validate
  - Style registry with validation and export functionality
- **Helper Library**: Public API for rule development (pkg/gomdlint/helpers)
  - Token manipulation and analysis utilities
  - Text analysis helpers (whitespace, formatting, structure detection)
  - Fix generation utilities for automated correction
  - Regular expressions for common markdown patterns

### Changed
- CLI architecture streamlined for better performance and maintainability
- Application description updated to emphasize plugin extensibility
- Performance testing approach updated from benchmarking to standard Go benchmarks
- Documentation restructured around plugin system and style management

### Removed
- **TUI (Terminal User Interface)**: Interactive terminal interface removed
  - Removed bubbletea dependency and associated UI components
  - Simplified CLI focus on automation and scripting
- **Benchmark Tool**: Built-in benchmarking commands removed
  - Removed benchmark command and associated test generation
  - Simplified to standard Go benchmark testing approach

### Fixed
- Configuration type system unified with ExtendedRuleConfiguration
- Parser token generation corrected for proper type compatibility
- Dependency cleanup and module organization

## [0.1.0] - 2024-12-19

### Added
- Initial release of gomdlint - A high-performance Go markdown linter
- **59 Built-in Rules**: Full implementation of all markdownlint rules (MD001-MD059)
- **Performance First**: Up to 10x faster than Node.js alternatives through Go's concurrency
- **Plugin System**: Extensible architecture for custom rules and functionality
- **Multiple Output Formats**: Support for default, JSON, JUnit, and Checkstyle formats
- **Auto-fixing**: Automatically fix violations where possible
- **Library Mode**: Use as a Go library in applications
- **Configuration Management**: Support for JSON, YAML, and TOML configuration files
- **Extensible Architecture**: Plugin system for custom rules
- **CLI Commands**:
  - `lint` - Lint markdown files with detailed violation reporting
  - `check` - CI-friendly checking with exit codes
  - `fix` - Automatic violation fixing where possible
  - `config` - Configuration management and validation
  - `theme` - Theme management for output customization
  - `rules` - Rule information and management
  - `plugin` - Plugin management for extensibility
  - `style` - Style configuration management
  - `version` - Version information display
- **Build System**: Comprehensive Makefile with cross-platform builds
- **Testing**: Unit tests, integration tests, and quality assurance tools
- **Documentation**: Complete README with usage examples and API documentation
- **Compatibility**: Full markdownlint compatibility with identical rule behavior

### Technical Details
- **Go Version**: Requires Go 1.24+
- **Dependencies**: Built with Cobra CLI framework and modular architecture
- **Architecture**: Clean architecture with domain-driven design
- **Performance**: Concurrent processing with intelligent caching
- **Platforms**: Linux, macOS, and Windows support (amd64 and arm64)

[Unreleased]: https://github.com/gomdlint/gomdlint/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/gomdlint/gomdlint/releases/tag/v0.1.0
