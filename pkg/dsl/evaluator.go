package dsl

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Evaluator evaluates DSL expressions against data
type Evaluator struct {
	data      interface{}
	functions map[string]Function
	resources map[string]map[string]interface{} // Resource registry for cross-resource references
}

// Function represents a DSL function
type Function func(args ...interface{}) (interface{}, error)

// NewEvaluator creates a new evaluator with the given data
func NewEvaluator(data interface{}) *Evaluator {
	e := &Evaluator{
		data:      data,
		functions: make(map[string]Function),
		resources: make(map[string]map[string]interface{}),
	}
	e.registerBuiltinFunctions()
	return e
}

// RegisterResource adds a resource to the registry for cross-resource references
func (e *Evaluator) RegisterResource(apiVersion, kind, name string, resource map[string]interface{}) {
	key := fmt.Sprintf("%s/%s/%s", apiVersion, kind, name)
	e.resources[key] = resource
}

// GetResources returns all registered resources
func (e *Evaluator) GetResources() map[string]map[string]interface{} {
	return e.resources
}

// RegisterFunction registers a custom function
func (e *Evaluator) RegisterFunction(name string, fn Function) {
	e.functions[name] = fn
}

// Evaluate evaluates an expression
func (e *Evaluator) Evaluate(expr *Expression) (interface{}, error) {
	switch expr.Type {
	case ExprPath:
		return e.evaluatePath(expr.Path)
	case ExprFunction:
		return e.evaluateFunction(expr.Function, expr.Args)
	case ExprBinary:
		return e.evaluateBinary(expr)
	case ExprLiteral:
		return e.evaluateLiteral(expr.Path)
	case ExprArrayIndex:
		return e.evaluateArrayIndex(expr)
	case ExprConcat:
		return e.evaluateConcat(expr)
	case ExprResourceRef:
		return e.evaluateResourceRef(expr.ResourceRef)
	case ExprUnary:
		return e.evaluateUnary(expr)
	default:
		return nil, fmt.Errorf("unknown expression type: %d", expr.Type)
	}
}

// EvaluateString evaluates a string that may contain variable substitutions
func (e *Evaluator) EvaluateString(input string) (string, error) {
	result := input

	// Find all $(...) and $if(...) expressions, handling nested parentheses
	for {
		// Check for $if( first (inline ternary)
		ifStart := strings.Index(result, "$if(")
		dollarStart := strings.Index(result, "$(")

		var start int
		var prefixLen int

		// Determine which pattern comes first
		if ifStart != -1 && (dollarStart == -1 || ifStart < dollarStart) {
			start = ifStart
			prefixLen = 4 // length of "$if("
		} else if dollarStart != -1 {
			start = dollarStart
			prefixLen = 2 // length of "$("
		} else {
			break
		}

		// Find matching closing parenthesis
		depth := 0
		end := -1
		for i := start + prefixLen - 1; i < len(result); i++ {
			if result[i] == '(' {
				depth++
			} else if result[i] == ')' {
				depth--
				if depth == 0 {
					end = i
					break
				}
			}
		}

		if end == -1 {
			return "", fmt.Errorf("unmatched parenthesis in expression")
		}

		// Extract expression (without prefix and ))
		fullMatch := result[start : end+1]
		exprStr := result[start+prefixLen : end]

		// If this was a $if( expression, wrap it as a function call
		if prefixLen == 4 {
			exprStr = "if(" + exprStr + ")"
		}

		expr, err := ParseExpression(exprStr)
		if err != nil {
			return "", fmt.Errorf("failed to parse expression '%s': %w", exprStr, err)
		}

		value, err := e.Evaluate(expr)
		if err != nil {
			return "", fmt.Errorf("failed to evaluate expression '%s': %w", exprStr, err)
		}

		// Convert value to string
		valueStr := fmt.Sprintf("%v", value)
		result = strings.Replace(result, fullMatch, valueStr, 1)
	}

	return result, nil
}

