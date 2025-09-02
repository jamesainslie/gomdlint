# Theming System

The gomdlint theming system allows you to control the visual appearance of CLI output, including emojis, symbols, and colors. This system provides a clean way to customize output for different environments and user preferences.

## Overview

The theming system follows clean architecture principles and uses functional programming patterns for immutable configurations. It features:

- **Theme Directory Management**: Themes are stored in `~/.config/gomdlint/themes/` 
- **CRUD Operations**: Create, edit, list, and delete themes via `gomdlint theme` commands
- **Simple Configuration**: Select themes by name in your config file
- **Built-in Themes**: Default, minimal, and ASCII themes included
- **Custom Themes**: Create and share custom theme definitions
- **Backward Compatibility**: Legacy theme configuration format still supported

## Quick Start

### 1. Install Built-in Themes

```bash
# Install the built-in themes
gomdlint theme install

# List available themes  
gomdlint theme list

# Show theme details
gomdlint theme show default
```

### 2. Simple Theme Configuration

**New Format (Recommended):**
```json
{
  "default": true,
  "theme": "minimal"
}
```

**Legacy Format (Still Supported):**
```json
{
  "default": true,
  "theme": {
    "theme": "minimal",
    "suppress_emojis": false,
    "custom_symbols": {}
  }
}

## Theme Management

### Installing Built-in Themes

First, install the built-in themes:

```bash
gomdlint theme install
```

This creates theme files in `~/.config/gomdlint/themes/`:
- `default.json` - Rich emoji theme 
- `minimal.json` - Simple ASCII symbols
- `ascii.json` - Pure text indicators

### Creating Custom Themes

```bash
# Create a new theme from template
gomdlint theme create my-theme --template default

# Create interactively
gomdlint theme create my-theme --interactive

# List all themes
gomdlint theme list

# Show theme details
gomdlint theme show my-theme

# Edit a theme (opens in $EDITOR)
gomdlint theme edit my-theme

# Delete a theme
gomdlint theme delete my-theme
```

### Theme File Format

Theme files are JSON documents stored in `~/.config/gomdlint/themes/`. Here's the structure:

```json
{
  "name": "my-custom-theme",
  "description": "My awesome custom theme",
  "author": "Your Name",
  "version": "1.0.0",
  "symbols": {
    "success": "✅",
    "error": "❌", 
    "warning": "⚠️",
    "info": "ℹ️",
    "processing": "🔄",
    "file_found": "📄",
    "file_saved": "💾",
    "performance": "⚡",
    "results": "📊",
    "winner": "🏆",
    "search": "🔍",
    "launch": "🚀"
  },
  "settings": {
    "use_colors": true
  }
}
```

### Theme Directory Structure

```
~/.config/gomdlint/
├── config.json          # Main configuration
└── themes/              # Theme directory
    ├── default.json     # Built-in default theme
    ├── minimal.json     # Built-in minimal theme  
    ├── ascii.json       # Built-in ASCII theme
    └── my-theme.json    # Your custom themes
```

## Configuration

### Simple Theme Selection (Recommended)

Simply specify the theme name in your configuration:

```json
{
  "default": true,
  "theme": "minimal"
}
```

Available built-in themes:
- `"default"` - Rich emojis and symbols
- `"minimal"` - Clean ASCII symbols  
- `"ascii"` - Pure text for automation

### Advanced Configuration (Legacy Format)

For more control, use the object format:

```json
{
  "default": true,
  "theme": {
    "theme": "default",
    "suppress_emojis": false,
    "custom_symbols": {
      "success": "✨",
      "error": "💥"
    }
  }
}

### Theme Options

#### Available Themes

1. **default** - Rich emoji theme (default)
   - Uses full emoji set for visual feedback
   - Best for interactive terminal use

2. **minimal** - Subtle symbols
   - Uses simple ASCII symbols instead of emojis
   - Good balance between visual feedback and simplicity

3. **ascii** - Pure ASCII
   - Uses text-based indicators only
   - Perfect for scripts and automation

#### Emoji Suppression

Set `suppress_emojis` to `true` to completely remove all emojis regardless of theme:

```json
{
  "theme": {
    "theme": "default",
    "suppress_emojis": true
  }
}
```

#### Custom Symbols

Override specific symbols while keeping the base theme:

```json
{
  "theme": {
    "theme": "minimal",
    "custom_symbols": {
      "success": "✓",
      "error": "✗",
      "processing": "⏳",
      "file_found": "📄"
    }
  }
}
```

### Available Symbol Types

You can customize any of these symbols:

