package cli

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/yourusername/krm-sdk/pkg/hydrator"
	"github.com/yourusername/krm-sdk/pkg/overlay"
	"github.com/yourusername/krm-sdk/pkg/validation"
	"sigs.k8s.io/yaml"
)

// Generator handles resource generation
type Generator struct {
	validator *validation.Validator
	hydrator  *hydrator.Hydrator
	verbose   bool
}

// GeneratorOptions contains options for the generator
type GeneratorOptions struct {
	InputFiles  []string
	OutputDir   string
	Overlay     string
	Validate    bool
	DryRun      bool
	Verbose     bool
}

// NewGenerator creates a new generator
func NewGenerator(opts GeneratorOptions) *Generator {
	return &Generator{
		validator: validation.NewValidator("config/crd", opts.Verbose),
		hydrator:  hydrator.NewHydrator("", opts.Verbose),
		verbose:   opts.Verbose,
	}
}

// Generate processes input files and generates K8s resources
func (g *Generator) Generate(opts GeneratorOptions) error {
	// Load validation schemas if validation is enabled
	if opts.Validate {
		if g.verbose {
			fmt.Println("Loading validation schemas...")
		}
		if err := g.validator.LoadSchemas(); err != nil {
			fmt.Printf("Warning: failed to load schemas: %v\n", err)
		}
	}
	
	// Process each input file
	var allResources []map[string]interface{}
	
	for _, inputPath := range opts.InputFiles {
		if g.verbose {
			fmt.Printf("Processing: %s\n", inputPath)
		}
		
		resources, err := g.processFile(inputPath, opts)
		if err != nil {
			return fmt.Errorf("failed to process %s: %w", inputPath, err)
		}
		
		allResources = append(allResources, resources...)
	}
	
	// Apply kustomize overlay if specified
	if opts.Overlay != "" {
		if g.verbose {
			fmt.Printf("Applying overlay: %s\n", opts.Overlay)
		}
		
		kustomizer := overlay.NewKustomizeEngine("base", "overlays", opts.Verbose)
		
		// Write base resources
		if err := kustomizer.WriteBase(allResources); err != nil {
			return fmt.Errorf("failed to write base: %w", err)
		}
		
		// Apply kustomize overlay
		kustomized, err := kustomizer.ApplyOverlay(opts.Overlay)
		if err != nil {
			// Clean up base directory
			kustomizer.Cleanup()
			return fmt.Errorf("failed to apply overlay '%s': %w", opts.Overlay, err)
		}
		
		// Clean up base directory
		defer kustomizer.Cleanup()
		
		allResources = kustomized
		
		if g.verbose {
			fmt.Printf("✓ Applied overlay: %s\n", opts.Overlay)
		}
	}
	
	// Output resources
	if opts.OutputDir != "" {
		return g.writeResources(allResources, opts.OutputDir)
	}
	
	return g.printResources(allResources, os.Stdout)
}

// processFile processes a single input file
func (g *Generator) processFile(path string, opts GeneratorOptions) ([]map[string]interface{}, error) {
	// Check if path is a directory
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	
	if info.IsDir() {
		return g.processDirectory(path, opts)
	}
	
	// Read file
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	
	// Parse YAML
	var instance map[string]interface{}
	if err := yaml.Unmarshal(data, &instance); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}
	
	// Validate if requested
	if opts.Validate {
		result, err := g.validator.Validate(instance)
		if err != nil {
			return nil, fmt.Errorf("validation error: %w", err)
		}
		
		if !result.Valid {
			return nil, fmt.Errorf("validation failed:\n  %s", strings.Join(result.Errors, "\n  "))
		}
		
		if g.verbose {
			fmt.Println("✓ Validation passed")
		}
	}
	
	// Hydrate
	hydrateResult, err := g.hydrator.Hydrate(instance)
	if err != nil {
		return nil, fmt.Errorf("hydration error: %w", err)
	}
	
	if len(hydrateResult.Errors) > 0 {
		for _, err := range hydrateResult.Errors {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}
	}
	
	return hydrateResult.Resources, nil
}

// processDirectory processes all YAML files in a directory
func (g *Generator) processDirectory(dirPath string, opts GeneratorOptions) ([]map[string]interface{}, error) {
	var allResources []map[string]interface{}
	
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}
	
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		
		if !strings.HasSuffix(file.Name(), ".yaml") && !strings.HasSuffix(file.Name(), ".yml") {
			continue
		}
		
		path := filepath.Join(dirPath, file.Name())
		resources, err := g.processFile(path, opts)
		if err != nil {
			return nil, err
		}
		
		allResources = append(allResources, resources...)
	}
	
	return allResources, nil
}

// writeResources writes resources to files in the output directory
func (g *Generator) writeResources(resources []map[string]interface{}, outputDir string) error {
	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	for i, resource := range resources {
		// Generate filename from resource metadata
		filename := g.generateFilename(resource, i)
		path := filepath.Join(outputDir, filename)
		
		if g.verbose {
			fmt.Printf("Writing: %s\n", path)
		}
		
		// Marshal to YAML
		data, err := yaml.Marshal(resource)
		if err != nil {
			return fmt.Errorf("failed to marshal resource: %w", err)
		}
		
		// Write file
		if err := ioutil.WriteFile(path, data, 0644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
	}
	
	fmt.Printf("\n✓ Generated %d resources in %s\n", len(resources), outputDir)
	return nil
}

// printResources prints resources to stdout
func (g *Generator) printResources(resources []map[string]interface{}, w io.Writer) error {
	for i, resource := range resources {
		if i > 0 {
			fmt.Fprintln(w, "---")
		}
		
		data, err := yaml.Marshal(resource)
		if err != nil {
			return fmt.Errorf("failed to marshal resource: %w", err)
		}
		
		fmt.Fprint(w, string(data))
	}
	
	return nil
}

// generateFilename generates a filename for a resource
func (g *Generator) generateFilename(resource map[string]interface{}, index int) string {
	kind := "resource"
	name := fmt.Sprintf("%d", index)
	
	if k, ok := resource["kind"].(string); ok {
		kind = strings.ToLower(k)
	}
	
	if metadata, ok := resource["metadata"].(map[string]interface{}); ok {
		if n, ok := metadata["name"].(string); ok {
			name = n
		}
	}
	
	return fmt.Sprintf("%s-%s.yaml", kind, name)
}