// evaluatePath evaluates a path expression like ".spec.name" or "envVar.name"
func (e *Evaluator) evaluatePath(path string) (interface{}, error) {
	// Handle paths that start with '.' (regular paths from root)
	var parts []string
	if strings.HasPrefix(path, ".") {
		// Remove leading dot
		path = path[1:]
		parts = strings.Split(path, ".")
	} else {
		// Loop variable path (e.g., "envVar.name")
		parts = strings.Split(path, ".")
	}

	// Navigate through the data structure
	current := e.data
	for _, part := range parts {
		if part == "" {
			continue
		}

		val := reflect.ValueOf(current)

		// Handle pointers
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}

		switch val.Kind() {
		case reflect.Map:
			// Handle map access
			key := reflect.ValueOf(part)
			mapVal := val.MapIndex(key)
			if !mapVal.IsValid() {
				return nil, fmt.Errorf("key '%s' not found in map", part)
			}
			current = mapVal.Interface()

		case reflect.Struct:
			// Handle struct field access
			field := val.FieldByName(strings.Title(part))
			if !field.IsValid() {
				// Try lowercase
				field = val.FieldByName(part)
			}
			if !field.IsValid() {
				return nil, fmt.Errorf("field '%s' not found in struct", part)
			}
			current = field.Interface()

		default:
			return nil, fmt.Errorf("cannot access '%s' on type %s", part, val.Kind())
		}
	}

	return current, nil
}

// evaluateFunction evaluates a function call
func (e *Evaluator) evaluateFunction(name string, args []string) (interface{}, error) {
	fn, ok := e.functions[name]
	if !ok {
		return nil, fmt.Errorf("unknown function: %s", name)
	}

	// Evaluate arguments
	evalArgs := make([]interface{}, len(args))
	for i, arg := range args {
		expr, err := ParseExpression(arg)
		if err != nil {
			return nil, fmt.Errorf("failed to parse argument: %w", err)
		}

		val, err := e.Evaluate(expr)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate argument: %w", err)
		}

		evalArgs[i] = val
	}

	return fn(evalArgs...)
}

// evaluateBinary evaluates a binary expression
func (e *Evaluator) evaluateBinary(expr *Expression) (interface{}, error) {
	left, err := e.Evaluate(expr.Left)
	if err != nil {
		// For comparison operators, treat evaluation errors (e.g., missing fields) as nil
		// This allows expressions like "ws.disabled != true" to work when disabled doesn't exist
		switch expr.Operator {
		case "==", "!=", ">", "<", ">=", "<=":
			left = nil
		default:
			return nil, err
		}
	}

	right, err := e.Evaluate(expr.Right)
	if err != nil {
		// Same treatment for right side
		switch expr.Operator {
		case "==", "!=", ">", "<", ">=", "<=":
			right = nil
		default:
			return nil, err
		}
	}

	switch expr.Operator {
	// Comparison operators
	case "==":
		return fmt.Sprintf("%v", left) == fmt.Sprintf("%v", right), nil
	case "!=":
		return fmt.Sprintf("%v", left) != fmt.Sprintf("%v", right), nil
	case ">":
		return compareValues(left, right) > 0, nil
	case "<":
		return compareValues(left, right) < 0, nil
	case ">=":
		return compareValues(left, right) >= 0, nil
	case "<=":
		return compareValues(left, right) <= 0, nil

	// Arithmetic operators
	case "+":
		// Check if either operand is a string - if so, do string concatenation
		_, leftIsStr := left.(string)
		_, rightIsStr := right.(string)
		if leftIsStr || rightIsStr {
			// String concatenation
			return fmt.Sprintf("%v", left) + fmt.Sprintf("%v", right), nil
		}
		// Numeric addition
		return performArithmetic(left, right, "+")
	case "-":
		return performArithmetic(left, right, "-")
	case "*":
		return performArithmetic(left, right, "*")
	case "/":
		return performArithmetic(left, right, "/")
	case "%":
		return performArithmetic(left, right, "%")

	default:
		return nil, fmt.Errorf("unknown operator: %s", expr.Operator)
	}
}

