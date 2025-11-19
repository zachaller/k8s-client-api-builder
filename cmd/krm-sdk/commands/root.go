package commands

import (
	"github.com/spf13/cobra"
)

var (
	// rootCmd represents the base command when called without any subcommands
	rootCmd = &cobra.Command{
		Use:   "krm-sdk",
		Short: "KRM SDK - Client-Side Kubernetes Resource Model Framework",
		Long: `KRM SDK is a framework for building client-side Kubernetes abstractions.

Think of it as kubebuilder for client-side hydrators - it provides the same
developer experience as kubebuilder but for generating Kubernetes resources
client-side instead of running controllers server-side.

Use krm-sdk to:
  - Initialize new abstraction projects
  - Scaffold new API types with Go structs and kubebuilder markers
  - Generate validation schemas and CRDs
  - Build custom platform tools that hydrate abstractions into K8s resources`,
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags can be added here
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
}
