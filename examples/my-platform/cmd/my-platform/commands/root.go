package commands

import (
	"github.com/yourusername/krm-sdk/pkg/cli"
)

// Execute runs the root command
func Execute() error {
	rootCmd := cli.BuildRootCommand("my-platform")
	return rootCmd.Execute()
}
