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

func TestParseForLoop(t *testing.T) {
	tests := []struct {
		name         string
		expr         string
		wantVarName  string
		wantIterPath string
		wantErr      bool
	}{
		{
			name:         "simple loop",
			expr:         "item in .spec.items",
			wantVarName:  "item",
			wantIterPath: ".spec.items",
			wantErr:      false,
		},
		{
			name:         "loop with underscore",
			expr:         "env_var in .spec.envVars",
			wantVarName:  "env_var",
			wantIterPath: ".spec.envVars",
			wantErr:      false,
		},
		{
			name:         "nested path",
			expr:         "container in .spec.template.containers",
			wantVarName:  "container",
			wantIterPath: ".spec.template.containers",
			wantErr:      false,
		},
		{
			name:    "missing 'in' keyword",
			expr:    "item .spec.items",
			wantErr: true,
		},
		{
			name:    "path doesn't start with dot",
			expr:    "item in spec.items",
			wantErr: true,
		},
		{
			name:         "loop with spaces",
			expr:         "  item  in  .spec.items  ",
			wantVarName:  "item",
			wantIterPath: ".spec.items",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			varName, iterPath, err := ParseForLoop(tt.expr)

			if tt.wantErr {
				if err == nil {
					t.Error("ParseForLoop() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseForLoop() unexpected error: %v", err)
				return
			}

			if varName != tt.wantVarName {
				t.Errorf("ParseForLoop() varName = %v, want %v", varName, tt.wantVarName)
			}

			if iterPath != tt.wantIterPath {
				t.Errorf("ParseForLoop() iterPath = %v, want %v", iterPath, tt.wantIterPath)
			}
		})
	}
}

func TestInlineIfFunction(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		data     interface{}
		expected interface{}
		wantErr  bool
	}{
		{
			name: "condition true returns first value",
			expr: "if(.spec.enabled, \"yes\", \"no\")",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"enabled": true,
				},
			},
			expected: "yes",
		},
		{
			name: "condition false returns second value",
			expr: "if(.spec.enabled, \"yes\", \"no\")",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"enabled": false,
				},
			},
			expected: "no",
		},
		{
			name: "numeric comparison true",
			expr: "if(.spec.replicas > 1, \"ClusterIP\", \"LoadBalancer\")",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"replicas": 3,
				},
			},
			expected: "ClusterIP",
		},
		{
			name: "numeric comparison false",
			expr: "if(.spec.replicas > 1, \"ClusterIP\", \"LoadBalancer\")",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"replicas": 1,
				},
			},
			expected: "LoadBalancer",
		},
		{
			name: "truthy string",
			expr: "if(.spec.environment, \"has-env\", \"no-env\")",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"environment": "production",
				},
			},
			expected: "has-env",
		},
		{
			name: "empty string is falsy",
			expr: "if(.spec.environment, \"has-env\", \"no-env\")",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"environment": "",
				},
			},
			expected: "no-env",
		},
		{
			name: "non-zero number is truthy",
			expr: "if(.spec.count, \"counted\", \"zero\")",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"count": 5,
				},
			},
			expected: "counted",
		},
		{
			name: "zero is falsy",
			expr: "if(.spec.count, \"counted\", \"zero\")",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"count": 0,
				},
			},
			expected: "zero",
		},
		{
			name: "inequality check",
			expr: "if(.spec.type != \"production\", \"non-prod\", \"prod\")",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"type": "staging",
				},
			},
			expected: "non-prod",
		},
		{
			name: "numeric values as results",
			expr: "if(.spec.enableHA, 3, 1)",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"enableHA": true,
				},
			},
			expected: int64(3),
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
				t.Errorf("Evaluate() = %v (type %T), want %v (type %T)", result, result, tt.expected, tt.expected)
			}
		})
	}
}

func TestEvaluateStringWithInlineIf(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		data     interface{}
		expected string
		wantErr  bool
	}{
		{
			name:  "inline if with $if syntax",
			input: "type: $if(.spec.enableIngress, \"ClusterIP\", \"LoadBalancer\")",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"enableIngress": true,
				},
			},
			expected: "type: ClusterIP",
		},
		{
			name:  "inline if false condition",
			input: "type: $if(.spec.enableIngress, \"ClusterIP\", \"LoadBalancer\")",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"enableIngress": false,
				},
			},
			expected: "type: LoadBalancer",
		},
		{
			name:  "inline if in middle of string",
			input: "The service type is $if(.spec.ha, \"highly available\", \"standard\")",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"ha": true,
				},
			},
			expected: "The service type is highly available",
		},
		{
			name:  "multiple inline ifs",
			input: "replicas: $if(.spec.ha, 3, 1), type: $if(.spec.public, \"LoadBalancer\", \"ClusterIP\")",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"ha":     true,
					"public": false,
				},
			},
			expected: "replicas: 3, type: ClusterIP",
		},
		{
			name:  "inline if with comparison",
			input: "tier: $if(.spec.replicas > 5, \"large\", \"small\")",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"replicas": 10,
				},
			},
			expected: "tier: large",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := NewEvaluator(tt.data)
			result, err := evaluator.EvaluateString(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result != tt.expected {
				t.Errorf("EvaluateString() = %v, want %v", result, tt.expected)
			}
		})
	}
}