// evaluateUnary evaluates unary expressions (!, -, etc.)
func (e *Evaluator) evaluateUnary(expr *Expression) (interface{}, error) {
	// Evaluate the operand
	operand, err := e.Evaluate(expr.Operand)
	if err != nil {
		// For NOT operator, treat missing fields as false/nil
		if expr.Operator == "!" {
			operand = nil
		} else {
			return nil, err
		}
	}

	switch expr.Operator {
	case "!":
		// Logical NOT
		return !isTruthy(operand), nil
	case "-":
		// Unary minus
		val, err := toFloat64(operand)
		if err != nil {
			return nil, fmt.Errorf("cannot negate non-numeric value: %v", operand)
		}
		return -val, nil
	default:
		return nil, fmt.Errorf("unknown unary operator: %s", expr.Operator)
	}
}

// isTruthy determines if a value is truthy in boolean context
func isTruthy(val interface{}) bool {
	if val == nil {
		return false
	}

	switch v := val.(type) {
	case bool:
		return v
	case string:
		return v != ""
	case int, int32, int64:
		return v != 0
	case float32, float64:
		return v != 0.0
	default:
		// Non-zero/non-nil values are truthy
		return true
	}
}

// evaluateArrayIndex evaluates array indexing expressions
func (e *Evaluator) evaluateArrayIndex(expr *Expression) (interface{}, error) {
	// Evaluate the base path to get the array/map
	baseValue, err := e.evaluatePath(expr.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate array path: %w", err)
	}

	// Evaluate the index expression
	indexValue, err := e.Evaluate(expr.Index)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate array index: %w", err)
	}

	// Access the array/map element
	val := reflect.ValueOf(baseValue)

	// Handle different collection types
	switch val.Kind() {
	case reflect.Map:
		// For maps, use the index value directly as a key (can be string or other type)
		keyVal := reflect.ValueOf(indexValue)
		mapVal := val.MapIndex(keyVal)
		if !mapVal.IsValid() {
			return nil, fmt.Errorf("key %v not found in map", indexValue)
		}
		return mapVal.Interface(), nil

	case reflect.Slice, reflect.Array:
		// For arrays/slices, convert index to int
		index, err := toInt(indexValue)
		if err != nil {
			return nil, fmt.Errorf("array index must be an integer: %w", err)
		}
		if index < 0 || index >= val.Len() {
			return nil, fmt.Errorf("array index %d out of bounds (length %d)", index, val.Len())
		}
		return val.Index(index).Interface(), nil

	default:
		return nil, fmt.Errorf("cannot index into type %s", val.Kind())
	}
}

// evaluateConcat evaluates concatenation expressions
func (e *Evaluator) evaluateConcat(expr *Expression) (interface{}, error) {
	var result strings.Builder

	for i, elem := range expr.Elements {
		value, err := e.Evaluate(elem)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate concatenation element %d: %w", i, err)
		}

		// Convert to string
		result.WriteString(fmt.Sprintf("%v", value))
	}

	return result.String(), nil
}

// evaluateResourceRef evaluates a resource reference
func (e *Evaluator) evaluateResourceRef(ref *ResourceReference) (interface{}, error) {
	// Evaluate the name expression
	nameValue, err := e.Evaluate(ref.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate resource name: %w", err)
	}

	name := fmt.Sprintf("%v", nameValue)

	// Build resource key
	key := fmt.Sprintf("%s/%s/%s", ref.APIVersion, ref.Kind, name)

	// Look up resource
	resource, ok := e.resources[key]
	if !ok {
		// Provide helpful error message with available resources
		available := []string{}
		for k := range e.resources {
			available = append(available, k)
		}
		return nil, fmt.Errorf("resource not found: %s\nAvailable resources: %v", key, available)
	}

	// If no field path, return the entire resource
	if ref.FieldPath == "" {
		return resource, nil
	}

	// Navigate to the field
	return e.navigateResourceField(resource, ref.FieldPath)
}

