package dsl

import (
	"reflect"
	"testing"
)

func TestArrayIndexing(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		data     interface{}
		expected interface{}
		wantErr  bool
	}{
		{
			name: "simple array index",
			expr: ".items[0]",
			data: map[string]interface{}{
				"items": []interface{}{"first", "second", "third"},
			},
			expected: "first",
		},
		{
			name: "array index with number",
			expr: ".items[2]",
			data: map[string]interface{}{
				"items": []interface{}{"a", "b", "c", "d"},
			},
			expected: "c",
		},
		{
			name: "nested array index",
			expr: ".spec.containers[1]",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"containers": []interface{}{
						map[string]interface{}{"name": "nginx"},
						map[string]interface{}{"name": "sidecar"},
					},
				},
			},
			expected: map[string]interface{}{"name": "sidecar"},
		},
		{
			name: "out of bounds",
			expr: ".items[10]",
			data: map[string]interface{}{
				"items": []interface{}{"a", "b"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := ParseExpression(tt.expr)
			if err != nil {
				t.Fatalf("ParseExpression() error = %v", err)
			}

			evaluator := NewEvaluator(tt.data)
			result, err := evaluator.Evaluate(expr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Evaluate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Evaluate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestArithmetic(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		data     interface{}
		expected interface{}
		wantErr  bool
	}{
		{
			name: "addition",
			expr: ".spec.replicas + 2",
			data: map[string]interface{}{
				"spec": map[string]interface{}{"replicas": int64(3)},
			},
			expected: int64(5),
		},
		{
			name: "subtraction",
			expr: ".spec.maxReplicas - .spec.minReplicas",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"maxReplicas": int64(10),
					"minReplicas": int64(2),
				},
			},
			expected: int64(8),
		},
		{
			name: "multiplication",
			expr: ".spec.replicas * 2",
			data: map[string]interface{}{
				"spec": map[string]interface{}{"replicas": int64(3)},
			},
			expected: int64(6),
		},
		{
			name: "division",
			expr: ".spec.total / 4",
			data: map[string]interface{}{
				"spec": map[string]interface{}{"total": int64(20)},
			},
			expected: int64(5),
		},
		{
			name: "modulo",
			expr: ".spec.value % 3",
			data: map[string]interface{}{
				"spec": map[string]interface{}{"value": int64(10)},
			},
			expected: int64(1),
		},
		{
			name: "complex expression",
			expr: "(.spec.base + .spec.increment) * 2",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"base":      int64(10),
					"increment": int64(5),
				},
			},
			expected: int64(30), // (10 + 5) * 2 - using parens for precedence
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := ParseExpression(tt.expr)
			if err != nil {
				t.Fatalf("ParseExpression() error = %v", err)
			}

			evaluator := NewEvaluator(tt.data)
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

func TestStringConcatenation(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		data     interface{}
		expected string
		wantErr  bool
	}{
		{
			name: "simple concatenation",
			expr: ".spec.prefix + \"-\" + .spec.suffix",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"prefix": "app",
					"suffix": "prod",
				},
			},
			expected: "app-prod",
		},
		{
			name: "concatenation with metadata",
			expr: ".metadata.namespace + \"/\" + .metadata.name",
			data: map[string]interface{}{
				"metadata": map[string]interface{}{
					"namespace": "default",
					"name":      "my-app",
				},
			},
			expected: "default/my-app",
		},
		{
			name: "multiple concatenations",
			expr: ".spec.protocol + \"://\" + .spec.host + \":\" + .spec.port",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"protocol": "https",
					"host":     "example.com",
					"port":     int64(8443),
				},
			},
			expected: "https://example.com:8443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := ParseExpression(tt.expr)
			if err != nil {
				t.Fatalf("ParseExpression() error = %v", err)
			}

			evaluator := NewEvaluator(tt.data)
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

func TestCombinedFeatures(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		data     interface{}
		expected interface{}
		wantErr  bool
	}{
		{
			name: "array index with variable",
			expr: ".items[.spec.index]",
			data: map[string]interface{}{
				"items": []interface{}{"a", "b", "c", "d"},
				"spec":  map[string]interface{}{"index": int64(2)},
			},
			expected: "c",
		},
		{
			name: "concatenation with array element",
			expr: ".spec.prefix + \"-\" + .items[0]",
			data: map[string]interface{}{
				"spec":  map[string]interface{}{"prefix": "app"},
				"items": []interface{}{"service", "deployment"},
			},
			expected: "app-service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := ParseExpression(tt.expr)
			if err != nil {
				t.Fatalf("ParseExpression() error = %v", err)
			}

			evaluator := NewEvaluator(tt.data)
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

