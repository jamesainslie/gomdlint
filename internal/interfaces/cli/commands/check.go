package commands

import (
	"github.com/spf13/cobra"
)

// NewCheckCommand creates the check command (alias for lint with exit code behavior).
func NewCheckCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check [files...]",
		Short: "Check markdown files (alias for lint)",
		Long: `Check markdown files for linting violations.
		
This is an alias for the 'lint' command with default behavior suitable for CI/CD.
Exit code is non-zero if any violations are found.`,
		Args: cobra.ArbitraryArgs,
		RunE: runLint, // Reuse the lint command logic
	}

	// Inherit the same flags as lint
	cmd.Flags().StringSlice("ignore", []string{}, "Ignore files matching these patterns")
	cmd.Flags().Bool("stdin", false, "Read from stdin instead of files")
	cmd.Flags().String("stdin-name", "stdin", "Name for stdin input")
	cmd.Flags().Bool("dot", false, "Include hidden files and directories")

	return cmd
}
