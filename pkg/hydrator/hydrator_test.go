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

func TestNestedLoops(t *testing.T) {
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
			name: "nested loop - containers with ports",
			key:  "$for(container in .spec.containers):",
			value: map[string]interface{}{
				"name": "$(container.name)",
				"ports": map[string]interface{}{
					"$for(port in container.ports):": map[string]interface{}{
						"containerPort": "$(port)",
					},
				},
			},
			context: map[string]interface{}{
				"spec": map[string]interface{}{
					"containers": []interface{}{
						map[string]interface{}{
							"name":  "nginx",
							"ports": []interface{}{80, 443},
						},
						map[string]interface{}{
							"name":  "sidecar",
							"ports": []interface{}{8080, 9090},
						},
					},
				},
			},
			expectedCount: 2,
			checkResult: func(t *testing.T, result []interface{}) {
				if len(result) != 2 {
					t.Errorf("expected 2 containers, got %d", len(result))
					return
				}

				// Check first container
				container1 := result[0].(map[string]interface{})
				if container1["name"] != "nginx" {
					t.Errorf("container1 name = %v, want nginx", container1["name"])
				}
				ports1 := container1["ports"].([]interface{})
				if len(ports1) != 2 {
					t.Errorf("container1 ports count = %d, want 2", len(ports1))
				}

				// Check second container
				container2 := result[1].(map[string]interface{})
				if container2["name"] != "sidecar" {
					t.Errorf("container2 name = %v, want sidecar", container2["name"])
				}
				ports2 := container2["ports"].([]interface{})
				if len(ports2) != 2 {
					t.Errorf("container2 ports count = %d, want 2", len(ports2))
				}
			},
		},
		{
			name: "nested loop - services with endpoints",
			key:  "$for(service in .spec.services):",
			value: map[string]interface{}{
				"serviceName": "$(service.name)",
				"endpoints": map[string]interface{}{
					"$for(endpoint in service.endpoints):": map[string]interface{}{
						"address": "$(endpoint.ip)",
						"port":    "$(endpoint.port)",
					},
				},
			},
			context: map[string]interface{}{
				"spec": map[string]interface{}{
					"services": []interface{}{
						map[string]interface{}{
							"name": "web",
							"endpoints": []interface{}{
								map[string]interface{}{"ip": "10.0.0.1", "port": 80},
								map[string]interface{}{"ip": "10.0.0.2", "port": 80},
							},
						},
						map[string]interface{}{
							"name": "api",
							"endpoints": []interface{}{
								map[string]interface{}{"ip": "10.0.1.1", "port": 8080},
							},
						},
					},
				},
			},
			expectedCount: 2,
			checkResult: func(t *testing.T, result []interface{}) {
				if len(result) != 2 {
					t.Errorf("expected 2 services, got %d", len(result))
					return
				}

				// Check first service
				service1 := result[0].(map[string]interface{})
				if service1["serviceName"] != "web" {
					t.Errorf("service1 name = %v, want web", service1["serviceName"])
				}
				endpoints1 := service1["endpoints"].([]interface{})
				if len(endpoints1) != 2 {
					t.Errorf("service1 endpoints count = %d, want 2", len(endpoints1))
				} else {
					ep1 := endpoints1[0].(map[string]interface{})
					if ep1["address"] != "10.0.0.1" || ep1["port"] != "80" {
						t.Errorf("endpoint1 mismatch: %v", ep1)
					}
				}

				// Check second service
				service2 := result[1].(map[string]interface{})
				if service2["serviceName"] != "api" {
					t.Errorf("service2 name = %v, want api", service2["serviceName"])
				}
				endpoints2 := service2["endpoints"].([]interface{})
				if len(endpoints2) != 1 {
					t.Errorf("service2 endpoints count = %d, want 1", len(endpoints2))
				}
			},
		},
		{
			name: "nested loop with conditionals",
			key:  "$for(app in .spec.apps):",
			value: map[string]interface{}{
				"name": "$(app.name)",
				"$if(app.envVars):": map[string]interface{}{
					"env": map[string]interface{}{
						"$for(envVar in app.envVars):": map[string]interface{}{
							"name":  "$(envVar.name)",
							"value": "$(envVar.value)",
						},
					},
				},
			},
			context: map[string]interface{}{
				"spec": map[string]interface{}{
					"apps": []interface{}{
						map[string]interface{}{
							"name": "app1",
							"envVars": []interface{}{
								map[string]interface{}{"name": "DB_HOST", "value": "localhost"},
								map[string]interface{}{"name": "DB_PORT", "value": "5432"},
							},
						},
						map[string]interface{}{
							"name": "app2",
							// No envVars
						},
					},
				},
			},
			expectedCount: 2,
			checkResult: func(t *testing.T, result []interface{}) {
				if len(result) != 2 {
					t.Errorf("expected 2 apps, got %d", len(result))
					return
				}

				// First app should have env vars
				app1 := result[0].(map[string]interface{})
				if app1["name"] != "app1" {
					t.Errorf("app1 name = %v, want app1", app1["name"])
				}
				if env, ok := app1["env"]; ok {
					envList := env.([]interface{})
					if len(envList) != 2 {
						t.Errorf("app1 env count = %d, want 2", len(envList))
					}
				} else {
					t.Error("app1 should have env field")
				}

				// Second app should not have env vars
				app2 := result[1].(map[string]interface{})
				if app2["name"] != "app2" {
					t.Errorf("app2 name = %v, want app2", app2["name"])
				}
				if _, ok := app2["env"]; ok {
					t.Error("app2 should not have env field")
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
