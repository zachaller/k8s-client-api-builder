package hydrator

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/zachaller/k8s-client-api-builder/pkg/ast"
	"sigs.k8s.io/yaml"
)

// Hydrator handles the hydration of abstractions into K8s resources
type Hydrator struct {
	templateDir string
	verbose     bool
}

// NewHydrator creates a new hydrator
func NewHydrator(templateDir string, verbose bool) *Hydrator {
	return &Hydrator{
		templateDir: templateDir,
		verbose:     verbose,
	}
}

// Template represents a hydration template
type Template struct {
	Resources interface{} `yaml:"resources"` // Can be []interface{} or map with conditionals
}

// HydrateResult contains the hydrated resources
type HydrateResult struct {
	Resources []map[string]interface{}
	Errors    []error
}

// Hydrate processes an abstraction instance and generates K8s resources
// Uses AST-based parsing and evaluation with two-pass processing for cross-resource references
func (h *Hydrator) Hydrate(instance map[string]interface{}) (*HydrateResult, error) {
	// Extract kind from instance
	kind, ok := instance["kind"].(string)
	if !ok {
		return nil, fmt.Errorf("instance missing 'kind' field")
	}

	// Extract version from apiVersion
	apiVersion, ok := instance["apiVersion"].(string)
	if !ok {
		return nil, fmt.Errorf("instance missing 'apiVersion' field")
	}

	parts := strings.Split(apiVersion, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid apiVersion format: %s", apiVersion)
	}
	version := parts[1]

	// Load template
	templatePath := h.findTemplate(kind, version)
	if templatePath == "" {
		return nil, fmt.Errorf("template not found for kind '%s' version '%s'", kind, version)
	}

	if h.verbose {
		fmt.Printf("Loading template: %s\n", templatePath)
	}

	template, err := h.loadTemplate(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load template: %w", err)
	}

	// Parse template YAML to AST
	astRoot, err := ast.ParseTemplate(template.Resources)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template to AST: %w", err)
	}

	if h.verbose {
		printer := ast.NewPrinter()
		astStr, _ := printer.Print(astRoot)
		fmt.Printf("Template AST:\n%s\n", astStr)
	}

	// Pass 1: Evaluate AST to generate resources (without resolving resource references)
	evaluator := ast.NewEvaluator(instance)
	pass1Resources, err := evaluator.Evaluate(astRoot)
	if err != nil {
		return nil, fmt.Errorf("pass 1 evaluation failed: %w", err)
	}

	// Pass 2: Resolve cross-resource references
	finalResources, errors := h.hydratePass2AST(pass1Resources, instance)

	return &HydrateResult{
		Resources: finalResources,
		Errors:    errors,
	}, nil
}

// hydratePass2AST resolves cross-resource references using AST evaluator
func (h *Hydrator) hydratePass2AST(resources []map[string]interface{}, instance map[string]interface{}) ([]map[string]interface{}, []error) {
	// Create new evaluator with instance data
	evaluator := ast.NewEvaluator(instance)

	// Register all resources
	for _, resource := range resources {
		if err := registerResourceInEvaluator(evaluator, resource); err != nil {
			// Skip resources that can't be registered
			continue
		}
	}

	// Build dependency graph for circular reference detection
	depGraph, err := h.buildDependencyGraph(resources)
	if err != nil {
		return nil, []error{err}
	}

	// Check for circular references
	if cycles := detectCircularReferences(depGraph); len(cycles) > 0 {
		return nil, []error{fmt.Errorf("circular resource references detected: %v", cycles)}
	}

	// Process each resource again to resolve references
	finalResources := []map[string]interface{}{}
	errors := []error{}

	for i, resource := range resources {
		if h.verbose {
			fmt.Printf("Pass 2: Resolving references in resource %d/%d\n", i+1, len(resources))
		}

		resolved, err := h.resolveResourceReferencesAST(resource, evaluator, instance)
		if err != nil {
			errors = append(errors, fmt.Errorf("resource %d: %w", i, err))
			// Still include the resource even if resolution fails
			finalResources = append(finalResources, resource)
			continue
		}

		// Type assert resolved value
		resolvedResource, ok := resolved.(map[string]interface{})
		if !ok {
			errors = append(errors, fmt.Errorf("resource %d: resolved value is not a map", i))
			finalResources = append(finalResources, resource)
			continue
		}

		finalResources = append(finalResources, resolvedResource)
	}

	return finalResources, errors
}

// resolveResourceReferencesAST resolves resource references in a resource using the AST evaluator
func (h *Hydrator) resolveResourceReferencesAST(resource map[string]interface{}, evaluator *ast.Evaluator, context map[string]interface{}) (interface{}, error) {
	result := make(map[string]interface{})

	for key, value := range resource {
		processedValue, err := h.resolveValueReferencesAST(value, evaluator, context)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve field %s: %w", key, err)
		}
		result[key] = processedValue
	}

	return result, nil
}

// resolveValueReferencesAST resolves references in any value type
func (h *Hydrator) resolveValueReferencesAST(value interface{}, evaluator *ast.Evaluator, context map[string]interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		// Check if string contains resource() reference
		if strings.Contains(v, "resource(") {
			// Use DSL evaluator to resolve
			dslEval := evaluator.GetDSLEvaluator()
			return dslEval.EvaluateString(v)
		}
		return v, nil

	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, val := range v {
			processedVal, err := h.resolveValueReferencesAST(val, evaluator, context)
			if err != nil {
				return nil, err
			}
			result[key] = processedVal
		}
		return result, nil

	case []interface{}:
		result := make([]interface{}, 0, len(v))
		for _, item := range v {
			processedItem, err := h.resolveValueReferencesAST(item, evaluator, context)
			if err != nil {
				return nil, err
			}
			result = append(result, processedItem)
		}
		return result, nil

	default:
		return v, nil
	}
}

// registerResourceInEvaluator registers a resource in the evaluator's resource registry
func registerResourceInEvaluator(evaluator *ast.Evaluator, resource map[string]interface{}) error {
	apiVersion, ok := resource["apiVersion"].(string)
	if !ok {
		return fmt.Errorf("resource missing apiVersion")
	}

	kind, ok := resource["kind"].(string)
	if !ok {
		return fmt.Errorf("resource missing kind")
	}

	metadata, ok := resource["metadata"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("resource missing metadata")
	}

	name, ok := metadata["name"].(string)
	if !ok {
		return fmt.Errorf("resource missing metadata.name")
	}

	evaluator.RegisterResource(apiVersion, kind, name, resource)
	return nil
}

// loadTemplate loads a template file
func (h *Hydrator) loadTemplate(path string) (*Template, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var template Template
	if err := yaml.Unmarshal(data, &template); err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	return &template, nil
}

// findTemplate finds the template file for a given kind and version
func (h *Hydrator) findTemplate(kind, version string) string {
	// Look for template in the template directory
	// Expected naming: <kind_lower>_<version>.yaml or <kind_lower>_template.yaml
	kindLower := strings.ToLower(kind)

	// Try with version first
	path := filepath.Join(h.templateDir, fmt.Sprintf("%s_%s.yaml", kindLower, version))
	if _, err := os.Stat(path); err == nil {
		return path
	}

	// Try without version
	path = filepath.Join(h.templateDir, fmt.Sprintf("%s_template.yaml", kindLower))
	if _, err := os.Stat(path); err == nil {
		return path
	}

	// Try exact kind name
	path = filepath.Join(h.templateDir, fmt.Sprintf("%s.yaml", kindLower))
	if _, err := os.Stat(path); err == nil {
		return path
	}

	return ""
}
