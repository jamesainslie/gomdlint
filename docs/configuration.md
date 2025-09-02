# Configuration Management

gomdlint provides flexible **hierarchical configuration management** following the XDG Base Directory Specification while maintaining backward compatibility with legacy locations. It supports multiple configuration file formats and **automatically merges configurations from different sources** for maximum flexibility.

## XDG Base Directory Support

gomdlint follows the [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html) for better organization of configuration files:

### XDG Directories

- **User Config**: `$XDG_CONFIG_HOME/gomdlint/` (default: `~/.config/gomdlint/`)
- **System Config**: `$XDG_CONFIG_DIRS/gomdlint/` (default: `/etc/xdg/gomdlint/`)

### Benefits of XDG

- **Organization**: Keeps configuration files organized in standard directories
- **Multi-user**: Supports both user-specific and system-wide configurations  
- **Standards Compliance**: Follows Unix/Linux desktop standards
- **Clean Home**: Avoids cluttering the home directory with dotfiles

## Configuration Files

### Supported Formats

gomdlint supports JSON configuration files (YAML support planned for future releases):

**XDG Locations (Recommended):**
- `$XDG_CONFIG_HOME/gomdlint/config.json`
- `$XDG_CONFIG_HOME/gomdlint/config.yaml` (planned)
- `$XDG_CONFIG_HOME/gomdlint/config.yml` (planned)
- `$XDG_CONFIG_HOME/gomdlint/.gomdlint.json`

**Legacy Locations (Backward Compatibility):**
- `.markdownlint.json`
- `.markdownlint.yaml` (planned)
- `.markdownlint.yml` (planned)
- `markdownlint.json`
- `markdownlint.yaml` (planned)
- `markdownlint.yml` (planned)

## Hierarchical Configuration

gomdlint implements a **hierarchical configuration system** that merges multiple configuration sources to provide maximum flexibility while maintaining simplicity.

### Configuration Hierarchy

Configurations are merged in this priority order (later sources override earlier ones):

1. **System Defaults** - Built-in configuration
2. **XDG System Config** (`/etc/xdg/gomdlint/`) - Organization-wide settings
3. **XDG User Config** (`~/.config/gomdlint/`) - Personal preferences
4. **Project Config** (current directory) - Team/project-specific settings
5. **Explicit Config** (`--config` flag) - Overrides all hierarchy

### Deep Merging Behavior

- **Nested objects** are merged recursively (e.g., theme settings)
- **Arrays** are replaced entirely (no merging of array elements)
- **Primitive values** are overridden by higher-priority sources
- **New keys** are added from any source in the hierarchy

### Search Order and Filenames

When no specific config file is provided, gomdlint searches for files in this order:

**Current Directory** (project config - highest priority):
- `config.json`, `config.yaml`, `config.yml`
- `.gomdlint.json`, `.gomdlint.yaml`, `.gomdlint.yml`
- `.markdownlint.json`, `.markdownlint.yaml`, `.markdownlint.yml`
- `markdownlint.json`, `markdownlint.yaml`, `markdownlint.yml`

**XDG User Config** (`~/.config/gomdlint/`) - medium priority:
- Same filename patterns as above

**XDG System Config** (`/etc/xdg/gomdlint/`) - low priority:
- Same filename patterns as above

If no configuration files are found, gomdlint uses built-in defaults.

## Configuration Commands

### Show Active Configuration

Display the effective configuration that would be used for linting:

```bash
gomdlint config show
```

This command shows:
- Which configuration file is being used (with XDG/legacy indication)
- The complete configuration content
- Guidance for creating configuration files

Example outputs:

**Hierarchical Configuration (Multiple Sources):**
```
# Hierarchical configuration merged from multiple sources:
#   1. /home/user/.config/gomdlint/config.json (user)
#   2. /path/to/project/.markdownlint.json (project)
#
# Higher-numbered sources override lower-numbered sources

{
  "default": true,
  "MD013": {
    "line_length": 120
  },
  "MD033": false,
  "theme": {
    "theme": "default",
    "suppress_emojis": true,
    "custom_symbols": {
      "success": "[USER-OK]",
      "error": "[PROJECT-ERR]"
    }
  }
}
```

