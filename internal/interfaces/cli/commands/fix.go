package commands

import (
	"github.com/spf13/cobra"
)

// NewFixCommand creates the fix command for auto-fixing violations.
func NewFixCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fix [files...]",
		Short: "Automatically fix markdown violations",
		Long: `Automatically fix markdown violations where possible.

This command will:
- Lint the specified files
- Apply automatic fixes for violations that support it
- Save the fixed content back to the original files
- Report which violations were fixed

Examples:
  gomdlint fix README.md
  gomdlint fix docs/*.md
  gomdlint fix --dry-run *.md`,
		Args: cobra.ArbitraryArgs,
		RunE: runFix,
	}

	cmd.Flags().Bool("dry-run", false, "Show what would be fixed without making changes")
	cmd.Flags().StringSlice("ignore", []string{}, "Ignore files matching these patterns")
	cmd.Flags().Bool("dot", false, "Include hidden files and directories")

	return cmd
}

func runFix(cmd *cobra.Command, args []string) error {
	// For now, delegate to lint with fix flag
	cmd.Flags().Set("fix", "true")
	return runLint(cmd, args)
}
