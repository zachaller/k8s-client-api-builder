package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yourusername/krm-sdk/pkg/scaffold"
)

var (
	initDomain string
	initRepo   string
)

var initCmd = &cobra.Command{
	Use:   "init [project-name]",
	Short: "Initialize a new KRM abstraction project",
	Long: `Initialize a new KRM abstraction project with the necessary scaffolding.

This command creates a new project directory with:
  - Go module and project structure
  - Makefile for building and code generation
  - API directory for defining abstractions
  - Configuration directories for CRDs and samples
  - Main entry point for the project binary

Example:
  krm-sdk init my-platform --domain platform.mycompany.com`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName := args[0]
		
		verbose, _ := cmd.Flags().GetBool("verbose")
		
		scaffolder := scaffold.NewProjectScaffolder(scaffold.ProjectConfig{
			Name:    projectName,
			Domain:  initDomain,
			Repo:    initRepo,
			Verbose: verbose,
		})
		
		if err := scaffolder.Scaffold(); err != nil {
			return fmt.Errorf("failed to scaffold project: %w", err)
		}
		
		fmt.Printf("\n✓ Project '%s' initialized successfully!\n\n", projectName)
		fmt.Println("Next steps:")
		fmt.Printf("  cd %s\n", projectName)
		fmt.Println("  krm-sdk create api --group <group> --version <version> --kind <Kind>")
		fmt.Println("  make build")
		fmt.Println()
		
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	
	initCmd.Flags().StringVar(&initDomain, "domain", "example.com", "domain for the project")
	initCmd.Flags().StringVar(&initRepo, "repo", "", "repository path (default: inferred from project name)")
}

