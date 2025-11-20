package ast

import (
	"fmt"

	"github.com/zachaller/k8s-client-api-builder/pkg/dsl"
)

// Evaluator evaluates an AST and produces Kubernetes resources
type Evaluator struct {
	instance      map[string]interface{}   // The input instance data
	dslEvaluator  *dsl.Evaluator           // DSL expression evaluator (exported for hydrator pass2)
	context       map[string]interface{}   // Current evaluation context (includes loop variables)
	resources     []map[string]interface{} // Collected resources
	resourceDepth int                      // Depth counter to track when we're inside a resource
}

// NewEvaluator creates a new AST evaluator
func NewEvaluator(instance map[string]interface{}) *Evaluator {
	return &Evaluator{
		instance:     instance,
		dslEvaluator: dsl.NewEvaluator(instance),
		context:      instance,
		resources:    []map[string]interface{}{},
	}
}

// GetDSLEvaluator returns the underlying DSL evaluator (for pass2 resource references)
func (e *Evaluator) GetDSLEvaluator() *dsl.Evaluator {
	return e.dslEvaluator
}

// Evaluate evaluates an AST and returns the generated resources
func (e *Evaluator) Evaluate(root *RootNode) ([]map[string]interface{}, error) {
	_, err := root.Accept(e)
	if err != nil {
		return nil, err
	}
	return e.resources, nil
}

// VisitRoot visits the root node
func (e *Evaluator) VisitRoot(node *RootNode) (interface{}, error) {
	for _, resource := range node.Resources {
		_, err := resource.Accept(e)
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

// VisitForLoop visits a for loop node
func (e *Evaluator) VisitForLoop(node *ForLoopNode) (interface{}, error) {
	// Evaluate the iterable expression
	iterableValue, err := e.evaluateExpression(node.Iterable)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate iterable: %w", err)
	}

	// Convert to slice
	items, ok := iterableValue.([]interface{})
	if !ok {
		// Try to convert from other types
		switch v := iterableValue.(type) {
		case []map[string]interface{}:
			items = make([]interface{}, len(v))
			for i, item := range v {
				items[i] = item
			}
		default:
			return nil, fmt.Errorf("iterable must be an array, got %T", iterableValue)
		}
	}

	// Iterate over items
	results := []interface{}{}
	for _, item := range items {
		// Create new context with loop variable
		loopContext := e.copyContext()
		loopContext[node.Variable] = item

		// If there's a where clause, evaluate it
		if node.WhereClause != nil {
			// Create evaluator with loop context
			loopEvaluator := dsl.NewEvaluator(loopContext)
			condResult, err := loopEvaluator.Evaluate(node.WhereClause)
			if err != nil {
				// Skip items where condition evaluation fails
				continue
			}

			// Check if condition is true
			include := false
			switch v := condResult.(type) {
			case bool:
				include = v
			case string:
				include = v != "" && v != "false"
			case int, int32, int64:
				include = v != 0
			default:
				include = condResult != nil
			}

			if !include {
				continue
			}
		}

		// Execute loop body with new context
		oldContext := e.context
		oldEvaluator := e.dslEvaluator
		e.context = loopContext
		e.dslEvaluator = dsl.NewEvaluator(loopContext)

		for _, bodyNode := range node.Body {
			result, err := bodyNode.Accept(e)
			if err != nil {
				e.context = oldContext
				e.dslEvaluator = oldEvaluator
				return nil, err
			}
			if result != nil {
				results = append(results, result)
			}
		}

		e.context = oldContext
		e.dslEvaluator = oldEvaluator
	}

	// Note: Resources have already been added to e.resources by VisitResource/VisitMap
	return results, nil
}

// VisitConditional visits a conditional node
func (e *Evaluator) VisitConditional(node *ConditionalNode) (interface{}, error) {
	// Evaluate the condition
	condResult, err := e.evaluateExpression(node.Condition)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate condition: %w", err)
	}

	// Check if condition is true
	conditionTrue := false
	switch v := condResult.(type) {
	case bool:
		conditionTrue = v
	case string:
		conditionTrue = v != "" && v != "false" && v != "0"
	case int, int32, int64:
		conditionTrue = v != 0
	case float32, float64:
		conditionTrue = v != 0.0
	default:
		conditionTrue = v != nil
	}

	// Execute the appropriate branch
	var branch []Node
	if conditionTrue {
		branch = node.ThenBranch
	} else {
		branch = node.ElseBranch
	}

	results := []interface{}{}
	for _, branchNode := range branch {
		result, err := branchNode.Accept(e)
		if err != nil {
			return nil, err
		}
		if result != nil {
			results = append(results, result)
		}
	}

	return results, nil
}