| Symbol | Purpose | Default (emoji) | Minimal | ASCII |
|--------|---------|-----------------|---------|-------|
| `success` | Successful operations | ✅ | ✓ | [OK] |
| `error` | Error conditions | ❌ | ✗ | [ERROR] |
| `warning` | Warning messages | ⚠️ | ! | [WARN] |
| `info` | Informational messages | ℹ️ | i | [INFO] |
| `processing` | Operations in progress | 🔍 | ... | [...] |
| `file_found` | File operations | 📁 | * | [FILE] |
| `file_saved` | File saved operations | 📁 | * | [SAVED] |
| `performance` | Performance operations | 🚀 | > | [PERF] |
| `performance` | Performance metrics | 📊 | # | [PERF] |
| `winner` | Best performance | 🏆 | * | [BEST] |
| `results` | Results display | 📈 | # | [RESULTS] |
| `search` | Search operations | 🔍 | ? | [SEARCH] |
| `launch` | Starting operations | 🚀 | > | [START] |
| `bullet` | List bullets | • | • | * |
| `arrow` | Directional indicators | → | -> | => |
| `separator` | Text separators | │ | \| | \| |

## Usage Examples

### CI/CD Environment

For continuous integration environments where emojis might not display correctly:

```json
{
  "theme": {
    "theme": "ascii",
    "suppress_emojis": true
  }
}
```

### Custom Corporate Theme

Create a professional theme with custom symbols:

```json
{
  "theme": {
    "theme": "minimal",
    "custom_symbols": {
      "success": "[PASS]",
      "error": "[FAIL]",
      "warning": "[WARN]",
      "processing": "[WORK]",
      "performance": "[PERF]"
    }
  }
}
```

### Partial Emoji Suppression

Keep some visual indicators while removing others:

```json
{
  "theme": {
    "theme": "default",
    "custom_symbols": {
      "processing": "...",
      "performance": "[PERFORMANCE]"
    }
  }
}
```

## Command Line Usage

### Validate Theme Configuration

```bash
gomdlint config validate
```

This will check your theme configuration for syntax errors and validate symbol names.

### Initialize with Theme

```bash
gomdlint config init
```

Creates a `.markdownlint.json` with default theme configuration.

### View Effective Configuration

```bash
gomdlint config show
```

Displays the complete configuration including theme settings.

## Environment-Specific Themes

### Development
```json
{
  "theme": {
    "theme": "default"
  }
}
```

### Testing/CI
```json
{
  "theme": {
    "theme": "ascii",
    "suppress_emojis": true
  }
}
```

### Production Logs
```json
{
  "theme": {
    "theme": "minimal",
    "custom_symbols": {
      "success": "OK",
      "error": "ERROR",
      "warning": "WARN"
    }
  }
}
```

## Architecture

### Design Principles

The theming system follows these principles from the go-bootstrapper methodology:

1. **Functional Programming**: Immutable theme configurations
2. **Clean Architecture**: Domain logic separated from presentation
3. **Provider Pattern**: Extensible theme providers
4. **Type Safety**: Compile-time guarantees with Go generics

### Key Components

- **Theme Domain**: Core theme types and business logic
- **Theme Providers**: Built-in and custom theme implementations
- **Theme Service**: Application service coordinating theme operations
- **Themed Output**: Interface layer for CLI output formatting

### Extension Points

You can extend the theming system by:

1. **Custom Providers**: Implement the `Provider` interface
2. **New Themes**: Add themes through provider registration
3. **Symbol Extensions**: Add new symbol types to the core domain

## Troubleshooting

### Theme Not Applied

1. Check configuration file syntax with `gomdlint config validate`
2. Ensure theme name is spelled correctly
3. Verify the configuration file is in the expected location

### Invalid Symbols

1. Check that custom symbol names match available symbol types
2. Ensure symbol values are strings, not other types
3. Keep symbols reasonably short (max 10 characters)

### Colors Not Working

1. Check if your terminal supports ANSI colors
2. Try disabling colors with `--no-color` flag
3. Verify color support with `echo -e "\033[31mRed Text\033[0m"`

## Migration

### From Hardcoded Emojis

If you have scripts that parse gomdlint output:

1. Update parsers to handle different symbol sets
2. Consider using JSON output format for reliable parsing
3. Test with different themes to ensure compatibility

### Version Compatibility

- Theme configuration is supported in gomdlint v1.0+
- Older configurations without theme sections work unchanged
- Default behavior remains the same for backward compatibility

## Best Practices

1. **Environment Consistency**: Use the same theme across team environments
2. **CI/CD Optimization**: Use ASCII theme for automated environments
3. **Symbol Clarity**: Choose symbols that are clear in context
4. **Testing**: Test output appearance in target environments
5. **Documentation**: Document custom themes for team consistency

## Configuration Schema

The theme configuration follows this JSON schema:

```json
{
  "theme": {
    "theme": "string (default|minimal|ascii|custom)",
    "suppress_emojis": "boolean",
    "custom_symbols": {
      "symbol_name": "string (max 10 chars)"
    }
  }
}
```

All theme properties are optional and will use sensible defaults if not specified.
