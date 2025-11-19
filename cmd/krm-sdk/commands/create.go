package commands

import (
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create new resources in the project",
	Long: `Create new resources like API types in your KRM project.

Available subcommands:
  api     Create a new API type with Go structs and hydration template`,
}

func init() {
	rootCmd.AddCommand(createCmd)
}