// navigateResourceField navigates to a field in a resource
func (e *Evaluator) navigateResourceField(resource map[string]interface{}, fieldPath string) (interface{}, error) {
	// Parse field path (e.g., "spec.clusterIP" or "spec.ports[0].port")
	parts := strings.Split(fieldPath, ".")

	current := interface{}(resource)
	for _, part := range parts {
		if part == "" {
			continue
		}

		// Check for array indexing in the part
		if strings.Contains(part, "[") && strings.Contains(part, "]") {
			// Parse array access: field[index]
			openBracket := strings.Index(part, "[")
			closeBracket := strings.Index(part, "]")

			fieldName := part[:openBracket]
			indexStr := part[openBracket+1 : closeBracket]

			// Navigate to the field first
			if fieldName != "" {
				val := reflect.ValueOf(current)
				if val.Kind() == reflect.Ptr {
					val = val.Elem()
				}

				switch val.Kind() {
				case reflect.Map:
					key := reflect.ValueOf(fieldName)
					mapVal := val.MapIndex(key)
					if !mapVal.IsValid() {
						return nil, fmt.Errorf("field '%s' not found in resource", fieldName)
					}
					current = mapVal.Interface()
				default:
					return nil, fmt.Errorf("cannot access field '%s' on type %s", fieldName, val.Kind())
				}
			}

			// Parse index
			index, err := strconv.Atoi(indexStr)
			if err != nil {
				return nil, fmt.Errorf("invalid array index '%s': %w", indexStr, err)
			}

			// Access array element
			val := reflect.ValueOf(current)
			if val.Kind() == reflect.Slice || val.Kind() == reflect.Array {
				if index < 0 || index >= val.Len() {
					return nil, fmt.Errorf("array index %d out of bounds (length %d)", index, val.Len())
				}
				current = val.Index(index).Interface()
			} else {
				return nil, fmt.Errorf("cannot index into type %s", val.Kind())
			}
		} else {
			// Regular field access
			val := reflect.ValueOf(current)
			if val.Kind() == reflect.Ptr {
				val = val.Elem()
			}

			switch val.Kind() {
			case reflect.Map:
				key := reflect.ValueOf(part)
				mapVal := val.MapIndex(key)
				if !mapVal.IsValid() {
					return nil, fmt.Errorf("field '%s' not found in resource", part)
				}
				current = mapVal.Interface()
			case reflect.Struct:
				field := val.FieldByName(strings.Title(part))
				if !field.IsValid() {
					field = val.FieldByName(part)
				}
				if !field.IsValid() {
					return nil, fmt.Errorf("field '%s' not found in struct", part)
				}
				current = field.Interface()
			default:
				return nil, fmt.Errorf("cannot access '%s' on type %s", part, val.Kind())
			}
		}
	}

	return current, nil
}

// evaluateLiteral evaluates a literal value
func (e *Evaluator) evaluateLiteral(value string) (interface{}, error) {
	// Try to parse as number
	if num, err := strconv.ParseInt(value, 10, 64); err == nil {
		return num, nil
	}

	if num, err := strconv.ParseFloat(value, 64); err == nil {
		return num, nil
	}

	// Try to parse as boolean
	if value == "true" {
		return true, nil
	}
	if value == "false" {
		return false, nil
	}

	// Remove quotes if present (support both double and single quotes)
	if len(value) >= 2 && strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
		return value[1 : len(value)-1], nil
	}
	if len(value) >= 2 && strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
		return value[1 : len(value)-1], nil
	}

	// Return as string
	return value, nil
}

