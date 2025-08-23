package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/gomdlint/gomdlint/internal/shared/utils"
	"github.com/spf13/cobra"
)

// NewThemeCommand creates the theme command for theme management
func NewThemeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "theme",
		Short: "Manage themes",
		Long:  `Manage gomdlint themes including creating, editing, and listing available themes.`,
	}

	cmd.AddCommand(
		newThemeListCommand(),
		newThemeShowCommand(),
		newThemeCreateCommand(),
		newThemeEditCommand(),
		newThemeDeleteCommand(),
		newThemeInstallCommand(),
	)

	return cmd
}

func newThemeListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available themes",
		Long:  `List all available themes in the theme directory.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return listThemes()
		},
	}
}

func newThemeShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show <theme-name>",
		Short: "Show theme details",
		Long:  `Display the complete definition of a specific theme.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return showTheme(args[0])
		},
	}
}

func newThemeCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <theme-name>",
		Short: "Create a new theme",
		Long:  `Create a new theme interactively or from a template.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			template, _ := cmd.Flags().GetString("template")
			interactive, _ := cmd.Flags().GetBool("interactive")
			return createTheme(args[0], template, interactive)
		},
	}

	cmd.Flags().StringP("template", "t", "minimal", "Base theme to use as template (default, minimal, ascii)")
	cmd.Flags().BoolP("interactive", "i", false, "Create theme interactively")
	return cmd
}

func newThemeEditCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "edit <theme-name>",
		Short: "Edit an existing theme",
		Long:  `Edit an existing theme definition. Opens the theme file in the default editor.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return editTheme(args[0])
		},
	}
}

func newThemeDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <theme-name>",
		Short: "Delete a theme",
		Long:  `Delete a custom theme. Built-in themes cannot be deleted.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			force, _ := cmd.Flags().GetBool("force")
			return deleteTheme(args[0], force)
		},
	}

	cmd.Flags().BoolP("force", "f", false, "Delete without confirmation")
	return cmd
}

func newThemeInstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install built-in themes",
		Long:  `Install or reinstall the built-in themes to the theme directory.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return installBuiltinThemes()
		},
	}
}

// listThemes displays all available themes
func listThemes() error {
	const appName = "gomdlint"

	tm, err := utils.NewThemeManager(appName)
	if err != nil {
		return fmt.Errorf("failed to initialize theme manager: %w", err)
	}

	themes, err := tm.ListThemes()
	if err != nil {
		return fmt.Errorf("failed to list themes: %w", err)
	}

	if len(themes) == 0 {
		fmt.Println("No themes found. Run 'gomdlint theme install' to install built-in themes.")
		return nil
	}

	// Get themed symbols
	helper := NewThemedCommandHelper()

	// Create styled output
	styles := newThemeListStyles()

	fmt.Printf("%s Available Themes:\n", styles.header.Render(helper.Theme()))
	fmt.Println()

	// Calculate column widths for proper alignment
	maxNameLen := 4    // "NAME"
	maxDescLen := 11   // "DESCRIPTION"
	maxAuthorLen := 6  // "AUTHOR"
	maxVersionLen := 7 // "VERSION"

	// Find the maximum width for each column (excluding styles for calculation)
	for _, theme := range themes {
		nameLen := len(theme.Name)
		if isBuiltinTheme(theme.Name) {
			nameLen += len(" (built-in)")
		}
		if nameLen > maxNameLen {
			maxNameLen = nameLen
		}

		descLen := len(theme.Description)
		if descLen > 45 { // Cap description length
			descLen = 45
		}
		if descLen > maxDescLen {
			maxDescLen = descLen
		}

		authorLen := len(theme.Author)
		if authorLen == 0 {
			authorLen = 1 // "-"
		}
		if authorLen > maxAuthorLen {
			maxAuthorLen = authorLen
		}

		versionLen := len(theme.Version)
		if versionLen == 0 {
			versionLen = 1 // "-"
		}
		if versionLen > maxVersionLen {
			maxVersionLen = versionLen
		}
	}

	// Print header with proper spacing
	fmt.Printf("%-*s  %-*s  %-*s  %-*s  %s\n",
		maxNameLen, "NAME",
		maxDescLen, "DESCRIPTION",
		maxAuthorLen, "AUTHOR",
		maxVersionLen, "VERSION",
		"SYMBOLS")

	// Print separator line
	fmt.Printf("%s  %s  %s  %s  %s\n",
		strings.Repeat("─", maxNameLen),
		strings.Repeat("─", maxDescLen),
		strings.Repeat("─", maxAuthorLen),
		strings.Repeat("─", maxVersionLen),
		strings.Repeat("─", 7))

	// Print theme data
	for _, theme := range themes {
		// Format name with styling
		name := theme.Name
		styledName := name
		if isBuiltinTheme(name) {
			name = name + " (built-in)"
			styledName = styles.builtin.Render(name)
		} else {
			styledName = styles.custom.Render(name)
		}

		// Format description
		description := theme.Description
		if len(description) > 45 {
			description = description[:42] + "..."
		}

		// Format author
		author := theme.Author
		if author == "" {
			author = "-"
		}

		// Format version
		version := theme.Version
		if version == "" {
			version = "-"
		}

		symbolCount := strconv.Itoa(len(theme.Symbols))

		// Print with calculated spacing (account for style codes in name)
		nameSpacing := maxNameLen - len(name) + len(styledName)
		fmt.Printf("%-*s  %-*s  %-*s  %-*s  %s\n",
			nameSpacing, styledName,
			maxDescLen, description,
			maxAuthorLen, author,
			maxVersionLen, version,
			symbolCount)
	}

	return nil
}

