package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/gomdlint/gomdlint/internal/app/service"
	"github.com/gomdlint/gomdlint/internal/domain/value"
)

// NewStyleCommand creates the style management command
func NewStyleCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "style",
		Short: "Style configuration management",
		Long:  `Manage predefined style configurations for different use cases.`,
	}

	cmd.AddCommand(
		newStyleListCommand(),
		newStyleShowCommand(),
		newStyleApplyCommand(),
		newStyleCreateCommand(),
		newStyleValidateCommand(),
	)

	return cmd
}

// newStyleListCommand creates the style list subcommand
func newStyleListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available styles",
		Long:  "Display all available predefined styles with descriptions.",
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")
			return listStyles(verbose)
		},
	}

	cmd.Flags().BoolP("verbose", "v", false, "Show detailed style information")
	return cmd
}

// newStyleShowCommand creates the style show subcommand
func newStyleShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show [style-name]",
		Short: "Show style configuration",
		Long:  "Display the complete configuration for a specific style.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return showStyle(args[0])
		},
	}
}

// newStyleApplyCommand creates the style apply subcommand
func newStyleApplyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply [style-name]",
		Short: "Apply a style to current configuration",
		Long:  "Generate a configuration file based on a predefined style.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			output, _ := cmd.Flags().GetString("output")
			merge, _ := cmd.Flags().GetBool("merge")
			return applyStyle(args[0], output, merge)
		},
	}

	cmd.Flags().StringP("output", "o", ".markdownlint.json", "Output configuration file")
	cmd.Flags().Bool("merge", false, "Merge with existing configuration instead of replacing")
	return cmd
}

// newStyleCreateCommand creates the style create subcommand
func newStyleCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [name] [config-file]",
		Short: "Create a custom style",
		Long:  "Create a custom style from an existing configuration file.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			description, _ := cmd.Flags().GetString("description")
			return createStyle(args[0], args[1], description)
		},
	}

	cmd.Flags().StringP("description", "d", "", "Style description")
	return cmd
}

// newStyleValidateCommand creates the style validate subcommand
func newStyleValidateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "validate [style-name]",
		Short: "Validate a style configuration",
		Long:  "Validate that a style configuration is correct and complete.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return validateStyle(args[0])
		},
	}
}

// Implementation functions

func listStyles(verbose bool) error {
	styleRegistry := service.NewStyleRegistry()
	styles := styleRegistry.ListStyles()

	if len(styles) == 0 {
		fmt.Println("No styles available.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	if verbose {
		fmt.Fprintln(w, "NAME\tDESCRIPTION\tRULES\tEXTENDS")
		for _, styleName := range styles {
			style, err := styleRegistry.GetStyle(styleName)
			if err != nil {
				continue
			}

			description := "No description"
			if style.Schema != "" {
				description = "Built-in style"
			}

			ruleCount := len(style.Rules)
			extends := len(style.Extends)

			fmt.Fprintf(w, "%s\t%s\t%d\t%d\n", styleName, description, ruleCount, extends)
		}
	} else {
		fmt.Fprintln(w, "NAME\tDESCRIPTION")
		for _, styleName := range styles {
			style, err := styleRegistry.GetStyle(styleName)
			if err != nil {
				continue
			}

			description := "Built-in style"
			if style.Version != "" {
				description = fmt.Sprintf("Built-in style v%s", style.Version)
			}

			fmt.Fprintf(w, "%s\t%s\n", styleName, description)
		}
	}

	return w.Flush()
}

func showStyle(styleName string) error {
	styleRegistry := service.NewStyleRegistry()
	style, err := styleRegistry.GetStyle(styleName)
	if err != nil {
		return fmt.Errorf("style not found: %w", err)
	}

	data, err := json.MarshalIndent(style, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format style: %w", err)
	}

	fmt.Println(string(data))
	return nil
}

func applyStyle(styleName, outputPath string, merge bool) error {
	styleRegistry := service.NewStyleRegistry()
	style, err := styleRegistry.GetStyle(styleName)
	if err != nil {
		return fmt.Errorf("style not found: %w", err)
	}

			finalConfig := style

	// If merge is requested, load existing config and merge
	if merge {
		if _, err := os.Stat(outputPath); err == nil {
			existingData, err := os.ReadFile(outputPath)
			if err != nil {
				return fmt.Errorf("failed to read existing config: %w", err)
			}

			var existingConfig value.Config
			if err := json.Unmarshal(existingData, &existingConfig); err != nil {
				return fmt.Errorf("failed to parse existing config: %w", err)
			}

			finalConfig = existingConfig.Merge(style)
		}
	}

	// Write configuration file
	data, err := json.MarshalIndent(finalConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format config: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	action := "applied"
	if merge {
		action = "merged"
	}

	fmt.Printf("Style '%s' %s to %s\n", styleName, action, outputPath)
	return nil
}

func createStyle(name, configFile, description string) error {
	// Load configuration from file
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config value.Config
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

			// Add description to schema field for now
		// In a full implementation, this would use a metadata field
		if description != "" {
			config.Schema = description
		}

	// Register the style
	styleRegistry := service.NewStyleRegistry()
	styleRegistry.RegisterStyle(name, &config)

	fmt.Printf("Custom style '%s' created successfully\n", name)
	return nil
}

func validateStyle(styleName string) error {
	styleRegistry := service.NewStyleRegistry()
	style, err := styleRegistry.GetStyle(styleName)
	if err != nil {
		return fmt.Errorf("style not found: %w", err)
	}

	// Validate configuration
	if err := style.Validate(); err != nil {
		return fmt.Errorf("style validation failed: %w", err)
	}

	fmt.Printf("Style '%s' is valid\n", styleName)
	
	// Show validation summary
	fmt.Printf("Configuration summary:\n")
	fmt.Printf("  Rules configured: %d\n", len(style.Rules))
	fmt.Printf("  Plugins configured: %d\n", len(style.Plugins))
	fmt.Printf("  Parsers configured: %d\n", len(style.Parsers))
	fmt.Printf("  Extends: %v\n", style.Extends)

	return nil
}
