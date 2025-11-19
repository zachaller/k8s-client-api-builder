package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// BuildRootCommand builds the root command for a generated project
func BuildRootCommand(projectName string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   projectName,
		Short: fmt.Sprintf("%s - KRM abstraction hydrator", projectName),
		Long: fmt.Sprintf(`%s is a KRM-based platform abstraction tool.

It validates and hydrates custom abstractions into Kubernetes resources.`, projectName),
	}
	
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
	
	// Add subcommands
	rootCmd.AddCommand(BuildGenerateCommand())
	rootCmd.AddCommand(BuildValidateCommand())
	rootCmd.AddCommand(BuildApplyCommand())
	
	return rootCmd
}

// BuildGenerateCommand builds the generate command
func BuildGenerateCommand() *cobra.Command {
	var (
		outputDir string
		overlay   string
		validate  bool
	)
	
	cmd := &cobra.Command{
		Use:   "generate -f <file|directory>",
		Short: "Generate Kubernetes resources from abstractions",
		Long: `Generate Kubernetes resources from abstraction instances.

This command reads abstraction instances, validates them (optionally),
and hydrates them into standard Kubernetes resources.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			inputFiles, err := cmd.Flags().GetStringSlice("file")
			if err != nil || len(inputFiles) == 0 {
				return fmt.Errorf("--file/-f is required")
			}
			
			verbose, _ := cmd.Flags().GetBool("verbose")
			
			generator := NewGenerator(GeneratorOptions{
				InputFiles: inputFiles,
				OutputDir:  outputDir,
				Overlay:    overlay,
				Validate:   validate,
				Verbose:    verbose,
			})
			
			return generator.Generate(GeneratorOptions{
				InputFiles: inputFiles,
				OutputDir:  outputDir,
				Overlay:    overlay,
				Validate:   validate,
				Verbose:    verbose,
			})
		},
	}
	
	cmd.Flags().StringSliceP("file", "f", []string{}, "input file or directory (required)")
	cmd.Flags().StringVarP(&outputDir, "output", "o", "", "output directory (default: stdout)")
	cmd.Flags().StringVar(&overlay, "overlay", "", "kustomize overlay path (directory or kustomization.yaml file)")
	cmd.Flags().BoolVar(&validate, "validate", true, "validate instances before hydration")
	cmd.MarkFlagRequired("file")
	
	return cmd
}

// BuildValidateCommand builds the validate command
func BuildValidateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate -f <file|directory>",
		Short: "Validate abstraction instances",
		Long: `Validate abstraction instances against their CRD schemas.

This command checks that instances conform to the defined schemas
without generating any resources.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			inputFiles, err := cmd.Flags().GetStringSlice("file")
			if err != nil || len(inputFiles) == 0 {
				return fmt.Errorf("--file/-f is required")
			}
			
			verbose, _ := cmd.Flags().GetBool("verbose")
			
			validator := NewValidator(ValidatorOptions{
				InputFiles: inputFiles,
				Verbose:    verbose,
			})
			
			return validator.Validate()
		},
	}
	
	cmd.Flags().StringSliceP("file", "f", []string{}, "input file or directory (required)")
	cmd.MarkFlagRequired("file")
	
	return cmd
}

// BuildApplyCommand builds the apply command
func BuildApplyCommand() *cobra.Command {
	var (
		overlay string
		dryRun  bool
	)
	
	cmd := &cobra.Command{
		Use:   "apply -f <file|directory>",
		Short: "Generate and apply resources to cluster",
		Long: `Generate Kubernetes resources and apply them to the cluster.

This command combines generation and kubectl apply in one step.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			inputFiles, err := cmd.Flags().GetStringSlice("file")
			if err != nil || len(inputFiles) == 0 {
				return fmt.Errorf("--file/-f is required")
			}
			
			verbose, _ := cmd.Flags().GetBool("verbose")
			
			applier := NewApplier(ApplierOptions{
				InputFiles: inputFiles,
				Overlay:    overlay,
				DryRun:     dryRun,
				Verbose:    verbose,
			})
			
			return applier.Apply()
		},
	}
	
	cmd.Flags().StringSliceP("file", "f", []string{}, "input file or directory (required)")
	cmd.Flags().StringVar(&overlay, "overlay", "", "kustomize overlay path (directory or kustomization.yaml file)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "perform a dry run")
	cmd.MarkFlagRequired("file")
	
	return cmd
}

// ValidatorOptions contains options for validation
type ValidatorOptions struct {
	InputFiles []string
	Verbose    bool
}

// Validator handles validation
type Validator struct {
	opts ValidatorOptions
}

// NewValidator creates a new validator
func NewValidator(opts ValidatorOptions) *Validator {
	return &Validator{opts: opts}
}

// Validate validates input files
func (v *Validator) Validate() error {
	validator := NewGenerator(GeneratorOptions{
		Validate: true,
		Verbose:  v.opts.Verbose,
	})
	
	// Use the generator with validation only
	for _, inputFile := range v.opts.InputFiles {
		if v.opts.Verbose {
			fmt.Printf("Validating: %s\n", inputFile)
		}
		
		_, err := validator.processFile(inputFile, GeneratorOptions{
			Validate: true,
			Verbose:  v.opts.Verbose,
		})
		
		if err != nil {
			fmt.Fprintf(os.Stderr, "✗ %s: %v\n", inputFile, err)
			return err
		}
		
		fmt.Printf("✓ %s\n", inputFile)
	}
	
	fmt.Println("\nAll files validated successfully!")
	return nil
}

// ApplierOptions contains options for applying resources
type ApplierOptions struct {
	InputFiles []string
	Overlay    string
	DryRun     bool
	Verbose    bool
}

// Applier handles resource application
type Applier struct {
	opts ApplierOptions
}

// NewApplier creates a new applier
func NewApplier(opts ApplierOptions) *Applier {
	return &Applier{opts: opts}
}

// Apply generates and applies resources
func (a *Applier) Apply() error {
	// For now, this is a placeholder
	// In a full implementation, this would:
	// 1. Generate resources
	// 2. Apply overlays
	// 3. Use kubectl or client-go to apply to cluster
	
	fmt.Println("Apply functionality coming soon!")
	fmt.Println("For now, use: generate -f <file> -o output/ | kubectl apply -f -")
	
	return nil
}

