package testing

import (
	"fmt"
)

// Expectation represents an expectation for generated resources
type Expectation struct {
	Kind      string
	Count     int
	Name      string
	Namespace string
	Labels    map[string]string
	Checks    []ResourceCheck
}

// ResourceCheck is a function that checks a resource
type ResourceCheck func(resource map[string]interface{}) error

// ExpectResource creates a resource expectation
func ExpectResource(kind string, count int) Expectation {
	return Expectation{
		Kind:   kind,
		Count:  count,
		Labels: make(map[string]string),
		Checks: []ResourceCheck{},
	}
}

// WithName adds a name matcher
func (e Expectation) WithName(name string) Expectation {
	e.Name = name
	return e
}

// WithNamespace adds a namespace matcher
func (e Expectation) WithNamespace(ns string) Expectation {
	e.Namespace = ns
	return e
}

// WithLabel adds a label matcher
func (e Expectation) WithLabel(key, value string) Expectation {
	if e.Labels == nil {
		e.Labels = make(map[string]string)
	}
	e.Labels[key] = value
	return e
}

// WithCheck adds a custom check function
func (e Expectation) WithCheck(check ResourceCheck) Expectation {
	e.Checks = append(e.Checks, check)
	return e
}

// Validate validates resources against this expectation
func (e Expectation) Validate(resources []map[string]interface{}) error {
	// Find matching resources
	matching := []map[string]interface{}{}

	for _, resource := range resources {
		if e.matches(resource) {
			matching = append(matching, resource)
		}
	}

	// Check count
	if e.Count > 0 && len(matching) != e.Count {
		return fmt.Errorf("expected %d resources of kind %s, got %d", e.Count, e.Kind, len(matching))
	}

	// Run custom checks on matching resources
	for i, resource := range matching {
		for j, check := range e.Checks {
			if err := check(resource); err != nil {
				return fmt.Errorf("check %d failed on resource %d: %w", j, i, err)
			}
		}
	}

	return nil
}

// matches checks if a resource matches this expectation
func (e Expectation) matches(resource map[string]interface{}) bool {
	// Check kind
	if e.Kind != "" {
		kind, ok := resource["kind"].(string)
		if !ok || kind != e.Kind {
			return false
		}
	}

	// Check metadata
	metadata, ok := resource["metadata"].(map[string]interface{})
	if !ok {
		return e.Name == "" && e.Namespace == "" && len(e.Labels) == 0
	}

	// Check name
	if e.Name != "" {
		name, ok := metadata["name"].(string)
		if !ok || name != e.Name {
			return false
		}
	}

	// Check namespace
	if e.Namespace != "" {
		namespace, ok := metadata["namespace"].(string)
		if !ok || namespace != e.Namespace {
			return false
		}
	}

	// Check labels
	if len(e.Labels) > 0 {
		labels, ok := metadata["labels"].(map[string]interface{})
		if !ok {
			return false
		}

		for key, expectedValue := range e.Labels {
			actualValue, ok := labels[key]
			if !ok || actualValue != expectedValue {
				return false
			}
		}
	}

	return true
}

// HasField checks if a resource has a specific field path
func HasField(path ...string) ResourceCheck {
	return func(resource map[string]interface{}) error {
		current := resource
		for i, key := range path {
			val, ok := current[key]
			if !ok {
				return fmt.Errorf("field not found: %s", key)
			}

			if i < len(path)-1 {
				current, ok = val.(map[string]interface{})
				if !ok {
					return fmt.Errorf("field %s is not a map", key)
				}
			}
		}
		return nil
	}
}

// FieldEquals checks if a field equals a specific value
func FieldEquals(value interface{}, path ...string) ResourceCheck {
	return func(resource map[string]interface{}) error {
		current := resource
		for i, key := range path {
			val, ok := current[key]
			if !ok {
				return fmt.Errorf("field not found: %s", key)
			}

			if i == len(path)-1 {
				if val != value {
					return fmt.Errorf("field %s: expected %v, got %v", key, value, val)
				}
				return nil
			}

			current, ok = val.(map[string]interface{})
			if !ok {
				return fmt.Errorf("field %s is not a map", key)
			}
		}
		return nil
	}
}

// HasLabel checks if a resource has a specific label
func HasLabel(key, value string) ResourceCheck {
	return func(resource map[string]interface{}) error {
		metadata, ok := resource["metadata"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("metadata not found")
		}

		labels, ok := metadata["labels"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("labels not found")
		}

		actualValue, ok := labels[key]
		if !ok {
			return fmt.Errorf("label %s not found", key)
		}

		if actualValue != value {
			return fmt.Errorf("label %s: expected %v, got %v", key, value, actualValue)
		}

		return nil
	}
}

// HasAnnotation checks if a resource has a specific annotation
func HasAnnotation(key, value string) ResourceCheck {
	return func(resource map[string]interface{}) error {
		metadata, ok := resource["metadata"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("metadata not found")
		}

		annotations, ok := metadata["annotations"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("annotations not found")
		}

		actualValue, ok := annotations[key]
		if !ok {
			return fmt.Errorf("annotation %s not found", key)
		}

		if actualValue != value {
			return fmt.Errorf("annotation %s: expected %v, got %v", key, value, actualValue)
		}

		return nil
	}
}

