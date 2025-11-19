package testing

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"sigs.k8s.io/yaml"
)

// TestFramework provides utilities for testing KRM SDK projects
type TestFramework struct {
	TempDir    string
	ProjectDir string
	BinaryPath string
	SDKBinary  string
	T          *testing.T
}

// NewTestFramework creates a new test framework instance
func NewTestFramework(t *testing.T) *TestFramework {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "krm-sdk-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Find krm-sdk binary
	sdkBinary := findSDKBinary(t)

	return &TestFramework{
		TempDir:   tempDir,
		SDKBinary: sdkBinary,
		T:         t,
	}
}

// InitProject initializes a test project
func (f *TestFramework) InitProject(name, domain string) error {
	f.T.Helper()

	f.ProjectDir = filepath.Join(f.TempDir, name)

	cmd := exec.Command(f.SDKBinary, "init", name, "--domain", domain)
	cmd.Dir = f.TempDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("init failed: %w\nOutput: %s", err, output)
	}

	// Add local replace directive for testing
	if err := f.addLocalReplace(); err != nil {
		return fmt.Errorf("failed to add local replace: %w", err)
	}

	return nil
}

// addLocalReplace adds a replace directive to use local krm-sdk
func (f *TestFramework) addLocalReplace() error {
	f.T.Helper()

	// Find the krm-sdk directory (where the framework is)
	sdkDir := filepath.Dir(filepath.Dir(f.SDKBinary)) // bin/krm-sdk -> .
	
	goModPath := filepath.Join(f.ProjectDir, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return err
	}

	// Replace the commented replace directive with an active one
	replaced := string(content)
	replaced = strings.Replace(replaced,
		"// replace github.com/yourusername/krm-sdk => /path/to/krm-sdk",
		fmt.Sprintf("replace github.com/yourusername/krm-sdk => %s", sdkDir),
		1)

	return os.WriteFile(goModPath, []byte(replaced), 0644)
}

// CreateAPI creates a new API in the project
func (f *TestFramework) CreateAPI(group, version, kind string) error {
	f.T.Helper()

	if f.ProjectDir == "" {
		return fmt.Errorf("project not initialized")
	}

	cmd := exec.Command(f.SDKBinary, "create", "api",
		"--group", group,
		"--version", version,
		"--kind", kind)
	cmd.Dir = f.ProjectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("create api failed: %w\nOutput: %s", err, output)
	}

	return nil
}

// BuildProject builds the project binary
func (f *TestFramework) BuildProject() error {
	f.T.Helper()

	if f.ProjectDir == "" {
		return fmt.Errorf("project not initialized")
	}

	// Run go mod tidy first
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = f.ProjectDir
	tidyOutput, err := tidyCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go mod tidy failed: %w\nOutput: %s", err, tidyOutput)
	}

	// Build the project
	cmd := exec.Command("make", "build")
	cmd.Dir = f.ProjectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build failed: %w\nOutput: %s", err, output)
	}

	// Set binary path
	projectName := filepath.Base(f.ProjectDir)
	f.BinaryPath = filepath.Join(f.ProjectDir, "bin", projectName)

	return nil
}

// GenerateResources generates resources from an instance file
func (f *TestFramework) GenerateResources(instanceFile string) ([]map[string]interface{}, error) {
	f.T.Helper()

	if f.BinaryPath == "" {
		return nil, fmt.Errorf("project binary not built")
	}

	cmd := exec.Command(f.BinaryPath, "generate", "-f", instanceFile)
	cmd.Dir = f.ProjectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("generate failed: %w\nOutput: %s", err, output)
	}

	// Parse YAML output
	return parseYAMLResources(output)
}

// GenerateWithOverlay generates resources with kustomize overlay
func (f *TestFramework) GenerateWithOverlay(instanceFile, overlay string) ([]map[string]interface{}, error) {
	f.T.Helper()

	if f.BinaryPath == "" {
		return nil, fmt.Errorf("project binary not built")
	}

	cmd := exec.Command(f.BinaryPath, "generate", "-f", instanceFile, "--overlay", overlay)
	cmd.Dir = f.ProjectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("generate with overlay failed: %w\nOutput: %s", err, output)
	}

	// Parse YAML output
	return parseYAMLResources(output)
}

// ValidateOutput validates generated output against expectations
func (f *TestFramework) ValidateOutput(resources []map[string]interface{}, expectations []Expectation) error {
	f.T.Helper()

	for _, expectation := range expectations {
		if err := expectation.Validate(resources); err != nil {
			return err
		}
	}

	return nil
}

// Cleanup cleans up test artifacts
func (f *TestFramework) Cleanup() {
	if f.TempDir != "" {
		os.RemoveAll(f.TempDir)
	}
}

// findSDKBinary finds the krm-sdk binary
func findSDKBinary(t *testing.T) string {
	t.Helper()

	// Try relative path from test location
	candidates := []string{
		"../../bin/krm-sdk",
		"../../../bin/krm-sdk",
		"../../../../bin/krm-sdk",
		"bin/krm-sdk",
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			abs, err := filepath.Abs(candidate)
			if err == nil {
				return abs
			}
		}
	}

	// Try in PATH
	path, err := exec.LookPath("krm-sdk")
	if err == nil {
		return path
	}

	t.Fatal("krm-sdk binary not found. Run 'make build' first.")
	return ""
}

// parseYAMLResources parses YAML output into resources
func parseYAMLResources(data []byte) ([]map[string]interface{}, error) {
	// Split by document separator
	docs := splitYAMLDocs(string(data))

	resources := make([]map[string]interface{}, 0, len(docs))
	for _, doc := range docs {
		if doc == "" {
			continue
		}

		var resource map[string]interface{}
		if err := yaml.Unmarshal([]byte(doc), &resource); err != nil {
			return nil, fmt.Errorf("failed to unmarshal resource: %w", err)
		}

		resources = append(resources, resource)
	}

	return resources, nil
}

// splitYAMLDocs splits YAML documents by ---
func splitYAMLDocs(data string) []string {
	docs := []string{}
	current := ""

	for _, line := range splitLines(data) {
		if line == "---" {
			if current != "" {
				docs = append(docs, current)
				current = ""
			}
		} else {
			if current != "" {
				current += "\n"
			}
			current += line
		}
	}

	if current != "" {
		docs = append(docs, current)
	}

	return docs
}

// splitLines splits string into lines
func splitLines(s string) []string {
	lines := []string{}
	current := ""

	for _, ch := range s {
		if ch == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(ch)
		}
	}

	if current != "" {
		lines = append(lines, current)
	}

	return lines
}