// showTheme displays detailed information about a specific theme
func showTheme(name string) error {
	const appName = "gomdlint"

	tm, err := utils.NewThemeManager(appName)
	if err != nil {
		return fmt.Errorf("failed to initialize theme manager: %w", err)
	}

	theme, err := tm.LoadTheme(name)
	if err != nil {
		return fmt.Errorf("failed to load theme: %w", err)
	}

	// Get themed symbols
	helper := NewThemedCommandHelper()

	// Create styled output
	styles := newThemeShowStyles()

	// Header
	fmt.Printf("%s %s\n", styles.header.Render(helper.Theme()+" Theme:"), styles.name.Render(theme.Name))
	fmt.Println()

	// Metadata
	if theme.Description != "" {
		fmt.Printf("%s %s\n", styles.label.Render("Description:"), theme.Description)
	}
	if theme.Author != "" {
		fmt.Printf("%s %s\n", styles.label.Render("Author:"), theme.Author)
	}
	if theme.Version != "" {
		fmt.Printf("%s %s\n", styles.label.Render("Version:"), theme.Version)
	}

	// Symbols
	fmt.Println()
	fmt.Println(styles.section.Render("Symbols:"))
	if len(theme.Symbols) == 0 {
		fmt.Println("  No custom symbols defined")
	} else {
		// Find the longest key for proper alignment
		maxKeyLen := 0
		for key := range theme.Symbols {
			if len(key) > maxKeyLen {
				maxKeyLen = len(key)
			}
		}

		// Print symbols with proper alignment
		for key, value := range theme.Symbols {
			fmt.Printf("  %-*s: %s\n", maxKeyLen, styles.symbolKey.Render(key), styles.symbolValue.Render(value))
		}
	}

	// Settings
	if len(theme.Settings) > 0 {
		fmt.Println()
		fmt.Println(styles.section.Render("Settings:"))
		settingsData, _ := json.MarshalIndent(theme.Settings, "  ", "  ")
		fmt.Printf("  %s\n", string(settingsData))
	}

	// File location
	fmt.Println()
	themePath := fmt.Sprintf("%s/%s.json", tm.GetThemesDirectory(), name)
	fmt.Printf("%s %s\n", styles.label.Render("Location:"), styles.path.Render(themePath))

	return nil
}

