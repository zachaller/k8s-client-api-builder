package overlay

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/yaml"
)

// KustomizeEngine handles kustomize integration
type KustomizeEngine struct {
	baseDir    string
	overlayDir string
	verbose    bool
	fs         filesys.FileSystem
}

// NewKustomizeEngine creates a new kustomize engine
func NewKustomizeEngine(baseDir, overlayDir string, verbose bool) *KustomizeEngine {
	return &KustomizeEngine{
		baseDir:    baseDir,
		overlayDir: overlayDir,
		verbose:    verbose,
		fs:         filesys.MakeFsOnDisk(),
	}
}

// WriteBase writes generated resources to base/ with kustomization.yaml
func (k *KustomizeEngine) WriteBase(resources []map[string]interface{}) error {
	// Create base directory
	if err := os.MkdirAll(k.baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create base directory: %w", err)
	}

	if k.verbose {
		fmt.Printf("Writing %d resources to %s/\n", len(resources), k.baseDir)
	}

	// Write each resource as a separate file
	var resourceFiles []string
	for i, resource := range resources {
		filename := k.generateFilename(resource, i)
		path := filepath.Join(k.baseDir, filename)

		if k.verbose {
			fmt.Printf("  Writing: %s\n", filename)
		}

		data, err := yaml.Marshal(resource)
		if err != nil {
			return fmt.Errorf("failed to marshal resource: %w", err)
		}

		if err := os.WriteFile(path, data, 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", path, err)
		}

		resourceFiles = append(resourceFiles, filename)
	}

	// Create kustomization.yaml
	if err := k.createBaseKustomization(resourceFiles); err != nil {
		return fmt.Errorf("failed to create base kustomization: %w", err)
	}

	if k.verbose {
		fmt.Printf("✓ Created base kustomization\n")
	}

	return nil
}

// ApplyOverlay runs kustomize build on the specified overlay path
// overlayPath can be:
// - A directory containing kustomization.yaml (e.g., "overlays/prod")
// - A kustomization.yaml file directly (e.g., "overlays/prod/kustomization.yaml")
func (k *KustomizeEngine) ApplyOverlay(overlayPath string) ([]map[string]interface{}, error) {
	resolvedPath, err := k.resolveOverlayPath(overlayPath)
	if err != nil {
		return nil, err
	}

	if k.verbose {
		fmt.Printf("Running kustomize build on %s\n", resolvedPath)
	}

	// Build with kustomize
	return k.Build(resolvedPath)
}

// resolveOverlayPath resolves the overlay path to a directory containing kustomization.yaml
func (k *KustomizeEngine) resolveOverlayPath(overlayPath string) (string, error) {
	// Check if it's a direct path to kustomization.yaml file
	if filepath.Base(overlayPath) == "kustomization.yaml" {
		if _, err := os.Stat(overlayPath); err == nil {
			// Return the directory containing the file
			return filepath.Dir(overlayPath), nil
		}
		return "", fmt.Errorf("kustomization.yaml file not found: %s", overlayPath)
	}

	// Check if it's a directory with kustomization.yaml
	info, err := os.Stat(overlayPath)
	if err != nil {
		return "", fmt.Errorf("overlay path not found: %s", overlayPath)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("overlay path is not a directory or kustomization.yaml file: %s", overlayPath)
	}

	// Verify directory contains kustomization.yaml
	kustomizationPath := filepath.Join(overlayPath, "kustomization.yaml")
	if _, err := os.Stat(kustomizationPath); os.IsNotExist(err) {
		return "", fmt.Errorf("directory does not contain kustomization.yaml: %s", overlayPath)
	}

	return overlayPath, nil
}


// Build runs kustomize build and returns resources
func (k *KustomizeEngine) Build(overlayPath string) ([]map[string]interface{}, error) {
	// Create kustomize options
	opts := krusty.MakeDefaultOptions()
	opts.LoadRestrictions = types.LoadRestrictionsNone

	// Create kustomizer
	kustomizer := krusty.MakeKustomizer(opts)

	// Run kustomize build
	resMap, err := kustomizer.Run(k.fs, overlayPath)
	if err != nil {
		return nil, fmt.Errorf("kustomize build failed: %w", err)
	}

	// Convert ResMap to []map[string]interface{}
	resources, err := k.resMapToResources(resMap)
	if err != nil {
		return nil, fmt.Errorf("failed to convert resources: %w", err)
	}

	if k.verbose {
		fmt.Printf("✓ Kustomize build completed: %d resources\n", len(resources))
	}

	return resources, nil
}

// createBaseKustomization creates a kustomization.yaml in the base directory
func (k *KustomizeEngine) createBaseKustomization(resourceFiles []string) error {
	kustomization := types.Kustomization{
		TypeMeta: types.TypeMeta{
			APIVersion: types.KustomizationVersion,
			Kind:       types.KustomizationKind,
		},
		Resources: resourceFiles,
	}

	// Marshal to YAML
	data, err := yaml.Marshal(kustomization)
	if err != nil {
		return fmt.Errorf("failed to marshal kustomization: %w", err)
	}

	// Write kustomization.yaml
	path := filepath.Join(k.baseDir, "kustomization.yaml")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write kustomization.yaml: %w", err)
	}

	return nil
}

// resMapToResources converts kustomize ResMap to []map[string]interface{}
func (k *KustomizeEngine) resMapToResources(resMap interface{}) ([]map[string]interface{}, error) {
	// Get YAML output from ResMap
	yamlBytes, err := resMap.(interface{ AsYaml() ([]byte, error) }).AsYaml()
	if err != nil {
		return nil, fmt.Errorf("failed to convert ResMap to YAML: %w", err)
	}

	// Split YAML documents
	docs := strings.Split(string(yamlBytes), "\n---\n")
	
	resources := make([]map[string]interface{}, 0, len(docs))
	for _, doc := range docs {
		doc = strings.TrimSpace(doc)
		if doc == "" || doc == "---" {
			continue
		}

		var resource map[string]interface{}
		if err := yaml.Unmarshal([]byte(doc), &resource); err != nil {
			return nil, fmt.Errorf("failed to unmarshal resource: %w", err)
		}

		resources = append(resources, resource)
	}

	return resources, nil
}

// generateFilename generates a filename for a resource
func (k *KustomizeEngine) generateFilename(resource map[string]interface{}, index int) string {
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

// Cleanup removes the base directory
func (k *KustomizeEngine) Cleanup() error {
	if k.baseDir != "" && k.baseDir != "." && k.baseDir != "/" {
		return os.RemoveAll(k.baseDir)
	}
	return nil
}

