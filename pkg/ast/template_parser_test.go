package ast

import (
	"strings"
	"testing"

	"github.com/zachaller/k8s-client-api-builder/pkg/dsl"
)

func TestParseSimpleTemplate(t *testing.T) {
	// Test parsing a simple resource
	template := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Service",
		"metadata": map[string]interface{}{
			"name": "@expr(.metadata.name + '-svc')",
		},
	}

	// Create parser and parse
	root, err := ParseTemplate([]interface{}{template})
	if err != nil {
		t.Fatalf("ParseTemplate() error = %v", err)
	}

	// Check that we got a root node
	if root == nil {
		t.Fatal("ParseTemplate() returned nil root")
	}

	// Check that there's one resource
	if len(root.Resources) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(root.Resources))
	}
}

func TestParseForLoop(t *testing.T) {
	// Test parsing a for loop
	template := map[string]interface{}{
		"@for(item in .spec.items)": []interface{}{
			map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]interface{}{
					"name": "@expr(item.name)",
				},
			},
		},
	}

	// Parse template
	root, err := ParseTemplate(template)
	if err != nil {
		t.Fatalf("ParseTemplate() error = %v", err)
	}

	// Check that we got a for loop node
	if len(root.Resources) != 1 {
		t.Fatalf("Expected 1 resource, got %d", len(root.Resources))
	}

	forNode, ok := root.Resources[0].(*ForLoopNode)
	if !ok {
		t.Fatalf("Expected ForLoopNode, got %T", root.Resources[0])
	}

	if forNode.Variable != "item" {
		t.Errorf("Expected variable 'item', got '%s'", forNode.Variable)
	}
}

func TestEvaluateSimpleTemplate(t *testing.T) {
	// Test evaluating a simple template
	template := []interface{}{
		map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]interface{}{
				"name": "@expr(.metadata.name + \"-svc\")",
			},
		},
	}

	// Parse
	root, err := ParseTemplate(template)
	if err != nil {
		t.Fatalf("ParseTemplate() error = %v", err)
	}

	// Prepare instance data
	instance := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name": "my-app",
		},
	}

	// Evaluate
	evaluator := NewEvaluator(instance)
	resources, err := evaluator.Evaluate(root)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}

	// Check results
	if len(resources) != 1 {
		t.Fatalf("Expected 1 resource, got %d", len(resources))
	}

	resource := resources[0]
	if resource["apiVersion"] != "v1" {
		t.Errorf("Expected apiVersion 'v1', got '%v'", resource["apiVersion"])
	}

	metadata := resource["metadata"].(map[string]interface{})
	if metadata["name"] != "my-app-svc" {
		t.Errorf("Expected name 'my-app-svc', got '%v'", metadata["name"])
	}
}

func TestEvaluateForLoop(t *testing.T) {
	// Test evaluating a for loop
	template := map[string]interface{}{
		"@for(item in .spec.items)": []interface{}{
			map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]interface{}{
					"name": "@expr(item.name)",
				},
				"data": map[string]interface{}{
					"value": "@expr(item.value)",
				},
			},
		},
	}

	// Parse
	root, err := ParseTemplate(template)
	if err != nil {
		t.Fatalf("ParseTemplate() error = %v", err)
	}

	// Prepare instance data
	instance := map[string]interface{}{
		"spec": map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{
					"name":  "config1",
					"value": "value1",
				},
				map[string]interface{}{
					"name":  "config2",
					"value": "value2",
				},
			},
		},
	}

	// Evaluate
	evaluator := NewEvaluator(instance)
	resources, err := evaluator.Evaluate(root)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}

	// Check results
	if len(resources) != 2 {
		t.Fatalf("Expected 2 resources, got %d", len(resources))
	}

	// Check first resource
	metadata1 := resources[0]["metadata"].(map[string]interface{})
	if metadata1["name"] != "config1" {
		t.Errorf("Expected name 'config1', got '%v'", metadata1["name"])
	}

	data1 := resources[0]["data"].(map[string]interface{})
	if data1["value"] != "value1" {
		t.Errorf("Expected value 'value1', got '%v'", data1["value"])
	}

	// Check second resource
	metadata2 := resources[1]["metadata"].(map[string]interface{})
	if metadata2["name"] != "config2" {
		t.Errorf("Expected name 'config2', got '%v'", metadata2["name"])
	}
}

func TestEvaluateForLoopWithWhere(t *testing.T) {
	// Test evaluating a for loop with where clause
	template := map[string]interface{}{
		"@for(item in .spec.items where item.enabled != false)": []interface{}{
			map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]interface{}{
					"name": "@expr(item.name)",
				},
			},
		},
	}

	// Parse
	root, err := ParseTemplate(template)
	if err != nil {
		t.Fatalf("ParseTemplate() error = %v", err)
	}

	// Prepare instance data
	instance := map[string]interface{}{
		"spec": map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{
					"name":    "config1",
					"enabled": true,
				},
				map[string]interface{}{
					"name":    "config2",
					"enabled": false,
				},
				map[string]interface{}{
					"name":    "config3",
					"enabled": true,
				},
			},
		},
	}

	// Evaluate
	evaluator := NewEvaluator(instance)
	resources, err := evaluator.Evaluate(root)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}

	// Check results - should only have 2 resources (config2 filtered out)
	if len(resources) != 2 {
		t.Fatalf("Expected 2 resources, got %d", len(resources))
	}

	// Check that config2 is not present
	for _, resource := range resources {
		metadata := resource["metadata"].(map[string]interface{})
		if metadata["name"] == "config2" {
			t.Errorf("config2 should have been filtered out")
		}
	}
}

func TestParseConditional(t *testing.T) {
	// Test parsing a conditional
	template := map[string]interface{}{
		"@if(.spec.enabled)": map[string]interface{}{
			"feature": "enabled",
		},
	}

	// Parse template
	root, err := ParseTemplate(template)
	if err != nil {
		t.Fatalf("ParseTemplate() error = %v", err)
	}

	// Check that we got a conditional node
	if len(root.Resources) != 1 {
		t.Fatalf("Expected 1 resource, got %d", len(root.Resources))
	}

	condNode, ok := root.Resources[0].(*ConditionalNode)
	if !ok {
		t.Fatalf("Expected ConditionalNode, got %T", root.Resources[0])
	}

	if condNode.Condition == nil {
		t.Error("Expected non-nil condition")
	}
}

func TestPrinter(t *testing.T) {
	// Test AST printer
	expr, _ := dsl.ParseExpression(".metadata.name")
	template := &RootNode{
		Resources: []Node{
			&ForLoopNode{
				Variable: "item",
				Iterable: expr,
				Body: []Node{
					&MapNode{
						Fields: map[string]Node{
							"name": &ExpressionNode{Expr: expr},
						},
					},
				},
			},
		},
	}

	printer := NewPrinter()
	output, err := printer.Print(template)
	if err != nil {
		t.Fatalf("Print() error = %v", err)
	}

	if output == "" {
		t.Error("Print() returned empty string")
	}

	// Just check that it contains expected keywords
	if !strings.Contains(output, "ForLoop") {
		t.Error("Print() output missing 'ForLoop'")
	}
}
