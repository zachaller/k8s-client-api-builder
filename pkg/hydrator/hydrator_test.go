package hydrator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zachaller/k8s-client-api-builder/pkg/dsl"
)

// NewEvaluator is a helper to create evaluators for testing
func NewEvaluator(data interface{}) *dsl.Evaluator {
	return dsl.NewEvaluator(data)
}

func TestHydrate(t *testing.T) {
	// Create temp directory for templates
	tempDir, err := os.MkdirTemp("", "hydrator-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create template directory structure
	apiDir := filepath.Join(tempDir, "api", "v1alpha1")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("failed to create api dir: %v", err)
	}

	// Create a simple template
	template := `resources:
  - apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: $(.metadata.name)
      namespace: $(.metadata.namespace)
    spec:
      replicas: $(.spec.replicas)
`
	templatePath := filepath.Join(apiDir, "test_service_template.yaml")
	if err := os.WriteFile(templatePath, []byte(template), 0644); err != nil {
		t.Fatalf("failed to write template: %v", err)
	}

	// Change to temp directory for test
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	// Create hydrator
	h := NewHydrator("", false)

	// Test instance
	instance := map[string]interface{}{
		"apiVersion": "test.example.com/v1alpha1",
		"kind":       "TestService",
		"metadata": map[string]interface{}{
			"name":      "my-app",
			"namespace": "default",
		},
		"spec": map[string]interface{}{
			"replicas": 3,
		},
	}

	// Hydrate
	result, err := h.Hydrate(instance)
	if err != nil {
		t.Fatalf("Hydrate() error = %v", err)
	}

	// Verify results
	if len(result.Resources) != 1 {
		t.Errorf("expected 1 resource, got %d", len(result.Resources))
	}

	if len(result.Errors) > 0 {
		t.Errorf("unexpected errors: %v", result.Errors)
	}

	// Verify resource content
	if len(result.Resources) > 0 {
		resource := result.Resources[0]

		if resource["kind"] != "Deployment" {
			t.Errorf("expected kind Deployment, got %v", resource["kind"])
		}

		metadata, ok := resource["metadata"].(map[string]interface{})
		if !ok {
			t.Fatal("metadata not found")
		}

		if metadata["name"] != "my-app" {
			t.Errorf("expected name my-app, got %v", metadata["name"])
		}

		spec, ok := resource["spec"].(map[string]interface{})
		if !ok {
			t.Fatal("spec not found")
		}

		if spec["replicas"] != "3" {
			t.Errorf("expected replicas 3, got %v", spec["replicas"])
		}
	}
}

func TestProcessConditional(t *testing.T) {
	h := NewHydrator("", false)

	tests := []struct {
		name      string
		key       string
		value     interface{}
		context   map[string]interface{}
		shouldAdd bool
	}{
		{
			name: "condition true",
			key:  "$if(.spec.enabled):",
			value: map[string]interface{}{
				"feature": "enabled",
			},
			context: map[string]interface{}{
				"spec": map[string]interface{}{
					"enabled": true,
				},
			},
			shouldAdd: true,
		},
		{
			name: "condition false",
			key:  "$if(.spec.enabled):",
			value: map[string]interface{}{
				"feature": "enabled",
			},
			context: map[string]interface{}{
				"spec": map[string]interface{}{
					"enabled": false,
				},
			},
			shouldAdd: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create evaluator with context
			evaluator := NewEvaluator(tt.context)

			result, err := h.processConditional(tt.key, tt.value, evaluator, tt.context)

			if tt.shouldAdd {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result == nil {
					t.Error("expected result, got nil")
				}
			} else {
				if result != nil {
					t.Error("expected nil result for false condition")
				}
			}
		})
	}
}

func TestFindTemplate(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "hydrator-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create template
	apiDir := filepath.Join(tempDir, "api", "v1alpha1")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("failed to create api dir: %v", err)
	}

	templatePath := filepath.Join(apiDir, "web_service_template.yaml")
	if err := os.WriteFile(templatePath, []byte("resources: []"), 0644); err != nil {
		t.Fatalf("failed to write template: %v", err)
	}

	// Change to temp directory
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	h := NewHydrator("", false)

	// Test finding template
	found := h.findTemplate("WebService", "v1alpha1")
	if found == "" {
		t.Error("template not found")
	}

	expected := filepath.Join("api", "v1alpha1", "web_service_template.yaml")
	if found != expected {
		t.Errorf("expected %s, got %s", expected, found)
	}

	// Test not found
	notFound := h.findTemplate("NonExistent", "v1alpha1")
	if notFound != "" {
		t.Errorf("expected empty string for non-existent template, got %s", notFound)
	}
}