// createTheme creates a new theme
func createTheme(name, template string, interactive bool) error {
	const appName = "gomdlint"

	tm, err := utils.NewThemeManager(appName)
	if err != nil {
		return fmt.Errorf("failed to initialize theme manager: %w", err)
	}

	// Check if theme already exists
	if tm.ThemeExists(name) {
		return fmt.Errorf("theme '%s' already exists", name)
	}

	var theme utils.ThemeDefinition

	if interactive {
		// Interactive theme creation
		theme, err = createThemeInteractive(name, tm)
		if err != nil {
			return fmt.Errorf("failed to create theme interactively: %w", err)
		}
	} else {
		// Create from template
		baseTheme, err := tm.LoadTheme(template)
		if err != nil {
			return fmt.Errorf("failed to load template theme '%s': %w", template, err)
		}

		theme = utils.ThemeDefinition{
			Name:        name,
			Description: fmt.Sprintf("Custom theme based on %s", template),
			Author:      "",
			Version:     "1.0.0",
			Symbols:     make(map[string]string),
			Settings:    make(map[string]interface{}),
		}

		// Copy symbols from template
		for key, value := range baseTheme.Symbols {
			theme.Symbols[key] = value
		}

		// Copy settings from template
		for key, value := range baseTheme.Settings {
			theme.Settings[key] = value
		}
	}

	// Validate theme
	if err := tm.ValidateTheme(&theme); err != nil {
		return fmt.Errorf("theme validation failed: %w", err)
	}

	// Save theme
	if err := tm.SaveTheme(&theme); err != nil {
		return fmt.Errorf("failed to save theme: %w", err)
	}

	// Get themed symbols
	helper := NewThemedCommandHelper()

	fmt.Printf("%s Created theme '%s' successfully\n", helper.Success(), name)
	fmt.Printf("%s Location: %s/%s.json\n", helper.Location(), tm.GetThemesDirectory(), name)
	fmt.Printf("%s Edit with: gomdlint theme edit %s\n", helper.Edit(), name)

	return nil
}

// editTheme opens a theme for editing
func editTheme(name string) error {
	const appName = "gomdlint"

	tm, err := utils.NewThemeManager(appName)
	if err != nil {
		return fmt.Errorf("failed to initialize theme manager: %w", err)
	}

	// Check if theme exists
	if !tm.ThemeExists(name) {
		return fmt.Errorf("theme '%s' not found", name)
	}

	// Get themed symbols
	helper := NewThemedCommandHelper()

	// Check if it's a built-in theme
	if isBuiltinTheme(name) {
		fmt.Printf("%s Warning: '%s' is a built-in theme\n", helper.Warning(), name)
		fmt.Printf("%s Consider creating a copy: gomdlint theme create my-%s --template %s\n", helper.Tip(), name, name)
		fmt.Println()

		// Ask for confirmation
		if !confirmAction(fmt.Sprintf("Edit built-in theme '%s'?", name)) {
			fmt.Println("Operation cancelled")
			return nil
		}
	}

	// Get theme file path
	themePath := fmt.Sprintf("%s/%s.json", tm.GetThemesDirectory(), name)

	// Try to open with default editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		// Try common editors as fallbacks
		editors := []string{"nano", "vim", "vi"}
		for _, e := range editors {
			if _, err := exec.LookPath(e); err == nil {
				editor = e
				break
			}
		}
		if editor == "" {
			return fmt.Errorf("no editor found. Set EDITOR environment variable or install nano, vim, or vi")
		}
	}

	fmt.Printf("%s Opening theme '%s' in %s...\n", helper.Theme(), name, editor)
	fmt.Printf("%s File: %s\n", helper.Location(), themePath)

	// Execute the editor
	cmd := exec.Command(editor, themePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open editor '%s': %w\nTip: Set EDITOR environment variable to your preferred editor", editor, err)
	}

	fmt.Printf("%s Theme editing completed\n", helper.Success())
	fmt.Printf("%s Validate with: gomdlint config validate\n", helper.Tip())

	return nil
}

// deleteTheme removes a theme
func deleteTheme(name string, force bool) error {
	const appName = "gomdlint"

	tm, err := utils.NewThemeManager(appName)
	if err != nil {
		return fmt.Errorf("failed to initialize theme manager: %w", err)
	}

	// Check if theme exists
	if !tm.ThemeExists(name) {
		return fmt.Errorf("theme '%s' not found", name)
	}

	// Confirmation for non-force deletion
	if !force {
		if !confirmAction(fmt.Sprintf("Delete theme '%s'?", name)) {
			fmt.Println("Operation cancelled")
			return nil
		}
	}

	// Delete theme
	if err := tm.DeleteTheme(name); err != nil {
		return fmt.Errorf("failed to delete theme: %w", err)
	}

	// Get themed symbols
	helper := NewThemedCommandHelper()

	fmt.Printf("%s Deleted theme '%s' successfully\n", helper.Success(), name)
	return nil
}

