package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/gomdlint/gomdlint/internal/app/service"
	"github.com/gomdlint/gomdlint/pkg/gomdlint/plugin"
)

// NewPluginCommand creates the plugin management command
func NewPluginCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Plugin management",
		Long:  `Manage gomdlint plugins for custom rules and functionality.`,
	}

	cmd.AddCommand(
		newPluginListCommand(),
		newPluginInstallCommand(),
		newPluginUninstallCommand(),
		newPluginInfoCommand(),
		newPluginBuildCommand(),
		newPluginHealthCommand(),
	)

	return cmd
}

// newPluginListCommand creates the plugin list subcommand
func newPluginListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installed plugins",
		Long:  "Display all currently installed and loaded plugins with their status.",
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")
			return listPlugins(verbose)
		},
	}

	cmd.Flags().BoolP("verbose", "v", false, "Show detailed plugin information")
	return cmd
}

// newPluginInstallCommand creates the plugin install subcommand
func newPluginInstallCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install [plugin-path]",
		Short: "Install a plugin",
		Long:  "Install a plugin from a .so file or build and install from source directory.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			force, _ := cmd.Flags().GetBool("force")
			return installPlugin(args[0], force)
		},
	}

	cmd.Flags().Bool("force", false, "Force installation even if plugin exists")
	return cmd
}

// newPluginUninstallCommand creates the plugin uninstall subcommand
func newPluginUninstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall [plugin-name]",
		Short: "Uninstall a plugin",
		Long:  "Remove and unload a plugin by name.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return uninstallPlugin(args[0])
		},
	}
}

// newPluginInfoCommand creates the plugin info subcommand
func newPluginInfoCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "info [plugin-name]",
		Short: "Show plugin information",
		Long:  "Display detailed information about a specific plugin.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return showPluginInfo(args[0])
		},
	}
}

// newPluginBuildCommand creates the plugin build subcommand
func newPluginBuildCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build [source-dir]",
		Short: "Build a plugin from source",
		Long:  "Build a Go plugin from source code in the specified directory.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			output, _ := cmd.Flags().GetString("output")
			return buildPlugin(args[0], output)
		},
	}

	cmd.Flags().StringP("output", "o", "", "Output plugin file path")
	return cmd
}

// newPluginHealthCommand creates the plugin health subcommand
func newPluginHealthCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Check plugin health",
		Long:  "Perform health checks on all loaded plugins.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return checkPluginHealth()
		},
	}
}

// Implementation functions

