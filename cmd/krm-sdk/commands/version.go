package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Version is set during build
	Version = "dev"
	// Commit is set during build
	Commit = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("krm-sdk version %s (commit: %s)\n", Version, Commit)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
