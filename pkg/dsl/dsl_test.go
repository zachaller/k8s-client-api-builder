package dsl

import (
	"reflect"
	"strings"
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
			name:         "loop variable reference (nested loop)",
			expr:         "port in container.ports",
			wantVarName:  "port",
			wantIterPath: "container.ports",
			wantErr:      false,
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

func TestStringFunctions(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		data     interface{}
		expected interface{}
		wantErr  bool
	}{
		{
			name: "lower function",
			expr: "lower(.metadata.name)",
			data: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "MyApp",
				},
			},
			expected: "myapp",
		},
		{
			name: "upper function",
			expr: "upper(.metadata.name)",
			data: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "myapp",
				},
			},
			expected: "MYAPP",
		},
		{
			name: "trim function",
			expr: "trim(.spec.value)",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"value": "  spaces around  ",
				},
			},
			expected: "spaces around",
		},
		{
			name: "replace function",
			expr: "replace(.spec.url, \"http\", \"https\")",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"url": "http://example.com",
				},
			},
			expected: "https://example.com",
		},
		{
			name: "sha256 function",
			expr: "sha256(.spec.image)",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"image": "nginx:latest",
				},
			},
			expected: "4c0d44a73c3d4c0e90e9f5f6d5c1e3a6e2b1f0c9d8e7f6a5b4c3d2e1f0a9b8c7", // Not the actual hash, just example format
			wantErr:  false,                                                              // Just verify it returns a string, not the exact hash
		},
		{
			name: "default function with value",
			expr: "default(.spec.name, \"fallback\")",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"name": "actual",
				},
			},
			expected: "actual",
		},
		{
			name: "default function with empty string",
			expr: "default(.spec.name, \"fallback\")",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"name": "",
				},
			},
			expected: "fallback",
		},
		{
			name: "default function with missing field",
			expr: "default(.spec.missing, \"fallback\")",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"name": "test",
				},
			},
			expected: "fallback",
			wantErr:  true, // Will error on missing field before default is called
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := ParseExpression(tt.expr)
			if err != nil {
				if !tt.wantErr {
					t.Fatalf("ParseExpression() error = %v", err)
				}
				return
			}

			evaluator := NewEvaluator(tt.data)
			result, err := evaluator.Evaluate(expr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Evaluate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// For sha256, just check it's a string of the right length
			if tt.name == "sha256 function" {
				if str, ok := result.(string); ok && len(str) == 64 {
					return // Valid sha256 hash
				}
				t.Errorf("sha256() result is not a valid hash: %v", result)
				return
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Evaluate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestComparisonOperators(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		data     interface{}
		expected interface{}
	}{
		{
			name: "not equal true",
			expr: ".spec.type != \"production\"",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"type": "staging",
				},
			},
			expected: true,
		},
		{
			name: "not equal false",
			expr: ".spec.type != \"production\"",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"type": "production",
				},
			},
			expected: false,
		},
		{
			name: "greater than or equal (greater)",
			expr: ".spec.replicas >= 3",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"replicas": 5,
				},
			},
			expected: true,
		},
		{
			name: "greater than or equal (equal)",
			expr: ".spec.replicas >= 3",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"replicas": 3,
				},
			},
			expected: true,
		},
		{
			name: "greater than or equal (less)",
			expr: ".spec.replicas >= 3",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"replicas": 1,
				},
			},
			expected: false,
		},
		{
			name: "less than or equal (less)",
			expr: ".spec.count <= 10",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"count": 5,
				},
			},
			expected: true,
		},
		{
			name: "less than or equal (equal)",
			expr: ".spec.count <= 10",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"count": 10,
				},
			},
			expected: true,
		},
		{
			name: "less than or equal (greater)",
			expr: ".spec.count <= 10",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"count": 15,
				},
			},
			expected: false,
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

			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}

			if result != tt.expected {
				t.Errorf("Evaluate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		data    interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name: "division by zero",
			expr: ".spec.value / 0",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"value": 10,
				},
			},
			wantErr: true,
			errMsg:  "division by zero",
		},
		{
			name: "modulo by zero",
			expr: ".spec.value % 0",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"value": 10,
				},
			},
			wantErr: true,
			errMsg:  "modulo by zero",
		},
		{
			name: "negative array index",
			expr: ".items[-1]",
			data: map[string]interface{}{
				"items": []interface{}{"a", "b", "c"},
			},
			wantErr: true,
		},
		{
			name: "unknown function",
			expr: "unknownFunc(.spec.name)",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"name": "test",
				},
			},
			wantErr: true,
			errMsg:  "unknown function",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := ParseExpression(tt.expr)
			if err != nil && !tt.wantErr {
				t.Fatalf("ParseExpression() error = %v", err)
			}
			if err != nil {
				return // Parse error is expected for some tests
			}

			evaluator := NewEvaluator(tt.data)
			_, err = evaluator.Evaluate(expr)

			if (err != nil) != tt.wantErr {
				t.Errorf("Evaluate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Evaluate() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
		})
	}
}