// registerBuiltinFunctions registers built-in DSL functions
func (e *Evaluator) registerBuiltinFunctions() {
	// String functions
	e.RegisterFunction("lower", func(args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("lower() requires 1 argument")
		}
		return strings.ToLower(fmt.Sprintf("%v", args[0])), nil
	})

	e.RegisterFunction("upper", func(args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("upper() requires 1 argument")
		}
		return strings.ToUpper(fmt.Sprintf("%v", args[0])), nil
	})

	e.RegisterFunction("trim", func(args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("trim() requires 1 argument")
		}
		return strings.TrimSpace(fmt.Sprintf("%v", args[0])), nil
	})

	e.RegisterFunction("replace", func(args ...interface{}) (interface{}, error) {
		if len(args) != 3 {
			return nil, fmt.Errorf("replace() requires 3 arguments")
		}
		str := fmt.Sprintf("%v", args[0])
		old := fmt.Sprintf("%v", args[1])
		new := fmt.Sprintf("%v", args[2])
		return strings.ReplaceAll(str, old, new), nil
	})

	e.RegisterFunction("trimPrefix", func(args ...interface{}) (interface{}, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("trimPrefix() requires 2 arguments")
		}
		str := fmt.Sprintf("%v", args[0])
		prefix := fmt.Sprintf("%v", args[1])
		return strings.TrimPrefix(str, prefix), nil
	})

	e.RegisterFunction("trimSuffix", func(args ...interface{}) (interface{}, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("trimSuffix() requires 2 arguments")
		}
		str := fmt.Sprintf("%v", args[0])
		suffix := fmt.Sprintf("%v", args[1])
		return strings.TrimSuffix(str, suffix), nil
	})

	// Hash functions
	e.RegisterFunction("sha256", func(args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("sha256() requires 1 argument")
		}
		str := fmt.Sprintf("%v", args[0])
		hash := sha256.Sum256([]byte(str))
		return hex.EncodeToString(hash[:]), nil
	})

	// Utility functions
	e.RegisterFunction("default", func(args ...interface{}) (interface{}, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("default() requires 2 arguments")
		}
		if args[0] == nil || args[0] == "" {
			return args[1], nil
		}
		return args[0], nil
	})

	// Inline if function (ternary operator)
	e.RegisterFunction("if", func(args ...interface{}) (interface{}, error) {
		if len(args) != 3 {
			return nil, fmt.Errorf("if() requires 3 arguments: condition, trueValue, falseValue")
		}

		// Evaluate the condition
		condition := false
		switch v := args[0].(type) {
		case bool:
			condition = v
		case string:
			condition = v != "" && v != "false" && v != "0"
		case int, int32, int64:
			num, _ := toInt(v)
			condition = num != 0
		case float32, float64:
			num, _ := toFloat64(v)
			condition = num != 0.0
		default:
			condition = v != nil
		}

		if condition {
			return args[1], nil
		}
		return args[2], nil
	})

	// Array manipulation functions
	e.RegisterFunction("prepend", func(args ...interface{}) (interface{}, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("prepend() requires at least 2 arguments: item(s) to prepend, array")
		}

		// Last argument is the array
		arrayArg := args[len(args)-1]
		itemsToPrepend := args[:len(args)-1]

		// Convert array argument to slice
		var arr []interface{}
		switch v := arrayArg.(type) {
		case []interface{}:
			arr = v
		case []string:
			arr = make([]interface{}, len(v))
			for i, s := range v {
				arr[i] = s
			}
		case []int:
			arr = make([]interface{}, len(v))
			for i, n := range v {
				arr[i] = n
			}
		default:
			return nil, fmt.Errorf("prepend() last argument must be an array, got %T", arrayArg)
		}

		// Create new array with prepended items
		result := make([]interface{}, 0, len(itemsToPrepend)+len(arr))
		result = append(result, itemsToPrepend...)
		result = append(result, arr...)

		return result, nil
	})

	e.RegisterFunction("append", func(args ...interface{}) (interface{}, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("append() requires at least 2 arguments: array, item(s) to append")
		}

		// First argument is the array
		arrayArg := args[0]
		itemsToAppend := args[1:]

		// Convert array argument to slice
		var arr []interface{}
		switch v := arrayArg.(type) {
		case []interface{}:
			arr = v
		case []string:
			arr = make([]interface{}, len(v))
			for i, s := range v {
				arr[i] = s
			}
		case []int:
			arr = make([]interface{}, len(v))
			for i, n := range v {
				arr[i] = n
			}
		default:
			return nil, fmt.Errorf("append() first argument must be an array, got %T", arrayArg)
		}

		// Create new array with appended items
		result := make([]interface{}, 0, len(arr)+len(itemsToAppend))
		result = append(result, arr...)
		result = append(result, itemsToAppend...)

		return result, nil
	})

	e.RegisterFunction("concat", func(args ...interface{}) (interface{}, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("concat() requires at least 2 arrays")
		}

		result := make([]interface{}, 0)

		for i, arg := range args {
			switch v := arg.(type) {
			case []interface{}:
				result = append(result, v...)
			case []string:
				for _, s := range v {
					result = append(result, s)
				}
			case []int:
				for _, n := range v {
					result = append(result, n)
				}
			default:
				return nil, fmt.Errorf("concat() argument %d must be an array, got %T", i, arg)
			}
		}

		return result, nil
	})

	// Existence checking functions
	e.RegisterFunction("has", func(args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("has() requires 1 argument: path expression")
		}

		// The argument should be a string representing a path
		// But since we're in the evaluator, we need to check if the path exists
		// This is tricky because we need to parse the path and check each level

		// For now, we'll just check if the value is nil or not
		// A better implementation would parse the path string
		return args[0] != nil, nil
	})

	e.RegisterFunction("exists", func(args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("exists() requires 1 argument: path expression")
		}

		// Same as has() - check if value is not nil
		return args[0] != nil, nil
	})

}

