package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewVersionCommand creates the version command.
func NewVersionCommand(version, commit, date string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  `Display detailed version information for gomdlint including build details.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("gomdlint version %s\n", version)
			fmt.Printf("  commit: %s\n", commit)
			fmt.Printf("  built: %s\n", date)
			fmt.Printf("  go: %s\n", "go1.24+")
		},
	}
}
