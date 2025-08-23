# Versioning Guide

This document outlines the versioning strategy for gomdlint, following Semantic Versioning (SemVer) principles.

## Semantic Versioning

gomdlint follows [Semantic Versioning 2.0.0](https://semver.org/) for all releases:

```
MAJOR.MINOR.PATCH
```

### Version Components

- **MAJOR** (X.0.0): Incompatible API changes
- **MINOR** (0.X.0): Backwards-compatible functionality additions
- **PATCH** (0.0.X): Backwards-compatible bug fixes

### Pre-release Versions

Pre-release versions follow the pattern:

```
MAJOR.MINOR.PATCH-<prerelease>.<number>
```

Examples:
- `1.2.0-alpha.1` - First alpha release
- `1.2.0-beta.3` - Third beta release  
- `1.2.0-rc.1` - First release candidate

## Release Types

### Major Version (X.0.0)

Increment when making **breaking changes** such as:

- Removing or changing existing CLI commands
- Changing CLI argument names or behavior
- Removing configuration options
- Changing API signatures in the Go library
- Changing output formats in incompatible ways
- Dropping support for older Go versions

**Example**: `1.0.0` → `2.0.0`

### Minor Version (0.X.0)

Increment when adding **new features** in a backwards-compatible manner:

- Adding new CLI commands or subcommands
- Adding new configuration options
- Adding new output formats
- Adding new rules
- Adding new API methods to the Go library
- Performance improvements
- New TUI features

**Example**: `1.1.0` → `1.2.0`

### Patch Version (0.0.X)

Increment when making **backwards-compatible bug fixes**:

- Fixing rule implementations
- Fixing configuration parsing
- Security fixes
- Performance bug fixes
- Documentation corrections
- TUI bug fixes

**Example**: `1.1.0` → `1.1.1`

## Version Management

### Git Tags

All versions are tracked using annotated git tags with the `v` prefix:

```bash
# Create a new version tag
git tag -a v1.2.0 -m "Release v1.2.0: Add new rules and TUI improvements"

# List all version tags
git tag -l "v*" --sort=-version:refname

# Show tag information
git show v1.2.0
```

### VERSION File

The `VERSION` file in the project root contains the current version number without the `v` prefix:

```
1.2.0
```

This file should be updated with each release.

### Build System Integration

The Makefile automatically detects the current version from git tags:

```makefile
VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
```

Version information is embedded in the binary during build:

```makefile
LDFLAGS = -ldflags="-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"
```

## Release Process

### 1. Prepare the Release

1. **Update VERSION file** with the new version number
2. **Update CHANGELOG.md** with release notes
3. **Run quality checks**: `make check`
4. **Test thoroughly** including version commands

### 2. Create the Release

```bash
# Commit version changes
git add VERSION CHANGELOG.md
git commit -m "Prepare release v1.2.0"

# Create annotated tag
git tag -a v1.2.0 -m "Release v1.2.0: Brief description of changes"

# Push commits and tags
git push origin main
git push origin v1.2.0
```

### 3. Build Release Artifacts

```bash
# Build all platform binaries
make release

# Verify build succeeded
ls -la releases/
```

### 4. Post-Release

1. **Create GitHub Release** with release notes
2. **Update documentation** if needed
3. **Announce the release** in relevant channels

## Version Commands

### Check Current Version

```bash
# From source
make version

# From built binary  
./bin/gomdlint version
./bin/gomdlint --version

# From installed binary
gomdlint version
gomdlint --version
```

### Version Information

The version commands provide:
- **Version**: Semantic version number
- **Commit**: Git commit hash
- **Built**: Build timestamp
- **Go**: Go version used for build

Example output:
```
gomdlint version v1.2.0
  commit: abc1234
  built: 2024-12-19T10:30:45Z
  go: go1.24+
```

## Backward Compatibility

### API Compatibility

The Go library API maintains backward compatibility within major versions:

- **Public functions** won't be removed or change signatures
- **Public types** won't be removed or change field types
- **Configuration formats** remain compatible
- **Output formats** remain consistent

### CLI Compatibility

Command-line interface maintains backward compatibility within major versions:

- **Existing commands** continue to work
- **Existing flags** continue to work
- **Configuration files** remain compatible
- **Exit codes** remain consistent

### Breaking Changes

When breaking changes are necessary:

1. **Deprecate** the old functionality first
2. **Document** the migration path
3. **Provide warnings** for deprecated usage
4. **Remove** deprecated functionality in the next major version

## Version Ranges

### Dependency Management

When depending on gomdlint as a library:

```go
// go.mod
require github.com/gomdlint/gomdlint v1.2.0

// For latest compatible version within v1.x
require github.com/gomdlint/gomdlint ^1.2.0
```

### CI/CD Integration

For CI/CD pipelines, pin to specific versions:

```yaml
# GitHub Actions
- name: Install gomdlint
  run: |
    curl -sSL https://github.com/gomdlint/gomdlint/releases/download/v1.2.0/gomdlint-linux-amd64.tar.gz | tar xz
```

## Development Versions

### Between Releases

Development builds use the version format:
```
v1.2.0-dev-abc1234
```

Where:
- `v1.2.0` is the last released version
- `dev` indicates development build
- `abc1234` is the current commit hash

### Testing Pre-releases

Pre-release versions for testing:

```bash
# Alpha releases for early testing
git tag -a v1.3.0-alpha.1 -m "Alpha release for testing"

# Beta releases for broader testing  
git tag -a v1.3.0-beta.1 -m "Beta release for testing"

# Release candidates for final testing
git tag -a v1.3.0-rc.1 -m "Release candidate"
```

## Changelog Maintenance

The `CHANGELOG.md` follows [Keep a Changelog](https://keepachangelog.com/) format:

```markdown
## [1.2.0] - 2024-12-19

### Added
- New CLI command for rule management

### Changed  
- Improved performance for large files

### Fixed
- Fixed configuration parsing issue

### Deprecated
- Old configuration format (will be removed in v2.0.0)
```

## References

- [Semantic Versioning Specification](https://semver.org/)
- [Keep a Changelog](https://keepachangelog.com/)
- [Git Tagging Documentation](https://git-scm.com/book/en/v2/Git-Basics-Tagging)