// compareValues compares two values numerically
func compareValues(a, b interface{}) int {
	// Try to convert to numbers
	aNum, aErr := toFloat64(a)
	bNum, bErr := toFloat64(b)

	if aErr == nil && bErr == nil {
		if aNum < bNum {
			return -1
		} else if aNum > bNum {
			return 1
		}
		return 0
	}

	// Fall back to string comparison
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)
	return strings.Compare(aStr, bStr)
}

// toFloat64 converts a value to float64
func toFloat64(v interface{}) (float64, error) {
	switch val := v.(type) {
	case float64:
		return val, nil
	case float32:
		return float64(val), nil
	case int:
		return float64(val), nil
	case int32:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case string:
		return strconv.ParseFloat(val, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", v)
	}
}

// toInt converts a value to int
func toInt(v interface{}) (int, error) {
	switch val := v.(type) {
	case int:
		return val, nil
	case int32:
		return int(val), nil
	case int64:
		return int(val), nil
	case float64:
		return int(val), nil
	case float32:
		return int(val), nil
	case string:
		i, err := strconv.ParseInt(val, 10, 64)
		return int(i), err
	default:
		return 0, fmt.Errorf("cannot convert %T to int", v)
	}
}

// performArithmetic performs arithmetic operations
func performArithmetic(left, right interface{}, operator string) (interface{}, error) {
	// Convert both operands to float64
	leftNum, err := toFloat64(left)
	if err != nil {
		return nil, fmt.Errorf("left operand: %w", err)
	}

	rightNum, err := toFloat64(right)
	if err != nil {
		return nil, fmt.Errorf("right operand: %w", err)
	}

	var result float64

	switch operator {
	case "+":
		result = leftNum + rightNum
	case "-":
		result = leftNum - rightNum
	case "*":
		result = leftNum * rightNum
	case "/":
		if rightNum == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		result = leftNum / rightNum
	case "%":
		if rightNum == 0 {
			return nil, fmt.Errorf("modulo by zero")
		}
		// For modulo, convert to int
		result = float64(int64(leftNum) % int64(rightNum))
	default:
		return nil, fmt.Errorf("unknown arithmetic operator: %s", operator)
	}

	// If the result is a whole number, return as int64
	if result == float64(int64(result)) {
		return int64(result), nil
	}

	return result, nil
}
