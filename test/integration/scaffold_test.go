package integration

import (
	"os"
	"path/filepath"
	"testing"

	krmtesting "github.com/zachaller/k8s-client-api-builder/pkg/testing"
)

func TestProjectScaffolding(t *testing.T) {
	framework := krmtesting.NewTestFramework(t)
	defer framework.Cleanup()

	err := framework.InitProject("test-project", "test.example.com")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Verify directory structure
	expectedDirs := []string{
		"cmd/test-project",
		"cmd/test-project/commands",
		"api/v1alpha1",
		"config/crd",
		"config/samples",
		"base",
		"overlays/dev",
		"overlays/dev/patches",
		"overlays/staging",
		"overlays/staging/patches",
		"overlays/prod",
		"overlays/prod/patches",
		"instances",
		"hack",
	}

	for _, dir := range expectedDirs {
		path := filepath.Join(framework.ProjectDir, dir)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected directory not found: %s", dir)
		}
	}

	// Verify key files
	expectedFiles := []string{
		"go.mod",
		"Makefile",
		"README.md",
		"PROJECT",
		".gitignore",
		"cmd/test-project/main.go",
		"cmd/test-project/commands/root.go",
		"api/v1alpha1/groupversion_info.go",
		"api/v1alpha1/register.go",
		"hack/boilerplate.go.txt",
		"overlays/dev/kustomization.yaml",
		"overlays/dev/patches/replicas.yaml",
		"overlays/staging/kustomization.yaml",
		"overlays/prod/kustomization.yaml",
		"overlays/prod/patches/replicas.yaml",
		"overlays/prod/patches/resources.yaml",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(framework.ProjectDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file not found: %s", file)
		}
	}
}

func TestAPIScaffolding(t *testing.T) {
	framework := krmtesting.NewTestFramework(t)
	defer framework.Cleanup()

	// Initialize project
	err := framework.InitProject("test-api-project", "test.example.com")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Create API
	err = framework.CreateAPI("platform", "v1alpha1", "WebService")
	if err != nil {
		t.Fatalf("create api failed: %v", err)
	}

	// Verify API files were created
	expectedFiles := []string{
		"api/v1alpha1/web_service_types.go",
		"api/v1alpha1/web_service_template.yaml",
		"config/samples/web_service.yaml",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(framework.ProjectDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected API file not found: %s", file)
		}
	}
}