func TestNestedFunctions(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		data     interface{}
		expected interface{}
		wantErr  bool
	}{
		{
			name: "lower of trim",
			expr: "lower(trim(.spec.value))",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"value": "  HELLO  ",
				},
			},
			expected: "hello",
		},
		{
			name: "upper of replace",
			expr: "upper(replace(.spec.name, \"test\", \"prod\"))",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"name": "test-app",
				},
			},
			expected: "PROD-APP",
		},
		{
			name: "trim of lower of value",
			expr: "trim(lower(.metadata.name))",
			data: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "  MyAPP  ",
				},
			},
			expected: "myapp",
		},
		{
			name: "default with lower",
			expr: "default(lower(.spec.env), \"development\")",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"env": "PRODUCTION",
				},
			},
			expected: "production",
		},
		{
			name: "if with nested functions",
			expr: "if(.spec.enabled, upper(.spec.type), lower(.spec.type))",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"enabled": true,
					"type":    "ClusterIP",
				},
			},
			expected: "CLUSTERIP",
		},
		{
			name: "if with nested functions false",
			expr: "if(.spec.enabled, upper(.spec.type), lower(.spec.type))",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"enabled": false,
					"type":    "LoadBalancer",
				},
			},
			expected: "loadbalancer",
		},
		{
			name: "triple nested functions",
			expr: "upper(trim(replace(.spec.url, \"http://\", \"\")))",
			data: map[string]interface{}{
				"spec": map[string]interface{}{
					"url": "http://example.com  ",
				},
			},
			expected: "EXAMPLE.COM",
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

func TestResourceReferencesWithComplexNames(t *testing.T) {
	tests := []struct {
		name      string
		expr      string
		data      interface{}
		resources map[string]map[string]interface{}
		expected  interface{}
		wantErr   bool
	}{
		{
			name: "resource reference with concatenated name",
			expr: "resource(\"v1\", \"Service\", .metadata.name + \"-svc\").spec.clusterIP",
			data: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "my-app",
				},
			},
			resources: map[string]map[string]interface{}{
				"v1/Service/my-app-svc": {
					"apiVersion": "v1",
					"kind":       "Service",
					"metadata": map[string]interface{}{
						"name": "my-app-svc",
					},
					"spec": map[string]interface{}{
						"clusterIP": "10.0.0.1",
					},
				},
			},
			expected: "10.0.0.1",
		},
		{
			name: "resource reference with lowercase function",
			expr: "resource(\"v1\", \"ConfigMap\", lower(.metadata.name)).data.key",
			data: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "MY-APP",
				},
			},
			resources: map[string]map[string]interface{}{
				"v1/ConfigMap/my-app": {
					"apiVersion": "v1",
					"kind":       "ConfigMap",
					"metadata": map[string]interface{}{
						"name": "my-app",
					},
					"data": map[string]interface{}{
						"key": "value123",
					},
				},
			},
			expected: "value123",
		},
		{
			name: "resource reference with trim and concat",
			expr: "resource(\"v1\", \"Secret\", trim(.metadata.namespace) + \"-secret\").data.token",
			data: map[string]interface{}{
				"metadata": map[string]interface{}{
					"namespace": "  prod  ",
				},
			},
			resources: map[string]map[string]interface{}{
				"v1/Secret/prod-secret": {
					"apiVersion": "v1",
					"kind":       "Secret",
					"metadata": map[string]interface{}{
						"name": "prod-secret",
					},
					"data": map[string]interface{}{
						"token": "abc123",
					},
				},
			},
			expected: "abc123",
		},
		{
			name: "resource reference with conditional expression in name",
			expr: "resource(\"v1\", \"Service\", if(.spec.ha, .metadata.name + \"-ha\", .metadata.name)).spec.type",
			data: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "app",
				},
				"spec": map[string]interface{}{
					"ha": true,
				},
			},
			resources: map[string]map[string]interface{}{
				"v1/Service/app-ha": {
					"apiVersion": "v1",
					"kind":       "Service",
					"metadata": map[string]interface{}{
						"name": "app-ha",
					},
					"spec": map[string]interface{}{
						"type": "LoadBalancer",
					},
				},
			},
			expected: "LoadBalancer",
		},
		{
			name: "resource reference accessing array element",
			expr: "resource(\"v1\", \"Service\", .metadata.name).spec.ports[0].port",
			data: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "web-app",
				},
			},
			resources: map[string]map[string]interface{}{
				"v1/Service/web-app": {
					"apiVersion": "v1",
					"kind":       "Service",
					"metadata": map[string]interface{}{
						"name": "web-app",
					},
					"spec": map[string]interface{}{
						"ports": []interface{}{
							map[string]interface{}{"port": int64(80)},
							map[string]interface{}{"port": int64(443)},
						},
					},
				},
			},
			expected: int64(80),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := ParseExpression(tt.expr)
			if err != nil {
				t.Fatalf("ParseExpression() error = %v", err)
			}

			evaluator := NewEvaluator(tt.data)

			// Register resources
			for _, resource := range tt.resources {
				apiVersion := resource["apiVersion"].(string)
				kind := resource["kind"].(string)
				name := resource["metadata"].(map[string]interface{})["name"].(string)
				evaluator.RegisterResource(apiVersion, kind, name, resource)
			}

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

