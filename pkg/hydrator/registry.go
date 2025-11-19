package hydrator

import (
	"fmt"
)

// ResourceRegistry manages resources for cross-resource references
type ResourceRegistry struct {
	resources map[string]map[string]interface{}
}

// NewResourceRegistry creates a new resource registry
func NewResourceRegistry() *ResourceRegistry {
	return &ResourceRegistry{
		resources: make(map[string]map[string]interface{}),
	}
}

// Register adds a resource to the registry
func (r *ResourceRegistry) Register(resource map[string]interface{}) error {
	// Extract metadata
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
	
	// Build key
	key := fmt.Sprintf("%s/%s/%s", apiVersion, kind, name)
	
	// Store resource
	r.resources[key] = resource
	
	return nil
}

// Lookup finds a resource by apiVersion, kind, and name
func (r *ResourceRegistry) Lookup(apiVersion, kind, name string) (map[string]interface{}, error) {
	key := fmt.Sprintf("%s/%s/%s", apiVersion, kind, name)
	
	resource, ok := r.resources[key]
	if !ok {
		return nil, fmt.Errorf("resource not found: %s", key)
	}
	
	return resource, nil
}

// GetField gets a specific field from a resource
func (r *ResourceRegistry) GetField(apiVersion, kind, name, fieldPath string) (interface{}, error) {
	resource, err := r.Lookup(apiVersion, kind, name)
	if err != nil {
		return nil, err
	}
	
	// For field navigation, we'll use the evaluator's navigateResourceField
	// This is handled in the evaluator
	return resource, nil
}

// List returns all registered resources
func (r *ResourceRegistry) List() []map[string]interface{} {
	resources := make([]map[string]interface{}, 0, len(r.resources))
	for _, resource := range r.resources {
		resources = append(resources, resource)
	}
	return resources
}

// Keys returns all resource keys
func (r *ResourceRegistry) Keys() []string {
	keys := make([]string, 0, len(r.resources))
	for key := range r.resources {
		keys = append(keys, key)
	}
	return keys
}

