package overlay

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteBase(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "kustomize-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	baseDir := filepath.Join(tempDir, "base")
	engine := NewKustomizeEngine(baseDir, "", false)

	// Test resources
	resources := []map[string]interface{}{
		{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":      "test-app",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"replicas": 3,
			},
		},
		{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]interface{}{
				"name":      "test-app",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"type": "ClusterIP",
			},
		},
	}

	// Write base
	err = engine.WriteBase(resources)
	if err != nil {
		t.Fatalf("WriteBase() error = %v", err)
	}

	// Verify base directory was created
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		t.Errorf("base directory was not created")
	}

	// Verify kustomization.yaml was created
	kustomizationPath := filepath.Join(baseDir, "kustomization.yaml")
	if _, err := os.Stat(kustomizationPath); os.IsNotExist(err) {
		t.Errorf("kustomization.yaml was not created")
	}

	// Verify resource files were created
	files, err := os.ReadDir(baseDir)
	if err != nil {
		t.Fatalf("failed to read base dir: %v", err)
	}

	// Should have kustomization.yaml + 2 resource files
	if len(files) != 3 {
		t.Errorf("expected 3 files in base, got %d", len(files))
	}
}

func TestGenerateFilename(t *testing.T) {
	engine := NewKustomizeEngine("", "", false)

	tests := []struct {
		name     string
		resource map[string]interface{}
		index    int
		expected string
	}{
		{
			name: "deployment with name",
			resource: map[string]interface{}{
				"kind": "Deployment",
				"metadata": map[string]interface{}{
					"name": "my-app",
				},
			},
			index:    0,
			expected: "deployment-my-app.yaml",
		},
		{
			name: "service with name",
			resource: map[string]interface{}{
				"kind": "Service",
				"metadata": map[string]interface{}{
					"name": "my-service",
				},
			},
			index:    1,
			expected: "service-my-service.yaml",
		},
		{
			name: "resource without metadata",
			resource: map[string]interface{}{
				"kind": "ConfigMap",
			},
			index:    5,
			expected: "configmap-5.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.generateFilename(tt.resource, tt.index)
			if result != tt.expected {
				t.Errorf("generateFilename() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestApplyOverlay(t *testing.T) {
	// Create temp directory structure
	tempDir, err := os.MkdirTemp("", "kustomize-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	baseDir := filepath.Join(tempDir, "base")
	overlayDir := filepath.Join(tempDir, "overlays")

	// Create base resources
	engine := NewKustomizeEngine(baseDir, overlayDir, false)

	resources := []map[string]interface{}{
		{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":      "test-app",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"replicas": 3,
			},
		},
	}

	// Write base
	if err := engine.WriteBase(resources); err != nil {
		t.Fatalf("WriteBase() error = %v", err)
	}

	// Create dev overlay
	devDir := filepath.Join(overlayDir, "dev")
	if err := os.MkdirAll(devDir, 0755); err != nil {
		t.Fatalf("failed to create dev overlay dir: %v", err)
	}

	// Create simple kustomization
	kustomization := `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - ../../base

commonLabels:
  environment: dev
`
	kustomizationPath := filepath.Join(devDir, "kustomization.yaml")
	if err := os.WriteFile(kustomizationPath, []byte(kustomization), 0644); err != nil {
		t.Fatalf("failed to write kustomization: %v", err)
	}

	// Apply overlay using directory path
	result, err := engine.ApplyOverlay(devDir)
	if err != nil {
		t.Fatalf("ApplyOverlay() error = %v", err)
	}

	// Verify result
	if len(result) != 1 {
		t.Errorf("expected 1 resource, got %d", len(result))
	}

	// Verify labels were added
	if len(result) > 0 {
		metadata, ok := result[0]["metadata"].(map[string]interface{})
		if !ok {
			t.Fatal("metadata not found")
		}

		labels, ok := metadata["labels"].(map[string]interface{})
		if !ok {
			t.Fatal("labels not found")
		}

		if labels["environment"] != "dev" {
			t.Errorf("expected environment label 'dev', got %v", labels["environment"])
		}
	}
}

func TestApplyOverlayNotFound(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kustomize-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	engine := NewKustomizeEngine(
		filepath.Join(tempDir, "base"),
		filepath.Join(tempDir, "overlays"),
		false,
	)

	// Try to apply non-existent overlay
	_, err = engine.ApplyOverlay("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent overlay, got nil")
	}
}

func TestResolveOverlayPath(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kustomize-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create overlay structure
	overlayDir := filepath.Join(tempDir, "overlays")
	devDir := filepath.Join(overlayDir, "dev")
	if err := os.MkdirAll(devDir, 0755); err != nil {
		t.Fatalf("failed to create dev dir: %v", err)
	}

	// Create kustomization.yaml
	kustomization := `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ../../base
`
	kustomizationPath := filepath.Join(devDir, "kustomization.yaml")
	if err := os.WriteFile(kustomizationPath, []byte(kustomization), 0644); err != nil {
		t.Fatalf("failed to write kustomization: %v", err)
	}

	engine := NewKustomizeEngine(
		filepath.Join(tempDir, "base"),
		overlayDir,
		false,
	)

	tests := []struct {
		name      string
		input     string
		shouldErr bool
	}{
		{
			name:      "directory path",
			input:     devDir,
			shouldErr: false,
		},
		{
			name:      "kustomization.yaml file",
			input:     kustomizationPath,
			shouldErr: false,
		},
		{
			name:      "non-existent",
			input:     "nonexistent",
			shouldErr: true,
		},
		{
			name:      "directory without kustomization",
			input:     overlayDir,
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, err := engine.resolveOverlayPath(tt.input)

			if tt.shouldErr {
				if err == nil {
					t.Errorf("expected error for input %s, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for input %s: %v", tt.input, err)
				}
				if resolved == "" {
					t.Errorf("expected resolved path, got empty string")
				}
			}
		})
	}
}
