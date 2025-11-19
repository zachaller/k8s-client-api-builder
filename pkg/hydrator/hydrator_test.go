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
			name: "boolean true",
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
			name: "boolean false",
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
		{
			name: "comparison equals true",
			key:  "$if(.spec.replicas == 3):",
			value: map[string]interface{}{
				"ha": "enabled",
			},
			context: map[string]interface{}{
				"spec": map[string]interface{}{
					"replicas": 3,
				},
			},
			shouldAdd: true,
		},
		{
			name: "comparison equals false",
			key:  "$if(.spec.replicas == 5):",
			value: map[string]interface{}{
				"ha": "enabled",
			},
			context: map[string]interface{}{
				"spec": map[string]interface{}{
					"replicas": 3,
				},
			},
			shouldAdd: false,
		},
		{
			name: "comparison greater than",
			key:  "$if(.spec.replicas > 1):",
			value: map[string]interface{}{
				"strategy": "RollingUpdate",
			},
			context: map[string]interface{}{
				"spec": map[string]interface{}{
					"replicas": 3,
				},
			},
			shouldAdd: true,
		},
		{
			name: "comparison less than",
			key:  "$if(.spec.replicas < 10):",
			value: map[string]interface{}{
				"tier": "small",
			},
			context: map[string]interface{}{
				"spec": map[string]interface{}{
					"replicas": 3,
				},
			},
			shouldAdd: true,
		},
		{
			name: "optional field missing (should be false)",
			key:  "$if(.spec.optional):",
			value: map[string]interface{}{
				"feature": "optional",
			},
			context: map[string]interface{}{
				"spec": map[string]interface{}{
					"enabled": true,
				},
			},
			shouldAdd: false,
		},
		{
			name: "non-empty string is truthy",
			key:  "$if(.spec.environment):",
			value: map[string]interface{}{
				"env": "set",
			},
			context: map[string]interface{}{
				"spec": map[string]interface{}{
					"environment": "production",
				},
			},
			shouldAdd: true,
		},
		{
			name: "empty string is falsy",
			key:  "$if(.spec.environment):",
			value: map[string]interface{}{
				"env": "set",
			},
			context: map[string]interface{}{
				"spec": map[string]interface{}{
					"environment": "",
				},
			},
			shouldAdd: false,
		},
		{
			name: "non-zero number is truthy",
			key:  "$if(.spec.count):",
			value: map[string]interface{}{
				"counted": "yes",
			},
			context: map[string]interface{}{
				"spec": map[string]interface{}{
					"count": 5,
				},
			},
			shouldAdd: true,
		},
		{
			name: "zero is falsy",
			key:  "$if(.spec.count):",
			value: map[string]interface{}{
				"counted": "yes",
			},
			context: map[string]interface{}{
				"spec": map[string]interface{}{
					"count": 0,
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

func TestProcessLoop(t *testing.T) {
	h := NewHydrator("", false)

	tests := []struct {
		name          string
		key           string
		value         interface{}
		context       map[string]interface{}
		expectedCount int
		checkResult   func(t *testing.T, result []interface{})
	}{
		{
			name: "simple loop",
			key:  "$for(item in .spec.items):",
			value: map[string]interface{}{
				"name": "$(item.name)",
			},
			context: map[string]interface{}{
				"spec": map[string]interface{}{
					"items": []interface{}{
						map[string]interface{}{"name": "item1"},
						map[string]interface{}{"name": "item2"},
						map[string]interface{}{"name": "item3"},
					},
				},
			},
			expectedCount: 3,
			checkResult: func(t *testing.T, result []interface{}) {
				for i, item := range result {
					m, ok := item.(map[string]interface{})
					if !ok {
						t.Errorf("item %d is not a map", i)
						continue
					}
					expectedName := ""
					switch i {
					case 0:
						expectedName = "item1"
					case 1:
						expectedName = "item2"
					case 2:
						expectedName = "item3"
					}
					if m["name"] != expectedName {
						t.Errorf("item %d: expected name %s, got %v", i, expectedName, m["name"])
					}
				}
			},
		},
		{
			name: "loop with nested fields",
			key:  "$for(envVar in .spec.env):",
			value: map[string]interface{}{
				"name":  "$(envVar.name)",
				"value": "$(envVar.value)",
			},
			context: map[string]interface{}{
				"spec": map[string]interface{}{
					"env": []interface{}{
						map[string]interface{}{"name": "DB_HOST", "value": "localhost"},
						map[string]interface{}{"name": "DB_PORT", "value": "5432"},
					},
				},
			},
			expectedCount: 2,
			checkResult: func(t *testing.T, result []interface{}) {
				if len(result) != 2 {
					return
				}
				m1 := result[0].(map[string]interface{})
				if m1["name"] != "DB_HOST" || m1["value"] != "localhost" {
					t.Errorf("first env var incorrect: %v", m1)
				}
				m2 := result[1].(map[string]interface{})
				if m2["name"] != "DB_PORT" || m2["value"] != "5432" {
					t.Errorf("second env var incorrect: %v", m2)
				}
			},
		},
		{
			name: "empty array",
			key:  "$for(item in .spec.items):",
			value: map[string]interface{}{
				"name": "$(item.name)",
			},
			context: map[string]interface{}{
				"spec": map[string]interface{}{
					"items": []interface{}{},
				},
			},
			expectedCount: 0,
			checkResult: func(t *testing.T, result []interface{}) {
				if len(result) != 0 {
					t.Errorf("expected empty result, got %d items", len(result))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := NewEvaluator(tt.context)
			result, err := h.processLoop(tt.key, tt.value, evaluator, tt.context)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result) != tt.expectedCount {
				t.Errorf("expected %d items, got %d", tt.expectedCount, len(result))
			}

			if tt.checkResult != nil {
				tt.checkResult(t, result)
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