// VisitResource visits a resource node (K8s resource)
func (e *Evaluator) VisitResource(node *ResourceNode) (interface{}, error) {
	resource := make(map[string]interface{})

	for key, valueNode := range node.Fields {
		value, err := valueNode.Accept(e)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate field %s: %w", key, err)
		}
		resource[key] = value
	}

	// Add resource to the collected resources
	e.resources = append(e.resources, resource)

	return resource, nil
}

// VisitField visits a field node
func (e *Evaluator) VisitField(node *FieldNode) (interface{}, error) {
	return node.Value.Accept(e)
}

// VisitExpression visits an expression node
func (e *Evaluator) VisitExpression(node *ExpressionNode) (interface{}, error) {
	return e.evaluateExpression(node.Expr)
}

// VisitLiteral visits a literal node
func (e *Evaluator) VisitLiteral(node *LiteralNode) (interface{}, error) {
	return node.Value, nil
}

// VisitArray visits an array node
func (e *Evaluator) VisitArray(node *ArrayNode) (interface{}, error) {
	result := make([]interface{}, 0)

	for _, elem := range node.Elements {
		// Check if element is a control flow node
		switch elemNode := elem.(type) {
		case *ForLoopNode:
			// For loop in array - expand inline
			loopResult, err := elemNode.Accept(e)
			if err != nil {
				return nil, err
			}
			// Flatten loop results into array
			if loopResults, ok := loopResult.([]interface{}); ok {
				result = append(result, loopResults...)
			}
		case *ConditionalNode:
			// Conditional in array - include if true
			condResult, err := elemNode.Accept(e)
			if err != nil {
				return nil, err
			}
			if condResults, ok := condResult.([]interface{}); ok {
				result = append(result, condResults...)
			}
		default:
			// Regular element
			value, err := elem.Accept(e)
			if err != nil {
				return nil, err
			}
			result = append(result, value)
		}
	}

	return result, nil
}

// VisitMap visits a map node
func (e *Evaluator) VisitMap(node *MapNode) (interface{}, error) {
	result := make(map[string]interface{})

	// Check if this is a top-level resource (has both apiVersion and kind)
	_, hasAPIVersion := node.Fields["apiVersion"]
	_, hasKind := node.Fields["kind"]
	isResource := hasAPIVersion && hasKind

	// If this is a resource, increment depth before processing children
	if isResource {
		e.resourceDepth++
	}

	for key, valueNode := range node.Fields {
		// Check if this is a control flow key
		switch vNode := valueNode.(type) {
		case *ForLoopNode:
			// For loop in map - merge results into parent map
			loopResult, err := vNode.Accept(e)
			if err != nil {
				return nil, err
			}
			// Merge loop results (each should be a map) into parent
			if loopResults, ok := loopResult.([]interface{}); ok {
				for _, lr := range loopResults {
					if lrMap, ok := lr.(map[string]interface{}); ok {
						for k, v := range lrMap {
							result[k] = v
						}
					}
				}
			}
		case *ConditionalNode:
			// Conditional in map - merge if true
			condResult, err := vNode.Accept(e)
			if err != nil {
				return nil, err
			}
			if condResults, ok := condResult.([]interface{}); ok {
				for _, cr := range condResults {
					if crMap, ok := cr.(map[string]interface{}); ok {
						for k, v := range crMap {
							result[k] = v
						}
					}
				}
			}
		default:
			// Regular field
			value, err := valueNode.Accept(e)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate map field %s: %w", key, err)
			}
			result[key] = value
		}
	}

	// Only collect as a resource if we're at depth 1 (top-level resource)
	if isResource && e.resourceDepth == 1 {
		e.resources = append(e.resources, result)
	}

	// Decrement depth after processing
	if isResource {
		e.resourceDepth--
	}

	return result, nil
}

// evaluateExpression evaluates a DSL expression in the current context
func (e *Evaluator) evaluateExpression(expr *dsl.Expression) (interface{}, error) {
	return e.dslEvaluator.Evaluate(expr)
}

// copyContext creates a copy of the current context
func (e *Evaluator) copyContext() map[string]interface{} {
	newContext := make(map[string]interface{})
	for k, v := range e.context {
		newContext[k] = v
	}
	return newContext
}

// GetResources returns all resources generated by the evaluator
func (e *Evaluator) GetResources() []map[string]interface{} {
	return e.resources
}

// RegisterResource registers a resource in the DSL evaluator (for cross-resource references)
func (e *Evaluator) RegisterResource(apiVersion, kind, name string, resource map[string]interface{}) {
	e.dslEvaluator.RegisterResource(apiVersion, kind, name, resource)
}
