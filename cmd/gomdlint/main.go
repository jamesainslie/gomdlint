package main

import (
	"context"
	"fmt"
	"os"

	"github.com/gomdlint/gomdlint/internal/interfaces/cli/commands"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Create root command
	rootCmd := &cobra.Command{
		Use:   "gomdlint",
		Short: "A high-performance Go markdown linter",
		Long: `gomdlint is a fast, extensible markdown linter written in Go.
It provides compatibility with markdownlint while offering superior performance
and comprehensive rule support with plugin extensibility.`,
		Version: fmt.Sprintf("%s (commit: %s, date: %s)", version, commit, date),
		Run: func(cmd *cobra.Command, args []string) {
			// If no subcommand specified, show help or run default linting
			if len(args) == 0 {
				cmd.Help()
				return
			}

			// Default behavior: lint the provided files
			ctx := context.Background()
			lintCmd := commands.NewLintCommand()
			lintCmd.SetArgs(args)
			if err := lintCmd.ExecuteContext(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	// Add global flags
	addGlobalFlags(rootCmd)

	// Add subcommands
	rootCmd.AddCommand(
		commands.NewLintCommand(),
		commands.NewCheckCommand(),
		commands.NewFixCommand(),
		commands.NewConfigCommand(),
		commands.NewThemeCommand(),
		commands.NewRulesCommand(),
		commands.NewPluginCommand(),
		commands.NewStyleCommand(),
		commands.NewVersionCommand(version, commit, date),
	)

	// Execute root command
	if err := rootCmd.ExecuteContext(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func addGlobalFlags(cmd *cobra.Command) {
	// Configuration flags
	cmd.PersistentFlags().StringP("config", "c", "", "Path to configuration file")
	cmd.PersistentFlags().Bool("no-config", false, "Ignore configuration files")

	// Output flags
	cmd.PersistentFlags().StringP("output", "o", "", "Output file (default: stdout)")
	cmd.PersistentFlags().StringP("format", "f", "default", "Output format (default, json, junit, checkstyle)")
	cmd.PersistentFlags().Bool("color", true, "Enable colored output")
	cmd.PersistentFlags().Bool("quiet", false, "Suppress non-error output")
	cmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")

	// Performance flags
	cmd.PersistentFlags().Int("concurrency", 0, "Number of concurrent workers (0 = auto)")
	cmd.PersistentFlags().Bool("cache", true, "Enable result caching")

	// Rule flags
	cmd.PersistentFlags().StringSlice("enable", []string{}, "Enable specific rules")
	cmd.PersistentFlags().StringSlice("disable", []string{}, "Disable specific rules")
	cmd.PersistentFlags().StringSlice("enable-tag", []string{}, "Enable rules by tag")
	cmd.PersistentFlags().StringSlice("disable-tag", []string{}, "Disable rules by tag")
}