**Single Source Configuration:**
```
# Configuration loaded from: /home/user/.config/gomdlint/config.json (user)

{
  "default": true,
  "MD013": {
    "line_length": 100
  },
  "theme": {
    "theme": "default",
    "suppress_emojis": false
  }
}
```

**No Configuration:**
```
# No configuration files found - using built-in defaults
# Use 'gomdlint config which' to see all search paths
# Use 'gomdlint config init' to create a configuration file

{
  "default": true
}
```

### Find Configuration File Location

Show which configuration files are being loaded:

```bash
# Simple tree view (default)
gomdlint config which

# Detailed information with search paths
gomdlint config which --verbose
```

Example outputs:

**Default Output - Hierarchical Configuration:**
```
 Configuration hierarchy (2 files merged):

├─  ~/.config/gomdlint/config.json user [1]
└─  /project/.markdownlint.json project [2]
```

**Default Output - Single Configuration:**
```
 Configuration: ~/.config/gomdlint/config.json user
```

**Default Output - No Configuration:**
```
Error: no configuration files found - using built-in defaults
```

**Verbose Output - Hierarchical Configuration:**
```bash
gomdlint config which --verbose
```
```
Hierarchical configuration active (2 sources merged):

1. /home/user/.config/gomdlint/config.json
   Type: XDG user config (personal preferences)
   Size: 180 bytes
   Modified: 2024-01-15 10:30:45

2. /path/to/project/.markdownlint.json
   Type: Project directory (team/project settings)
   Size: 120 bytes
   Modified: 2024-01-15 14:20:30

Configuration merge order: system < user < project
Higher priority sources override settings from lower priority sources.

Search paths and detailed information:
Search order:

1. /home/user/project (current directory - legacy)
   - config.json (not found)
   - config.yaml (not found)
   - .gomdlint.json (not found)
   - .markdownlint.json 
   [... additional filenames ...]

2. /home/user/.config/gomdlint (XDG user config)
   - config.json 
   - config.yaml (not found)
   [... additional filenames ...]

3. /etc/xdg/gomdlint (XDG system config)
   - config.json (not found)
   [... additional filenames ...]
```

**Verbose Output - Single Configuration:**
```bash
gomdlint config which --verbose  
```
```
Configuration file: /home/user/.config/gomdlint/config.json
Type: XDG user config (personal preferences)
File size: 180 bytes
Last modified: 2024-01-15 10:30:45

Search paths and detailed information:
[... complete search path information ...]
```

### Validate Configuration

Check if your configuration files are valid:

```bash
# Validate the same hierarchy used for linting (recommended)
gomdlint config validate

# Validate a specific configuration file
gomdlint config validate path/to/config.json
```

By default, validates the same configuration hierarchy that would be used for linting, ensuring consistency across commands.

This validates:
- JSON syntax across all configuration sources
- Theme configuration (if present) 
- Rule configurations
- Custom symbol definitions
- Hierarchical merge compatibility

Example outputs:

**Single Configuration:**
```
Theme configuration is valid
Configuration file /home/user/.config/gomdlint/config.json is valid
Found 3 configuration entries
```

**Hierarchical Configuration:**
```
Theme configuration is valid
Hierarchical configuration is valid (2 sources merged)
  - /home/user/.config/gomdlint/config.json (user)
  - /project/.markdownlint.json (project)
Found 4 configuration entries
```

**No Configuration:**
```
No configuration found - will use defaults
```

### Initialize Configuration

Create a new configuration file with defaults:

```bash
# Create in XDG directory (recommended)
gomdlint config init

# Create in current directory (legacy)
gomdlint config init --legacy
```

**XDG Creation (Default):**
```
Configuration file created: /home/user/.config/gomdlint/config.json
Created in XDG config directory (recommended)
Use --legacy flag to create in current directory instead
```

**Legacy Creation:**
```
Configuration file created: /home/user/project/.markdownlint.json
Created in current directory (legacy location)
Consider migrating to XDG config directory with 'gomdlint config init'
```

The created configuration includes:
- Default rule settings
- Default theme configuration
- Common customizations

