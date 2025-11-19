package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zachaller/k8s-client-api-builder/pkg/scaffold"
)

var (
	apiGroup   string
	apiVersion string
	apiKind    string
)

var createAPICmd = &cobra.Command{
	Use:   "api",
	Short: "Create a new API type",
	Long: `Create a new API type with Go struct definitions and hydration template.

This command generates:
  - Go struct file with kubebuilder validation markers
  - Hydration template YAML file
  - Updates to registration and scheme code
  - Sample instance file

Example:
  krm-sdk create api --group platform --version v1alpha1 --kind WebService`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if apiGroup == "" || apiVersion == "" || apiKind == "" {
			return fmt.Errorf("--group, --version, and --kind are required")
		}

		verbose, _ := cmd.Flags().GetBool("verbose")

		scaffolder := scaffold.NewAPIScaffolder(scaffold.APIConfig{
			Group:   apiGroup,
			Version: apiVersion,
			Kind:    apiKind,
			Verbose: verbose,
		})

		if err := scaffolder.Scaffold(); err != nil {
			return fmt.Errorf("failed to scaffold API: %w", err)
		}

		fmt.Printf("\n✓ API '%s' created successfully!\n\n", apiKind)
		fmt.Println("Next steps:")
		fmt.Printf("  1. Edit api/%s/%s_types.go to add fields and validation\n", apiVersion, scaffold.ToSnakeCase(apiKind))
		fmt.Printf("  2. Edit api/%s/%s_template.yaml to define resource hydration\n", apiVersion, scaffold.ToSnakeCase(apiKind))
		fmt.Println("  3. Run 'make generate' to update generated code")
		fmt.Println("  4. Run 'make build' to build the project binary")
		fmt.Println()

		return nil
	},
}

func init() {
	createCmd.AddCommand(createAPICmd)

	createAPICmd.Flags().StringVar(&apiGroup, "group", "", "API group name (required)")
	createAPICmd.Flags().StringVar(&apiVersion, "version", "", "API version (required)")
	createAPICmd.Flags().StringVar(&apiKind, "kind", "", "API kind name (required)")
	createAPICmd.MarkFlagRequired("group")
	createAPICmd.MarkFlagRequired("version")
	createAPICmd.MarkFlagRequired("kind")
}
