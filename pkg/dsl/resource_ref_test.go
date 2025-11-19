package dsl

import (
	"testing"
)

func TestParseResourceRef(t *testing.T) {
	tests := []struct {
		name      string
		expr      string
		wantErr   bool
		checkFunc func(*testing.T, *Expression)
	}{
		{
			name: "simple resource reference",
			expr: `resource("v1", "Service", "my-app").spec.clusterIP`,
			checkFunc: func(t *testing.T, expr *Expression) {
				if expr.Type != ExprResourceRef {
					t.Errorf("expected ExprResourceRef, got %v", expr.Type)
				}
				if expr.ResourceRef == nil {
					t.Fatal("ResourceRef is nil")
				}
				if expr.ResourceRef.APIVersion != "v1" {
					t.Errorf("expected apiVersion 'v1', got %s", expr.ResourceRef.APIVersion)
				}
				if expr.ResourceRef.Kind != "Service" {
					t.Errorf("expected kind 'Service', got %s", expr.ResourceRef.Kind)
				}
				if expr.ResourceRef.FieldPath != "spec.clusterIP" {
					t.Errorf("expected fieldPath 'spec.clusterIP', got %s", expr.ResourceRef.FieldPath)
				}
			},
		},
		{
			name: "resource reference with array index",
			expr: `resource("v1", "Service", "my-app").spec.ports[0].port`,
			checkFunc: func(t *testing.T, expr *Expression) {
				if expr.ResourceRef.FieldPath != "spec.ports[0].port" {
					t.Errorf("expected fieldPath 'spec.ports[0].port', got %s", expr.ResourceRef.FieldPath)
				}
			},
		},
		{
			name: "resource reference with expression name",
			expr: `resource("v1", "Service", .metadata.name).metadata.name`,
			checkFunc: func(t *testing.T, expr *Expression) {
				if expr.ResourceRef.Name == nil {
					t.Fatal("Name expression is nil")
				}
				if expr.ResourceRef.Name.Type != ExprPath {
					t.Errorf("expected name to be ExprPath, got %v", expr.ResourceRef.Name.Type)
				}
			},
		},
		{
			name: "resource reference with concatenated name",
			expr: `resource("v1", "Secret", .metadata.name + "-secret").metadata.name`,
			checkFunc: func(t *testing.T, expr *Expression) {
				if expr.ResourceRef.Name == nil {
					t.Fatal("Name expression is nil")
				}
				if expr.ResourceRef.Name.Type != ExprConcat {
					t.Errorf("expected name to be ExprConcat, got %v", expr.ResourceRef.Name.Type)
				}
			},
		},
		{
			name: "resource reference without field path",
			expr: `resource("v1", "Service", "my-app")`,
			checkFunc: func(t *testing.T, expr *Expression) {
				if expr.ResourceRef.FieldPath != "" {
					t.Errorf("expected empty fieldPath, got %s", expr.ResourceRef.FieldPath)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := ParseExpression(tt.expr)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseExpression() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.checkFunc != nil {
				tt.checkFunc(t, expr)
			}
		})
	}
}

func TestEvaluateResourceRef(t *testing.T) {
	// Create evaluator
	instance := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":      "my-app",
			"namespace": "default",
		},
		"spec": map[string]interface{}{
			"port": int64(8080),
		},
	}

	evaluator := NewEvaluator(instance)

	// Register some resources
	service := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Service",
		"metadata": map[string]interface{}{
			"name":      "my-app",
			"namespace": "default",
		},
		"spec": map[string]interface{}{
			"clusterIP": "10.0.0.1",
			"ports": []interface{}{
				map[string]interface{}{
					"port":       int64(80),
					"targetPort": int64(8080),
				},
			},
		},
	}

	evaluator.RegisterResource("v1", "Service", "my-app", service)

	tests := []struct {
		name     string
		expr     string
		expected interface{}
		wantErr  bool
	}{
		{
			name:     "reference service clusterIP",
			expr:     `resource("v1", "Service", "my-app").spec.clusterIP`,
			expected: "10.0.0.1",
		},
		{
			name:     "reference service port",
			expr:     `resource("v1", "Service", "my-app").spec.ports[0].port`,
			expected: int64(80),
		},
		{
			name:     "reference service name",
			expr:     `resource("v1", "Service", "my-app").metadata.name`,
			expected: "my-app",
		},
		{
			name:     "reference with expression name",
			expr:     `resource("v1", "Service", .metadata.name).spec.clusterIP`,
			expected: "10.0.0.1",
		},
		{
			name:    "reference non-existent resource",
			expr:    `resource("v1", "Service", "nonexistent").spec.clusterIP`,
			wantErr: true,
		},
		{
			name:    "reference non-existent field",
			expr:    `resource("v1", "Service", "my-app").spec.nonexistent`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := ParseExpression(tt.expr)
			if err != nil {
				t.Fatalf("ParseExpression() error = %v", err)
			}

			result, err := evaluator.Evaluate(expr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Evaluate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result != tt.expected {
				t.Errorf("Evaluate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestResourceRefInString(t *testing.T) {
	instance := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name": "my-app",
		},
	}

	evaluator := NewEvaluator(instance)

	// Register a service
	service := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Service",
		"metadata": map[string]interface{}{
			"name": "my-app",
		},
		"spec": map[string]interface{}{
			"clusterIP": "10.0.0.1",
			"ports": []interface{}{
				map[string]interface{}{"port": int64(80)},
			},
		},
	}

	evaluator.RegisterResource("v1", "Service", "my-app", service)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "service URL",
			input:    `http://$(resource("v1", "Service", "my-app").spec.clusterIP):$(resource("v1", "Service", "my-app").spec.ports[0].port)`,
			expected: "http://10.0.0.1:80",
		},
		{
			name:     "service name reference",
			input:    `backend-service: $(resource("v1", "Service", "my-app").metadata.name)`,
			expected: "backend-service: my-app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluator.EvaluateString(tt.input)
			if err != nil {
				t.Fatalf("EvaluateString() error = %v", err)
			}

			if result != tt.expected {
				t.Errorf("EvaluateString() = %v, want %v", result, tt.expected)
			}
		})
	}
}