func listPlugins(verbose bool) error {
	pluginManager := service.GetGlobalPluginManager()
	plugins := pluginManager.GetAllPlugins()

	if len(plugins) == 0 {
		fmt.Println("No plugins installed.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	
	if verbose {
		fmt.Fprintln(w, "NAME\tVERSION\tDESCRIPTION\tAUTHOR\tRULES\tSTATUS")
		for name, p := range plugins {
			status := pluginManager.GetPluginStatus(name)
			statusStr := "Unknown"
			if status.Loaded && status.Initialized {
				statusStr = "Active"
			} else if status.Loaded {
				statusStr = "Loaded"
			} else {
				statusStr = "Error"
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\n",
				p.Name(),
				p.Version(),
				p.Description(),
				p.Author(),
				len(p.Rules()),
				statusStr)
		}
	} else {
		fmt.Fprintln(w, "NAME\tVERSION\tDESCRIPTION\tRULES")
		for _, p := range plugins {
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\n",
				p.Name(),
				p.Version(),
				p.Description(),
				len(p.Rules()))
		}
	}

	return w.Flush()
}

func installPlugin(pluginPath string, force bool) error {
	pluginManager := service.GetGlobalPluginManager()

	// Validate plugin file
	if filepath.Ext(pluginPath) != ".so" {
		// Check if it's a directory (source code)
		if stat, err := os.Stat(pluginPath); err == nil && stat.IsDir() {
			// Build from source first
			builtPath, err := buildPluginFromSource(pluginPath)
			if err != nil {
				return fmt.Errorf("failed to build plugin from source: %w", err)
			}
			pluginPath = builtPath
		} else {
			return fmt.Errorf("plugin must be a .so file or source directory")
		}
	}

	// Check if plugin already exists
	if !force {
		// Try to load temporarily to get plugin name
		tempManager := service.NewPluginManager(pluginManager.GetConfig())
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := tempManager.LoadPlugin(ctx, pluginPath); err == nil {
			plugins := tempManager.GetAllPlugins()
			for name := range plugins {
				if _, exists := pluginManager.GetAllPlugins()[name]; exists {
					return fmt.Errorf("plugin %s already installed (use --force to override)", name)
				}
			}
		}
	}

	// Install plugin
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := pluginManager.LoadPlugin(ctx, pluginPath); err != nil {
		return fmt.Errorf("failed to install plugin: %w", err)
	}

	fmt.Printf("Plugin installed successfully: %s\n", pluginPath)
	return nil
}

func uninstallPlugin(pluginName string) error {
	pluginManager := service.GetGlobalPluginManager()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := pluginManager.UnloadPlugin(ctx, pluginName); err != nil {
		return fmt.Errorf("failed to uninstall plugin: %w", err)
	}

	fmt.Printf("Plugin uninstalled successfully: %s\n", pluginName)
	return nil
}

func showPluginInfo(pluginName string) error {
	pluginManager := service.GetGlobalPluginManager()

	info, err := pluginManager.GetPluginInfo(pluginName)
	if err != nil {
		return fmt.Errorf("failed to get plugin info: %w", err)
	}

	status := pluginManager.GetPluginStatus(pluginName)

	fmt.Printf("Plugin Information:\n")
	fmt.Printf("  Name: %s\n", info.Name)
	fmt.Printf("  Version: %s\n", info.Version)
	fmt.Printf("  Description: %s\n", info.Description)
	fmt.Printf("  Author: %s\n", info.Author)
	fmt.Printf("  Rules: %d\n", info.RuleCount)
	fmt.Printf("  Status: %s\n", getStatusString(status))

	if status.Error != nil {
		fmt.Printf("  Error: %s\n", status.Error.Error())
	}

	// Show rules provided by the plugin
	plugin, err := pluginManager.GetPlugin(pluginName)
	if err == nil {
		fmt.Printf("\nRules provided:\n")
		for _, rule := range plugin.Rules() {
			fmt.Printf("  - %s: %s\n", strings.Join(rule.Names(), ", "), rule.Description())
		}
	}

	return nil
}

func buildPlugin(sourceDir, outputPath string) error {
	// Validate source directory
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return fmt.Errorf("source directory does not exist: %s", sourceDir)
	}

	// Default output path
	if outputPath == "" {
		outputPath = filepath.Join(sourceDir, "plugin.so")
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	fmt.Printf("Building plugin from %s...\n", sourceDir)

	// Build plugin using go build
	cmd := exec.Command("go", "build", "-buildmode=plugin", "-o", outputPath, ".")
	cmd.Dir = sourceDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build plugin: %w", err)
	}

	fmt.Printf("Plugin built successfully: %s\n", outputPath)
	return nil
}

func checkPluginHealth() error {
	pluginManager := service.GetGlobalPluginManager()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	healthResults := pluginManager.HealthCheckAll(ctx)

	if len(healthResults) == 0 {
		fmt.Println("No plugins installed.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PLUGIN\tSTATUS\tERROR")

	allHealthy := true
	for name, err := range healthResults {
		status := "Healthy"
		errorMsg := ""
		
		if err != nil {
			status = "Unhealthy"
			errorMsg = err.Error()
			allHealthy = false
		}

		fmt.Fprintf(w, "%s\t%s\t%s\n", name, status, errorMsg)
	}

	if err := w.Flush(); err != nil {
		return err
	}

	if !allHealthy {
		fmt.Println("\nSome plugins are experiencing health issues.")
		return fmt.Errorf("plugin health check failed")
	}

	fmt.Println("\nAll plugins are healthy.")
	return nil
}

// Helper functions

func getStatusString(status plugin.PluginStatus) string {
	if !status.Loaded {
		return "Not Loaded"
	}
	if !status.Initialized {
		return "Loaded (Not Initialized)"
	}
	if status.Error != nil {
		return "Error"
	}
	return "Active"
}

func buildPluginFromSource(sourceDir string) (string, error) {
	outputPath := filepath.Join(sourceDir, "plugin.so")
	
	// Check if go.mod exists
	goModPath := filepath.Join(sourceDir, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		return "", fmt.Errorf("source directory must contain go.mod file")
	}

	// Build the plugin
	if err := buildPlugin(sourceDir, outputPath); err != nil {
		return "", err
	}

	return outputPath, nil
}
