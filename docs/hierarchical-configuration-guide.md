# Hierarchical Configuration Guide

This document provides a comprehensive guide to gomdlint's hierarchical configuration system, which replaces the previous "first-found-wins" approach with intelligent configuration merging.

## Overview

gomdlint now supports **true hierarchical configuration merging** that combines settings from multiple sources, allowing for flexible, layered configuration management while maintaining backward compatibility.

## Configuration Hierarchy

Configurations are merged in this priority order:

1. **Built-in Defaults** - Base configuration
2. **System XDG** (`/etc/xdg/gomdlint/`) - Organization-wide settings  
3. **User XDG** (`~/.config/gomdlint/`) - Personal preferences
4. **Project Directory** (current working directory) - Team/project settings
5. **Explicit Config** (`--config` flag) - Overrides all hierarchy

## Key Features

### ✅ Hierarchical Merging
- Multiple configuration files are automatically discovered and merged
- Higher priority sources override lower priority ones
- Deep merging for nested objects (like theme settings)
- Additive merging for compatible settings

### ✅ XDG Base Directory Support
- Follows Unix/Linux standards for configuration file organization
- System-wide and user-specific configuration support
- Respects `XDG_CONFIG_HOME` and `XDG_CONFIG_DIRS` environment variables

### ✅ Backward Compatibility
- All existing `.markdownlint.json` files continue to work unchanged
- Legacy project configs take precedence over XDG configs
- No breaking changes to existing workflows

### ✅ Intelligent Deep Merging
- **Nested objects**: Merged recursively (theme settings, rule configurations)
- **Primitive values**: Higher priority sources override lower priority
- **Arrays**: Completely replaced (no array merging)
- **New keys**: Added from any source in the hierarchy

## Practical Examples

### Scenario 1: Organization + Personal + Project

**System Admin** sets organization defaults in `/etc/xdg/gomdlint/config.json`:
```json
{
  "default": true,
  "MD013": { "line_length": 80 },
  "theme": { "theme": "minimal" }
}
```

**Developer** adds personal preferences in `~/.config/gomdlint/config.json`:
```json
{
  "MD013": { "line_length": 100 },
  "theme": { "theme": "default", "suppress_emojis": false }
}
```

**Team** defines project standards in `.markdownlint.json`:
```json
{
  "MD013": { "line_length": 120 },
  "MD041": false,
  "theme": { "suppress_emojis": true }
}
```

**Final Merged Configuration**:
```json
{
  "default": true,
  "MD013": { "line_length": 120 },    // Project wins
  "MD041": false,                      // From project
  "theme": {
    "theme": "default",                // User preference wins
    "suppress_emojis": true            // Project requirement wins
  }
}
```

### Scenario 2: Development vs CI/CD

**Development** (uses hierarchical merging):
```bash
cd /project
gomdlint lint *.md
# Uses: system + user + project configs merged
```

**CI/CD** (explicit config for predictability):
```bash
gomdlint lint --config .ci/markdownlint-strict.json *.md
# Uses: only the specified config file
```

## Command Enhancements

### `gomdlint config which`

Now shows hierarchical information:

**Multiple Sources**:
```
Hierarchical configuration active (3 sources merged):

1. /etc/xdg/gomdlint/config.json
   Type: XDG system config (organization-wide)
   
2. /home/user/.config/gomdlint/config.json
   Type: XDG user config (personal preferences)
   
3. /project/.markdownlint.json
   Type: Project directory (team/project settings)

Configuration merge order: system < user < project
Higher priority sources override settings from lower priority sources.
```

### `gomdlint config show`

Displays merged configuration with source attribution:

```
# Hierarchical configuration merged from multiple sources:
#   1. /etc/xdg/gomdlint/config.json (system)
#   2. /home/user/.config/gomdlint/config.json (user)  
#   3. /project/.markdownlint.json (project)
#
# Higher-numbered sources override lower-numbered sources

{
  "MD013": { "line_length": 120 },
  "theme": {
    "theme": "default",
    "suppress_emojis": true,
    "custom_symbols": {
      "info": "[INFO]",      // from system
      "success": "[OK]",     // from user
      "error": "[ERR]"       // from project
    }
  }
}
```

### `gomdlint config validate`

Validates the complete merged hierarchy:

```
Hierarchical configuration is valid (3 sources merged)
  - /etc/xdg/gomdlint/config.json (system)
  - /home/user/.config/gomdlint/config.json (user)
  - /project/.markdownlint.json (project)
Found 8 configuration entries
```

## Implementation Details

### Deep Merge Algorithm
The hierarchical merging uses a sophisticated deep merge algorithm that:

- Recursively merges nested objects
- Preserves all compatible settings from all sources
- Handles type conflicts by using higher priority values
- Provides detailed source tracking for debugging

### Performance Optimization
- Only reads configuration files that exist
- Caches merged configurations within a single command execution
- Efficient search order (stops at first file found per directory)
- Minimal memory overhead for configuration tracking

### Error Handling
- Gracefully handles missing or malformed configuration files
- Provides clear error messages with source file information
- Continues processing if individual sources fail to load
- Validates merged result after successful merging

## Migration Benefits

### Before (First-Found-Wins)
```
Found: .markdownlint.json
Result: Only project config used
Missing: User preferences, system defaults
```

### After (Hierarchical Merging)  
```
Found: 
  - /etc/xdg/gomdlint/config.json (system)
  - ~/.config/gomdlint/config.json (user)
  - .markdownlint.json (project)
Result: Intelligent merge of all sources
Benefit: System defaults + user preferences + project requirements
```

## Best Practices

### For System Administrators
- Set organization-wide defaults in `/etc/xdg/gomdlint/config.json`
- Keep system configs minimal and foundational
- Document expected user/project overrides

### For Developers
- Use `~/.config/gomdlint/config.json` for personal preferences
- Don't override team-critical settings in personal configs
- Leverage hierarchical merging for development flexibility

### For Teams
- Define project standards in `.markdownlint.json` 
- Version control project configs
- Use `gomdlint config show` to verify merged results
- Test configs with `gomdlint config validate`

### For CI/CD
- Use explicit `--config` for predictable behavior
- Don't rely on hierarchical merging in automated environments
- Validate configuration files in build pipelines

## Technical Architecture

The hierarchical configuration system is built on several key components:

### 1. **Deep Merge Utilities** (`internal/shared/utils/merge.go`)
- Generic configuration merging with type safety
- Recursive object merging with priority handling
- Source tracking and metadata preservation

### 2. **XDG Path Resolution** (`internal/shared/utils/xdg.go`) 
- Standards-compliant XDG Base Directory implementation
- Environment variable support and fallback handling
- Cross-platform compatibility with sensible defaults

### 3. **Configuration Loading** (`internal/interfaces/cli/commands/config.go`)
- Hierarchical discovery and loading of all configuration sources
- Intelligent merging with priority-based override resolution
- Enhanced command interfaces with hierarchy information

### 4. **Source Tracking** (`ConfigurationSource` struct)
- Detailed metadata about each configuration source
- Merge order tracking and source attribution
- Support for both hierarchical and single-source modes

This implementation provides a robust, flexible, and backward-compatible configuration system that scales from individual developers to large organizations while maintaining the simplicity that makes gomdlint easy to use.
