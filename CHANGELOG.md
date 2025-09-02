# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Additional planned features and improvements

### Changed
- Future changes and modifications

### Deprecated
- Features to be removed in future versions

### Removed
- Removed features

### Fixed
- Bug fixes and corrections

### Security
- Security improvements and vulnerability fixes

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
