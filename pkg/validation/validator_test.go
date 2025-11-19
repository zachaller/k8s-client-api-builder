package validation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidatorCreation(t *testing.T) {
	validator := NewValidator("config/crd", false)
	if validator == nil {
		t.Error("expected validator, got nil")
	}

	if validator.crdDir != "config/crd" {
		t.Errorf("expected crdDir 'config/crd', got %s", validator.crdDir)
	}
}

func TestLoadSchemas(t *testing.T) {
	// Create temp directory with CRD
	tempDir, err := os.MkdirTemp("", "validator-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	crdDir := filepath.Join(tempDir, "config", "crd")
	if err := os.MkdirAll(crdDir, 0755); err != nil {
		t.Fatalf("failed to create crd dir: %v", err)
	}

	// Create a simple CRD
	crd := `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: webservices.platform.example.com
spec:
  group: platform.example.com
  names:
    kind: WebService
    plural: webservices
  scope: Namespaced
  versions:
  - name: v1alpha1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        properties:
          spec:
            type: object
            properties:
              image:
                type: string
              replicas:
                type: integer
                minimum: 1
                maximum: 100
`
	crdPath := filepath.Join(crdDir, "webservice.yaml")
	if err := os.WriteFile(crdPath, []byte(crd), 0644); err != nil {
		t.Fatalf("failed to write CRD: %v", err)
	}

	// Create validator
	validator := NewValidator(crdDir, false)

	// Load schemas
	err = validator.LoadSchemas()
	if err != nil {
		t.Fatalf("LoadSchemas() error = %v", err)
	}

	// Verify schema was loaded
	if len(validator.schemas) == 0 {
		t.Error("expected schemas to be loaded")
	}
}

func TestValidate(t *testing.T) {
	// For this test, we'll skip actual validation since it requires
	// a full CRD setup. This is more of an integration test.
	// Just test the basic structure

	validator := NewValidator("", false)

	instance := map[string]interface{}{
		"apiVersion": "platform.example.com/v1alpha1",
		"kind":       "WebService",
		"metadata": map[string]interface{}{
			"name": "test",
		},
		"spec": map[string]interface{}{
			"image":    "nginx:latest",
			"replicas": 3,
		},
	}

	// This will fail because no schemas are loaded, but it shouldn't crash
	_, err := validator.Validate(instance)
	if err == nil {
		t.Error("expected error for missing schema")
	}
}

func TestValidateMissingFields(t *testing.T) {
	validator := NewValidator("", false)

	tests := []struct {
		name     string
		instance map[string]interface{}
		wantErr  bool
	}{
		{
			name: "missing apiVersion",
			instance: map[string]interface{}{
				"kind": "WebService",
			},
			wantErr: false, // Returns validation result, not error
		},
		{
			name: "missing kind",
			instance: map[string]interface{}{
				"apiVersion": "platform.example.com/v1alpha1",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validator.Validate(tt.instance)

			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if result != nil && result.Valid {
				t.Error("expected validation to fail for missing fields")
			}
		})
	}
}
