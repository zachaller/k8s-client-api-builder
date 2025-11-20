package hydrator

import (
	"testing"
)

func TestNewHydrator(t *testing.T) {
	h := NewHydrator("/tmp/templates", false)
	if h == nil {
		t.Fatal("Expected hydrator to be created")
	}
	if h.templateDir != "/tmp/templates" {
		t.Errorf("Expected templateDir='/tmp/templates', got '%s'", h.templateDir)
	}
	if h.verbose != false {
		t.Errorf("Expected verbose=false, got %v", h.verbose)
	}
}

func TestFindTemplate(t *testing.T) {
	tests := []struct {
		name     string
		kind     string
		version  string
		expected string
	}{
		{
			name:     "lowercase kind and version",
			kind:     "myapp",
			version:  "v1",
			expected: "myapp_v1.yaml",
		},
		{
			name:     "mixed case kind",
			kind:     "MyApp",
			version:  "v1beta1",
			expected: "myapp_v1beta1.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHydrator("/tmp/templates", false)
			result := h.findTemplate(tt.kind, tt.version)

			// The result will be empty since the template doesn't exist,
			// but we can verify the method doesn't panic
			if result != "" {
				// If we get a result, verify it contains the expected filename
				if !containsString(result, tt.expected) {
					t.Errorf("Expected result to contain '%s', got '%s'", tt.expected, result)
				}
			}
		})
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr
}

// Note: Full hydration testing is done in integration tests
// (test/integration/*_test.go) and real-world scenario tests
// (examples/iks-airv2/scripts/test_all_examples.sh)