## Configuration During Linting

### Verbose Mode

Use the `-v` flag to see which configuration file is loaded:

```bash
gomdlint lint -v README.md
```

Output includes:
```
Using configuration from: .markdownlint.json
Found 2 files to lint
No violations found in 2 files (0.12s)
```

### Override Configuration File

Specify a different configuration file:

```bash
gomdlint lint --config custom-config.json *.md
```

### Disable Configuration Loading

Skip configuration loading entirely:

```bash
gomdlint lint --no-config *.md
```

This uses only built-in defaults, ignoring any configuration files.

## Configuration Structure

### Basic Configuration

```json
{
  "default": true,
  "MD013": {
    "line_length": 120
  },
  "MD033": false,
  "MD041": false
}
```

### Configuration with Theming

```json
{
  "default": true,
  "MD013": {
    "line_length": 120
  },
  "theme": {
    "theme": "minimal",
    "suppress_emojis": false,
    "custom_symbols": {
      "success": "",
      "error": ""
    }
  }
}
```

## Environment-Specific Configuration

### Project Setup

Create different configuration files for different environments:

```
project/
├── .markdownlint.json          # Default config
├── config/
│   ├── ci.json                 # CI/CD config
│   ├── dev.json                # Development config
│   └── strict.json             # Strict validation config
└── docs/
    └── README.md
```

### Usage Examples

```bash
# Use default config
gomdlint lint docs/

# Use CI config
gomdlint lint --config config/ci.json docs/

# Use strict config for documentation
gomdlint lint --config config/strict.json docs/
```

### CI/CD Configuration Example

`config/ci.json`:
```json
{
  "default": true,
  "theme": {
    "theme": "ascii",
    "suppress_emojis": true
  }
}
```

## XDG Environment Variables

gomdlint respects XDG Base Directory environment variables:

- `XDG_CONFIG_HOME` - User configuration directory (default: `~/.config`)
- `XDG_CONFIG_DIRS` - System configuration directories (default: `/etc/xdg`)

Example custom setup:
```bash
export XDG_CONFIG_HOME="$HOME/my-configs"
gomdlint config which  # Now looks in ~/my-configs/gomdlint/
```

## Hierarchical Configuration Examples

### Example 1: Complete Hierarchy

This example demonstrates how configurations merge across all hierarchy levels.

**System Config** (`/etc/xdg/gomdlint/config.json`):
```json
{
  "default": true,
  "MD013": { "line_length": 80 },
  "MD033": false,
  "theme": {
    "theme": "minimal",
    "suppress_emojis": false,
    "custom_symbols": { "info": "[INFO]" }
  }
}
```

**User Config** (`~/.config/gomdlint/config.json`):
```json
{
  "MD013": { "line_length": 100 },
  "theme": {
    "theme": "default",
    "custom_symbols": { "success": "[OK]", "warning": "[WARN]" }
  }
}
```

**Project Config** (`.markdownlint.json`):
```json
{
  "MD013": { "line_length": 120 },
  "MD041": false,
  "theme": {
    "suppress_emojis": true,
    "custom_symbols": { "error": "[ERR]" }
  }
}
```

**Merged Result**:
```json
{
  "default": true,
  "MD013": { "line_length": 120 },      // Project override
  "MD033": false,                        // From system
  "MD041": false,                        // From project
  "theme": {
    "theme": "default",                  // User override
    "suppress_emojis": true,             // Project override
    "custom_symbols": {                  // Deep merged from all sources
      "info": "[INFO]",                  // From system
      "success": "[OK]",                 // From user
      "warning": "[WARN]",               // From user
      "error": "[ERR]"                   // From project
    }
  }
}
```

### Example 2: User + Project Configuration

Most common scenario with personal preferences and project settings.

**User Config** (`~/.config/gomdlint/config.json`):
```json
{
  "default": true,
  "theme": {
    "theme": "default",
    "suppress_emojis": false
  },
  "MD013": { "line_length": 100 }
}
```

**Project Config** (`.markdownlint.json`):
```json
{
  "MD013": { "line_length": 120 },
  "MD033": false,
  "theme": { "suppress_emojis": true }
}
```

