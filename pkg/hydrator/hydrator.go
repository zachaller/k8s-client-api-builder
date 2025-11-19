package hydrator

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/yourusername/krm-sdk/pkg/dsl"
	"gopkg.in/yaml.v3"
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
	Resources []map[string]interface{} `yaml:"resources"`
}

// HydrateResult contains the hydrated resources
type HydrateResult struct {
	Resources []map[string]interface{}
	Errors    []error
}

// Hydrate processes an abstraction instance and generates K8s resources
func (h *Hydrator) Hydrate(instance map[string]interface{}) (*HydrateResult, error) {
	result := &HydrateResult{
		Resources: []map[string]interface{}{},
		Errors:    []error{},
	}
	
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
	
	// Create evaluator with instance data
	evaluator := dsl.NewEvaluator(instance)
	
	// Process each resource in the template
	for i, resourceTemplate := range template.Resources {
		if h.verbose {
			fmt.Printf("Processing resource %d/%d\n", i+1, len(template.Resources))
		}
		
		resource, err := h.processResource(resourceTemplate, evaluator, instance)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("resource %d: %w", i, err))
			continue
		}
		
		result.Resources = append(result.Resources, resource)
	}
	
	return result, nil
}

// processResource processes a single resource template
func (h *Hydrator) processResource(template map[string]interface{}, evaluator *dsl.Evaluator, context map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	
	for key, value := range template {
		// Check for control structures
		if strings.HasPrefix(key, "$if(") {
			// Handle conditional
			processed, err := h.processConditional(key, value, evaluator, context)
			if err != nil {
				return nil, err
			}
			if processed != nil {
				// Merge conditional results into parent
				for k, v := range processed {
					result[k] = v
				}
			}
			continue
		}
		
		if strings.HasPrefix(key, "$for(") {
			// Handle loop
			processed, err := h.processLoop(key, value, evaluator, context)
			if err != nil {
				return nil, err
			}
			return nil, fmt.Errorf("$for at resource level not yet supported")
		}
		
		// Process the value
		processedValue, err := h.processValue(value, evaluator, context)
		if err != nil {
			return nil, fmt.Errorf("failed to process key '%s': %w", key, err)
		}
		
		result[key] = processedValue
	}
	
	return result, nil
}

// processValue processes a value, which can be a string, map, slice, or primitive
func (h *Hydrator) processValue(value interface{}, evaluator *dsl.Evaluator, context map[string]interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		// Evaluate string with variable substitutions
		return evaluator.EvaluateString(v)
		
	case map[string]interface{}:
		// Recursively process map
		result := make(map[string]interface{})
		for key, val := range v {
			// Check for control structures
			if strings.HasPrefix(key, "$if(") {
				processed, err := h.processConditional(key, val, evaluator, context)
				if err != nil {
					return nil, err
				}
				if processed != nil {
					for k, pv := range processed {
						result[k] = pv
					}
				}
				continue
			}
			
			if strings.HasPrefix(key, "$for(") {
				processed, err := h.processLoop(key, val, evaluator, context)
				if err != nil {
					return nil, err
				}
				// For loops in maps should return an array
				return processed, nil
			}
			
			processedVal, err := h.processValue(val, evaluator, context)
			if err != nil {
				return nil, err
			}
			result[key] = processedVal
		}
		return result, nil
		
	case []interface{}:
		// Recursively process slice
		result := make([]interface{}, 0, len(v))
		for _, item := range v {
			processedItem, err := h.processValue(item, evaluator, context)
			if err != nil {
				return nil, err
			}
			result = append(result, processedItem)
		}
		return result, nil
		
	default:
		// Return primitive values as-is
		return v, nil
	}
}

// processConditional handles $if(...): constructs
func (h *Hydrator) processConditional(key string, value interface{}, evaluator *dsl.Evaluator, context map[string]interface{}) (map[string]interface{}, error) {
	// Extract condition from key: $if(condition):
	if !strings.HasPrefix(key, "$if(") || !strings.HasSuffix(key, "):") {
		return nil, fmt.Errorf("invalid conditional syntax: %s", key)
	}
	
	condition := key[4 : len(key)-2]
	
	// Parse and evaluate condition
	expr, err := dsl.ParseExpression(condition)
	if err != nil {
		return nil, fmt.Errorf("failed to parse condition '%s': %w", condition, err)
	}
	
	result, err := evaluator.Evaluate(expr)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate condition '%s': %w", condition, err)
	}
	
	// Check if condition is true
	isTrue := false
	switch v := result.(type) {
	case bool:
		isTrue = v
	case string:
		isTrue = v != "" && v != "false"
	case int, int32, int64:
		isTrue = v != 0
	default:
		isTrue = result != nil
	}
	
	if !isTrue {
		return nil, nil
	}
	
	// Process the conditional block
	processedValue, err := h.processValue(value, evaluator, context)
	if err != nil {
		return nil, err
	}
	
	// Return as map
	if m, ok := processedValue.(map[string]interface{}); ok {
		return m, nil
	}
	
	return nil, fmt.Errorf("conditional block must be a map")
}

// processLoop handles $for(...): constructs
func (h *Hydrator) processLoop(key string, value interface{}, evaluator *dsl.Evaluator, context map[string]interface{}) ([]interface{}, error) {
	// Extract loop expression from key: $for(item in .path):
	if !strings.HasPrefix(key, "$for(") || !strings.HasSuffix(key, "):") {
		return nil, fmt.Errorf("invalid loop syntax: %s", key)
	}
	
	loopExpr := key[5 : len(key)-2]
	
	// Parse loop expression
	varName, iterPath, err := dsl.ParseForLoop(loopExpr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse loop expression: %w", err)
	}
	
	// Evaluate the iteration path
	pathExpr, err := dsl.ParseExpression(iterPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse iteration path: %w", err)
	}
	
	items, err := evaluator.Evaluate(pathExpr)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate iteration path: %w", err)
	}
	
	// Ensure items is a slice
	itemsSlice, ok := items.([]interface{})
	if !ok {
		return nil, fmt.Errorf("iteration path must evaluate to an array")
	}
	
	// Process each item
	results := make([]interface{}, 0, len(itemsSlice))
	for _, item := range itemsSlice {
		// Create new context with loop variable
		loopContext := make(map[string]interface{})
		for k, v := range context {
			loopContext[k] = v
		}
		loopContext[varName] = item
		
		// Create new evaluator with loop context
		loopEvaluator := dsl.NewEvaluator(loopContext)
		
		// Process the loop body
		processedItem, err := h.processValue(value, loopEvaluator, loopContext)
		if err != nil {
			return nil, fmt.Errorf("failed to process loop item: %w", err)
		}
		
		results = append(results, processedItem)
	}
	
	return results, nil
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
	// Convert kind to snake_case for filename
	snakeName := toSnakeCase(kind)
	
	// Try api/{version}/{kind}_template.yaml
	path := filepath.Join("api", version, snakeName+"_template.yaml")
	if _, err := os.Stat(path); err == nil {
		return path
	}
	
	// Try in template directory if specified
	if h.templateDir != "" {
		path = filepath.Join(h.templateDir, version, snakeName+"_template.yaml")
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	
	return ""
}

// toSnakeCase converts a string to snake_case
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

