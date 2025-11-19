package scaffold

import (
	"os"
	"testing"
)

func TestProjectScaffold(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "scaffold-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	scaffolder := NewProjectScaffolder(ProjectConfig{
		Name:    "test-project",
		Domain:  "test.example.com",
		Verbose: false,
	})

	err = scaffolder.Scaffold()
	if err != nil {
		t.Fatalf("Scaffold() error = %v", err)
	}

	// Verify directory structure
	expectedDirs := []string{
		"test-project/cmd/test-project",
		"test-project/api/v1alpha1",
		"test-project/config/crd",
		"test-project/base",
		"test-project/overlays/dev",
		"test-project/overlays/staging",
		"test-project/overlays/prod",
	}

	for _, dir := range expectedDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("expected directory not found: %s", dir)
		}
	}

	// Verify key files
	expectedFiles := []string{
		"test-project/go.mod",
		"test-project/Makefile",
		"test-project/README.md",
		"test-project/PROJECT",
		"test-project/overlays/dev/kustomization.yaml",
		"test-project/overlays/prod/kustomization.yaml",
	}

	for _, file := range expectedFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("expected file not found: %s", file)
		}
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"WebService", "web_service"},
		{"Database", "database"},
		{"MyAPIService", "my_a_p_i_service"},
		{"HTTPServer", "h_t_t_p_server"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ToSnakeCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToSnakeCase(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToLowerPlural(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"WebService", "webservices"},
		{"Database", "databases"},
		{"Process", "processes"},
		{"Service", "services"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ToLowerPlural(tt.input)
			if result != tt.expected {
				t.Errorf("ToLowerPlural(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}