**Result**: Project line length (120) and emoji suppression (true) override user settings, while user theme ("default") is preserved.

### Example 3: Explicit Config Override

When using `--config`, hierarchical merging is bypassed:

```bash
# This ignores all hierarchy and uses only the specified file
gomdlint lint --config /path/to/special-config.json
```

## Migration Guide

### From Legacy to XDG

**Step 1: Check current configuration**
```bash
gomdlint config which
```

**Step 2: Create XDG configuration**
```bash
gomdlint config init
```

**Step 3: Copy existing settings**
```bash
# If you have custom settings in .markdownlint.json
cp .markdownlint.json ~/.config/gomdlint/config.json
```

**Step 4: Verify new configuration**
```bash
gomdlint config show
gomdlint config which
```

**Step 5: Remove legacy file (optional)**
```bash
rm .markdownlint.json
```

### System-wide Configuration

For system administrators wanting to set default configurations:

```bash
# Create system-wide config
sudo mkdir -p /etc/xdg/gomdlint
sudo cp your-config.json /etc/xdg/gomdlint/config.json

# Users can override with their own configs in ~/.config/gomdlint/
```

## Best Practices

### 1. Version Control

- **Include** project-specific configuration files in version control
- Use XDG directories for personal/user-specific overrides
- Document configuration choices in your README

### 2. Hierarchical Configuration Strategy

- **System admins**: Use `/etc/xdg/gomdlint/` for organization defaults
- **Users**: Use `~/.config/gomdlint/` for personal preferences
- **Teams**: Use project directory configs for team consistency
- **CI/CD**: Use explicit `--config` to bypass hierarchy when needed

### 3. Team Consistency

- Establish team standards for configuration
- Use `config validate` in CI/CD pipelines
- Share configuration examples in documentation
- Consider hybrid approach: legacy for project, XDG for user preferences

### 4. Environment Optimization

- **Development**: Use hierarchical configs for flexibility
- **CI/CD**: Use explicit `--config` for predictable behavior
- **Teams**: Combine project configs with personal XDG overrides
- **Organizations**: Set system defaults, allow user customization

### 5. Configuration Testing

Test your configuration changes:

```bash
# Validate configuration
gomdlint config validate

# Show what will be used
gomdlint config show

# Quick check of loaded configs
gomdlint config which

# Detailed debugging with search paths
gomdlint config which --verbose

# Test with dry run
gomdlint lint --dry-run *.md
```

### 6. Debugging Configuration Issues

Use these commands to troubleshoot configuration problems:

```bash
# See which configs are loaded (simple tree)
gomdlint config which

# Get full search path details for debugging
gomdlint config which --verbose

# Check if configuration is valid
gomdlint config validate

# See final merged configuration
gomdlint config show
```

Common debugging scenarios:
- **Config not loading**: Use `--verbose` to see search paths
- **Wrong config priority**: Check hierarchy order in output  
- **Merge conflicts**: Compare individual files with merged result
- **Validation errors**: Fix syntax issues before debugging hierarchy

## Troubleshooting

### Configuration Not Found

If `gomdlint config which` shows no configuration:

1. Check current directory for config files
2. Verify file names match expected patterns
3. Check file permissions
4. Use absolute path with `--config`

### Configuration Not Applied

If settings seem ignored:

1. Validate configuration syntax: `gomdlint config validate`
2. Check which file is loaded: `gomdlint config which`
3. Verify rule names and structure
4. Use `--verbose` to see loading process

### Theme Not Working

If theme settings don't apply:

1. Check theme configuration syntax
2. Verify symbol names in `custom_symbols`
3. Ensure theme name is valid (`default`, `minimal`, `ascii`)
4. Test with `config show` to see effective settings

## Migration

### From markdownlint

gomdlint uses compatible configuration format:

1. Copy existing `.markdownlint.json`
2. Add theme section if desired
3. Validate with `gomdlint config validate`
4. Test with `gomdlint config show`

### From Other Tools

1. Create new config: `gomdlint config init`
2. Migrate rule settings manually
3. Add theme preferences
4. Validate and test configuration
