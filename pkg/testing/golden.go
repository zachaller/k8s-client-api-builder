package testing

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "update golden files")

// CompareWithGolden compares output with golden file
func CompareWithGolden(t *testing.T, got []byte, goldenFile string) {
	t.Helper()

	if *update {
		if err := UpdateGolden(t, got, goldenFile); err != nil {
			t.Fatalf("failed to update golden file: %v", err)
		}
		t.Logf("Updated golden file: %s", goldenFile)
		return
	}

	want, err := os.ReadFile(goldenFile)
	if err != nil {
		t.Fatalf("failed to read golden file %s: %v", goldenFile, err)
	}

	if !bytes.Equal(got, want) {
		t.Errorf("output differs from golden file %s\n\nTo update: go test -update\n\nGot:\n%s\n\nWant:\n%s",
			goldenFile, string(got), string(want))
	}
}

// UpdateGolden updates golden file with new output
func UpdateGolden(t *testing.T, data []byte, goldenFile string) error {
	t.Helper()

	dir := filepath.Dir(goldenFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(goldenFile, data, 0644)
}

// LoadGolden loads a golden file
func LoadGolden(t *testing.T, goldenFile string) []byte {
	t.Helper()

	data, err := os.ReadFile(goldenFile)
	if err != nil {
		t.Fatalf("failed to read golden file: %v", err)
	}

	return data
}

// CompareYAMLWithGolden compares YAML output with golden file (ignoring formatting differences)
func CompareYAMLWithGolden(t *testing.T, got []byte, goldenFile string) {
	t.Helper()

	// For now, just use byte comparison
	// In the future, could parse YAML and do semantic comparison
	CompareWithGolden(t, got, goldenFile)
}

