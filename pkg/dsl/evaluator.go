package dsl

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// Evaluator evaluates DSL expressions against data
type Evaluator struct {
	data      interface{}
	functions map[string]Function
}

// Function represents a DSL function
type Function func(args ...interface{}) (interface{}, error)

// NewEvaluator creates a new evaluator with the given data
func NewEvaluator(data interface{}) *Evaluator {
	e := &Evaluator{
		data:      data,
		functions: make(map[string]Function),
	}
	e.registerBuiltinFunctions()
	return e
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
	default:
		return nil, fmt.Errorf("unknown expression type: %d", expr.Type)
	}
}

// EvaluateString evaluates a string that may contain variable substitutions
func (e *Evaluator) EvaluateString(input string) (string, error) {
	// Pattern to match $(...) expressions
	pattern := regexp.MustCompile(`\$\(([^)]+)\)`)
	
	result := input
	matches := pattern.FindAllStringSubmatch(input, -1)
	
	for _, match := range matches {
		fullMatch := match[0]
		exprStr := match[1]
		
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

// evaluatePath evaluates a path expression like ".spec.name"
func (e *Evaluator) evaluatePath(path string) (interface{}, error) {
	if !strings.HasPrefix(path, ".") {
		return nil, fmt.Errorf("path must start with '.': %s", path)
	}
	
	// Remove leading dot
	path = path[1:]
	
	// Split path into parts
	parts := strings.Split(path, ".")
	
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
		return nil, err
	}
	
	right, err := e.Evaluate(expr.Right)
	if err != nil {
		return nil, err
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

// evaluateArrayIndex evaluates array indexing expressions
func (e *Evaluator) evaluateArrayIndex(expr *Expression) (interface{}, error) {
	// Evaluate the base path to get the array
	baseValue, err := e.evaluatePath(expr.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate array path: %w", err)
	}
	
	// Evaluate the index expression
	indexValue, err := e.Evaluate(expr.Index)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate array index: %w", err)
	}
	
	// Convert index to int
	index, err := toInt(indexValue)
	if err != nil {
		return nil, fmt.Errorf("array index must be an integer: %w", err)
	}
	
	// Access the array element
	val := reflect.ValueOf(baseValue)
	
	// Handle different collection types
	switch val.Kind() {
	case reflect.Slice, reflect.Array:
		if index < 0 || index >= val.Len() {
			return nil, fmt.Errorf("array index %d out of bounds (length %d)", index, val.Len())
		}
		return val.Index(index).Interface(), nil
	
	case reflect.Map:
		// For maps, try to use the index as a key
		keyVal := reflect.ValueOf(indexValue)
		mapVal := val.MapIndex(keyVal)
		if !mapVal.IsValid() {
			return nil, fmt.Errorf("key %v not found in map", indexValue)
		}
		return mapVal.Interface(), nil
	
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
	
	// Remove quotes if present
	if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
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

