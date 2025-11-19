package commands

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate code and manifests",
	Long: `Generate code and manifests for the current project.

This command runs controller-gen to generate:
  - OpenAPI validation schemas
  - CRD manifests
  - DeepCopy methods for Go types

This command should be run from within a KRM project directory.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		verbose, _ := cmd.Flags().GetBool("verbose")
		
		// Check if we're in a project directory
		if _, err := os.Stat("PROJECT"); err != nil {
			return fmt.Errorf("not in a KRM project directory (PROJECT file not found)")
		}
		
		if verbose {
			fmt.Println("Running code generation...")
		}
		
		// Run make generate
		makeCmd := exec.Command("make", "generate")
		makeCmd.Stdout = os.Stdout
		makeCmd.Stderr = os.Stderr
		
		if err := makeCmd.Run(); err != nil {
			return fmt.Errorf("failed to run make generate: %w", err)
		}
		
		fmt.Println("\n✓ Code generation completed successfully!")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
}