// installBuiltinThemes installs the built-in themes
func installBuiltinThemes() error {
	const appName = "gomdlint"

	tm, err := utils.NewThemeManager(appName)
	if err != nil {
		return fmt.Errorf("failed to initialize theme manager: %w", err)
	}

	// Get themed symbols
	helper := NewThemedCommandHelper()

	fmt.Printf("%s Installing built-in themes...\n", helper.Theme())

	if err := tm.InstallBuiltinThemes(); err != nil {
		return fmt.Errorf("failed to install built-in themes: %w", err)
	}

	builtinNames := utils.GetBuiltinThemeNames()
	fmt.Printf("%s Installed %d built-in themes:\n", helper.Success(), len(builtinNames))
	for _, name := range builtinNames {
		fmt.Printf("  • %s\n", name)
	}

	fmt.Printf("\n%s Location: %s\n", helper.Location(), tm.GetThemesDirectory())
	fmt.Printf("%s List themes with: gomdlint theme list\n", helper.Tip())

	return nil
}

// createThemeInteractive creates a theme with interactive prompts
func createThemeInteractive(name string, tm *utils.ThemeManager) (utils.ThemeDefinition, error) {
	theme := utils.ThemeDefinition{
		Name:     name,
		Symbols:  make(map[string]string),
		Settings: make(map[string]interface{}),
	}

	// Get themed symbols
	helper := NewThemedCommandHelper()

	fmt.Printf("%s Creating theme '%s' interactively...\n\n", helper.Theme(), name)

	// Get description
	theme.Description = promptString("Description", fmt.Sprintf("Custom theme: %s", name))

	// Get author
	theme.Author = promptString("Author", "")

	// Get version
	theme.Version = promptString("Version", "1.0.0")

	// Set basic symbols
	fmt.Println("\n📝 Configure symbols (press Enter to skip):")
	symbolKeys := []string{"success", "error", "warning", "info", "processing"}

	for _, key := range symbolKeys {
		value := promptString(fmt.Sprintf("Symbol for '%s'", key), "")
		if value != "" {
			theme.Symbols[key] = value
		}
	}

	// Basic settings
	theme.Settings["use_colors"] = true

	return theme, nil
}

// Helper functions

func newThemeListStyles() struct {
	header  lipgloss.Style
	builtin lipgloss.Style
	custom  lipgloss.Style
} {
	return struct {
		header  lipgloss.Style
		builtin lipgloss.Style
		custom  lipgloss.Style
	}{
		header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12")),
		builtin: lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
			Bold(true),
		custom: lipgloss.NewStyle().
			Foreground(lipgloss.Color("14")),
	}
}

func newThemeShowStyles() struct {
	header      lipgloss.Style
	name        lipgloss.Style
	label       lipgloss.Style
	section     lipgloss.Style
	symbolKey   lipgloss.Style
	symbolValue lipgloss.Style
	path        lipgloss.Style
} {
	return struct {
		header      lipgloss.Style
		name        lipgloss.Style
		label       lipgloss.Style
		section     lipgloss.Style
		symbolKey   lipgloss.Style
		symbolValue lipgloss.Style
		path        lipgloss.Style
	}{
		header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12")),
		name: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("11")),
		label: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("8")),
		section: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("10")),
		symbolKey: lipgloss.NewStyle().
			Foreground(lipgloss.Color("14")),
		symbolValue: lipgloss.NewStyle().
			Foreground(lipgloss.Color("13")),
		path: lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")),
	}
}

func isBuiltinTheme(name string) bool {
	builtinNames := utils.GetBuiltinThemeNames()
	for _, builtin := range builtinNames {
		if name == builtin {
			return true
		}
	}
	return false
}

func confirmAction(message string) bool {
	// Get themed symbols
	helper := NewThemedCommandHelper()

	fmt.Printf("%s %s (y/N): ", helper.Question(), message)
	var response string
	fmt.Scanln(&response)
	return strings.ToLower(strings.TrimSpace(response)) == "y"
}

func promptString(prompt, defaultValue string) string {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultValue)
	} else {
		fmt.Printf("%s: ", prompt)
	}

	var response string
	fmt.Scanln(&response)

	response = strings.TrimSpace(response)
	if response == "" && defaultValue != "" {
		return defaultValue
	}

	return response
}