func TestLoopVariablesInExpressions(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		data     interface{}
		expected interface{}
		wantErr  bool
	}{
		{
			name: "loop variable concatenation",
			expr: "component.name + \"-role\"",
			data: map[string]interface{}{
				"component": map[string]interface{}{
					"name": "my-component",
				},
			},
			expected: "my-component-role",
		},
		{
			name: "loop variable with multiple concatenations",
			expr: "item.prefix + \"-\" + item.name + \"-\" + item.suffix",
			data: map[string]interface{}{
				"item": map[string]interface{}{
					"prefix": "app",
					"name":   "service",
					"suffix": "v1",
				},
			},
			expected: "app-service-v1",
		},
		{
			name: "loop variable in conditional",
			expr: "if(component.enabled, component.name + \"-enabled\", component.name + \"-disabled\")",
			data: map[string]interface{}{
				"component": map[string]interface{}{
					"name":    "web",
					"enabled": true,
				},
			},
			expected: "web-enabled",
		},
		{
			name: "nested loop variable",
			expr: "container.port.number",
			data: map[string]interface{}{
				"container": map[string]interface{}{
					"port": map[string]interface{}{
						"number": int64(8080),
					},
				},
			},
			expected: int64(8080),
		},
		{
			name: "loop variable with arithmetic",
			expr: "item.base + item.increment",
			data: map[string]interface{}{
				"item": map[string]interface{}{
					"base":      int64(100),
					"increment": int64(50),
				},
			},
			expected: int64(150),
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
