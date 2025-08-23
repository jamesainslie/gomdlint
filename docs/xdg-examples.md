# XDG Base Directory Examples

This document provides examples of how to set up gomdlint configuration files using the XDG Base Directory Specification.

## Directory Structure

```
# User-specific configuration (recommended)
~/.config/gomdlint/
├── config.json              # Main config file
├── project-overrides.json   # Project-specific overrides
└── themes/                   # Custom themes (future)

# System-wide configuration (admin setup)
/etc/xdg/gomdlint/
├── config.json              # Organization defaults
└── strict.json              # Strict validation profile
```

## Example Configurations

### User XDG Config (`~/.config/gomdlint/config.json`)

```json
{
  "default": true,
  "MD013": {
    "line_length": 120
  },
  "MD033": false,
  "MD041": false,
  "theme": {
    "theme": "default",
    "suppress_emojis": false,
    "custom_symbols": {}
  }
}
```

### System XDG Config (`/etc/xdg/gomdlint/config.json`)

```json
{
  "default": true,
  "MD013": {
    "line_length": 100
  },
  "MD033": false,
  "MD041": false,
  "theme": {
    "theme": "minimal",
    "suppress_emojis": false,
    "custom_symbols": {
      "success": "[OK]",
      "error": "[ERROR]",
      "warning": "[WARN]"
    }
  }
}
```

## Setup Commands

### Create User Configuration

```bash
# Create user config directory and file
mkdir -p ~/.config/gomdlint
gomdlint config init

# Or manually create
cat > ~/.config/gomdlint/config.json << EOF
{
  "default": true,
  "MD013": { "line_length": 120 },
  "theme": { "theme": "default" }
}
EOF
```

### Create System Configuration

```bash
# Create system config (requires admin privileges)
sudo mkdir -p /etc/xdg/gomdlint
sudo cp configs/examples/xdg-system-config.json /etc/xdg/gomdlint/config.json
```

### Environment Variables

```bash
# Custom XDG config location
export XDG_CONFIG_HOME="$HOME/my-configs"
gomdlint config which  # Shows: ~/my-configs/gomdlint/

# Multiple system config directories
export XDG_CONFIG_DIRS="/etc/xdg:/usr/local/etc/xdg:/opt/company/xdg"
gomdlint config which  # Shows search in all directories
```

## Migration Examples

### From Legacy to XDG

```bash
# 1. Check current setup
gomdlint config which
# Output: Configuration file: /project/.markdownlint.json
#         Type: Legacy location (backward compatibility)

# 2. Create XDG config
gomdlint config init
# Output: Configuration file created: ~/.config/gomdlint/config.json
#         Created in XDG config directory (recommended)

# 3. Copy existing settings
cp .markdownlint.json ~/.config/gomdlint/config.json

# 4. Verify
gomdlint config which
# Output: Configuration file: /project/.markdownlint.json  (legacy still takes precedence)

# 5. Remove legacy to use XDG
mv .markdownlint.json .markdownlint.json.backup

gomdlint config which
# Output: Configuration file: ~/.config/gomdlint/config.json
#         Type: XDG Base Directory (recommended)
```

### Hybrid Setup (Project + Personal)

```bash
# Project config (legacy location for team compatibility)
cat > .markdownlint.json << EOF
{
  "default": true,
  "MD013": { "line_length": 100 },
  "theme": { "theme": "ascii", "suppress_emojis": true }
}
EOF

# Personal overrides (XDG location)
mkdir -p ~/.config/gomdlint
cat > ~/.config/gomdlint/config.json << EOF
{
  "default": true,
  "MD013": { "line_length": 120 },
  "theme": { "theme": "default", "suppress_emojis": false }
}
EOF

# Project config takes precedence when in project directory
cd /path/to/project
gomdlint config which
# Output: Configuration file: /path/to/project/.markdownlint.json (legacy)

# Personal config used when outside project
cd ~
gomdlint config which
# Output: Configuration file: ~/.config/gomdlint/config.json (XDG)
```

## Best Practices

1. **New Projects**: Use XDG locations (`gomdlint config init`)
2. **Existing Projects**: Keep legacy files for team compatibility
3. **Personal Preferences**: Use XDG user directory (`~/.config/gomdlint/`)
4. **Organization Standards**: Use XDG system directory (`/etc/xdg/gomdlint/`)
5. **CI/CD**: Use project-specific config files in version control

## Troubleshooting

### Configuration Not Found

```bash
# Check all search paths
gomdlint config which

# Create config in appropriate location
gomdlint config init                    # XDG (recommended)
gomdlint config init --legacy          # Current directory

# Verify environment variables
echo $XDG_CONFIG_HOME
echo $XDG_CONFIG_DIRS
```

### Wrong Config Being Used

```bash
# Check priority order
gomdlint config which

# Current directory config always takes precedence
# Remove or rename to use XDG config
mv .markdownlint.json .markdownlint.json.backup
```

### Permission Issues

```bash
# Fix user config permissions
chmod 644 ~/.config/gomdlint/config.json
chmod 755 ~/.config/gomdlint/

# Fix system config permissions (admin)
sudo chmod 644 /etc/xdg/gomdlint/config.json
sudo chmod 755 /etc/xdg/gomdlint/
```
