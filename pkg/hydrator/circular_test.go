package hydrator

import (
	"testing"
)

func TestDetectCircularReferences(t *testing.T) {
	tests := []struct {
		name      string
		graph     DependencyGraph
		wantCycle bool
	}{
		{
			name: "no cycles",
			graph: DependencyGraph{
				"v1/Service/a":    []string{"v1/ConfigMap/b"},
				"v1/ConfigMap/b":  []string{},
				"apps/v1/Deployment/c": []string{"v1/Service/a"},
			},
			wantCycle: false,
		},
		{
			name: "simple cycle",
			graph: DependencyGraph{
				"v1/Service/a":   []string{"v1/ConfigMap/b"},
				"v1/ConfigMap/b": []string{"v1/Service/a"},
			},
			wantCycle: true,
		},
		{
			name: "three-way cycle",
			graph: DependencyGraph{
				"v1/Service/a":   []string{"v1/ConfigMap/b"},
				"v1/ConfigMap/b": []string{"v1/Secret/c"},
				"v1/Secret/c":    []string{"v1/Service/a"},
			},
			wantCycle: true,
		},
		{
			name: "self-reference",
			graph: DependencyGraph{
				"v1/Service/a": []string{"v1/Service/a"},
			},
			wantCycle: true,
		},
		{
			name: "no cycle with wildcard",
			graph: DependencyGraph{
				"v1/Service/a":   []string{"v1/ConfigMap/*"},
				"v1/ConfigMap/b": []string{"v1/Service/*"},
			},
			wantCycle: false, // Wildcards are ignored
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cycles := detectCircularReferences(tt.graph)

			hasCycle := len(cycles) > 0

			if hasCycle != tt.wantCycle {
				t.Errorf("detectCircularReferences() found cycles = %v, want %v\nCycles: %v", hasCycle, tt.wantCycle, cycles)
			}
		})
	}
}

func TestGetResourceKey(t *testing.T) {
	tests := []struct {
		name     string
		resource map[string]interface{}
		expected string
		wantErr  bool
	}{
		{
			name: "valid resource",
			resource: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Service",
				"metadata": map[string]interface{}{
					"name": "my-app",
				},
			},
			expected: "v1/Service/my-app",
		},
		{
			name: "missing apiVersion",
			resource: map[string]interface{}{
				"kind": "Service",
				"metadata": map[string]interface{}{
					"name": "my-app",
				},
			},
			wantErr: true,
		},
		{
			name: "missing kind",
			resource: map[string]interface{}{
				"apiVersion": "v1",
				"metadata": map[string]interface{}{
					"name": "my-app",
				},
			},
			wantErr: true,
		},
		{
			name: "missing name",
			resource: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Service",
				"metadata":   map[string]interface{}{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getResourceKey(tt.resource)

			if (err != nil) != tt.wantErr {
				t.Errorf("getResourceKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result != tt.expected {
				t.Errorf("getResourceKey() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractResourceRefsFromString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single reference",
			input:    `$(resource("v1", "Service", "my-app").spec.clusterIP)`,
			expected: []string{"v1/Service/my-app"},
		},
		{
			name:  "multiple references",
			input: `http://$(resource("v1", "Service", "api").spec.clusterIP):$(resource("v1", "Service", "api").spec.ports[0].port)`,
			expected: []string{
				"v1/Service/api",
				"v1/Service/api",
			},
		},
		{
			name:     "no references",
			input:    `just a regular string`,
			expected: []string{},
		},
		{
			name:     "reference with expression name",
			input:    `$(resource("v1", "Secret", .metadata.name + "-secret").metadata.name)`,
			expected: []string{"v1/Secret/*"}, // Can't determine name statically
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractResourceRefsFromString(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("extractResourceRefsFromString() returned %d refs, want %d\nGot: %v\nWant: %v",
					len(result), len(tt.expected), result, tt.expected)
				return
			}

			for i, ref := range result {
				if ref != tt.expected[i] {
					t.Errorf("extractResourceRefsFromString()[%d] = %v, want %v", i, ref, tt.expected[i])
				}
			}
		})
	}
}

