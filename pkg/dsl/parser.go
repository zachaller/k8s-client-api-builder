package dsl

import (
	"fmt"
	"regexp"
	"strings"
)

// Token types for DSL parsing
type TokenType int

const (
	TokenText TokenType = iota
	TokenVariable
	TokenIf
	TokenFor
	TokenEndBlock
	TokenFunction
)

// Token represents a parsed token
type Token struct {
	Type  TokenType
	Value string
	Line  int
	Col   int
}

// Parser handles DSL parsing
type Parser struct {
	input  string
	tokens []Token
	pos    int
}

// NewParser creates a new DSL parser
func NewParser(input string) *Parser {
	return &Parser{
		input:  input,
		tokens: []Token{},
		pos:    0,
	}
}

// Parse tokenizes the input
func (p *Parser) Parse() ([]Token, error) {
	// Regular expressions for matching DSL constructs
	varPattern := regexp.MustCompile(`\$\(([^)]+)\)`)
	ifPattern := regexp.MustCompile(`\$if\(([^)]+)\):`)
	forPattern := regexp.MustCompile(`\$for\(([^)]+)\):`)
	
	lines := strings.Split(p.input, "\n")
	
	for lineNum, line := range lines {
		// Check for control structures first
		if ifMatch := ifPattern.FindStringSubmatch(line); ifMatch != nil {
			p.tokens = append(p.tokens, Token{
				Type:  TokenIf,
				Value: ifMatch[1],
				Line:  lineNum,
			})
			continue
		}
		
		if forMatch := forPattern.FindStringSubmatch(line); forMatch != nil {
			p.tokens = append(p.tokens, Token{
				Type:  TokenFor,
				Value: forMatch[1],
				Line:  lineNum,
			})
			continue
		}
		
		// Process variable substitutions in the line
		if varPattern.MatchString(line) {
			// Line contains variables
			p.tokens = append(p.tokens, Token{
				Type:  TokenText,
				Value: line,
				Line:  lineNum,
			})
		} else {
			// Plain text line
			p.tokens = append(p.tokens, Token{
				Type:  TokenText,
				Value: line,
				Line:  lineNum,
			})
		}
	}
	
	return p.tokens, nil
}

// Expression represents a parsed DSL expression
type Expression struct {
	Type     ExprType
	Path     string
	Index    *Expression  // For array indexing
	Function string
	Args     []string
	Operator string
	Left     *Expression
	Right    *Expression
	Elements []*Expression // For concatenation
}

// ExprType represents the type of expression
type ExprType int

const (
	ExprPath ExprType = iota
	ExprFunction
	ExprBinary
	ExprLiteral
	ExprArrayIndex
	ExprConcat
)

// ParseExpression parses a DSL expression like ".spec.name" or "lower(.metadata.name)"
func ParseExpression(expr string) (*Expression, error) {
	expr = strings.TrimSpace(expr)
	
	// Check for string concatenation first (but only if it contains a literal string or multiple + operators)
	// This prevents infinite recursion by only treating it as concat if there are multiple elements
	if strings.Count(expr, "+") > 0 && (strings.Contains(expr, "\"") || hasMultiplePlusOutsideParens(expr)) {
		if concatExpr, ok := tryParseConcatExpr(expr); ok {
			return concatExpr, nil
		}
	}
	
	// Check for arithmetic operations
	for _, op := range []string{"-", "*", "/", "%"} {
		if containsOperatorOutsideParens(expr, op) {
			return parseBinaryExpr(expr, op)
		}
	}
	
	// Check for + operator (could be arithmetic if no quotes)
	if containsOperatorOutsideParens(expr, "+") && !strings.Contains(expr, "\"") {
		return parseBinaryExpr(expr, "+")
	}
	
	// Check if it's a function call (but not array indexing or parenthesized expression)
	if strings.Contains(expr, "(") && strings.HasSuffix(expr, ")") && !strings.Contains(expr, "[") {
		// Check if it's a parenthesized expression (starts with '(')
		if strings.HasPrefix(expr, "(") {
			// It's a parenthesized expression, parse the inner part
			inner := expr[1 : len(expr)-1]
			return ParseExpression(inner)
		}
		return parseFunctionExpr(expr)
	}
	
	// Check for array indexing
	if strings.Contains(expr, "[") && strings.Contains(expr, "]") {
		return parseArrayIndexExpr(expr)
	}
	
	// Check if it's a comparison expression (for conditionals)
	for _, op := range []string{"==", "!=", ">=", "<=", ">", "<"} {
		if containsOperatorOutsideParens(expr, op) {
			return parseBinaryExpr(expr, op)
		}
	}
	
	// It's a simple path expression
	if strings.HasPrefix(expr, ".") {
		return &Expression{
			Type: ExprPath,
			Path: expr,
		}, nil
	}
	
	// It's a literal
	return &Expression{
		Type: ExprLiteral,
		Path: expr,
	}, nil
}

// hasMultiplePlusOutsideParens checks if there are multiple + operators outside parens
func hasMultiplePlusOutsideParens(expr string) bool {
	count := 0
	depth := 0
	inQuotes := false
	
	for i := 0; i < len(expr); i++ {
		ch := expr[i]
		
		if ch == '"' {
			inQuotes = !inQuotes
			continue
		}
		
		if inQuotes {
			continue
		}
		
		if ch == '(' {
			depth++
		} else if ch == ')' {
			depth--
		} else if ch == '+' && depth == 0 {
			count++
			if count > 1 {
				return true
			}
		}
	}
	
	return count > 1
}

func parseFunctionExpr(expr string) (*Expression, error) {
	openParen := strings.Index(expr, "(")
	if openParen == -1 {
		return nil, fmt.Errorf("invalid function expression: %s", expr)
	}
	
	funcName := strings.TrimSpace(expr[:openParen])
	argsStr := strings.TrimSpace(expr[openParen+1 : len(expr)-1])
	
	var args []string
	if argsStr != "" {
		// Simple argument parsing (doesn't handle nested functions yet)
		args = []string{argsStr}
	}
	
	return &Expression{
		Type:     ExprFunction,
		Function: funcName,
		Args:     args,
	}, nil
}

func parseBinaryExpr(expr string, operator string) (*Expression, error) {
	// Find the operator position outside of parentheses
	pos := findOperatorPosition(expr, operator)
	if pos == -1 {
		return nil, fmt.Errorf("operator %s not found in expression: %s", operator, expr)
	}
	
	leftStr := strings.TrimSpace(expr[:pos])
	rightStr := strings.TrimSpace(expr[pos+len(operator):])
	
	left, err := ParseExpression(leftStr)
	if err != nil {
		return nil, err
	}
	
	right, err := ParseExpression(rightStr)
	if err != nil {
		return nil, err
	}
	
	return &Expression{
		Type:     ExprBinary,
		Operator: operator,
		Left:     left,
		Right:    right,
	}, nil
}

// parseArrayIndexExpr parses array indexing like ".spec.items[0]" or ".spec.items[.spec.index]"
func parseArrayIndexExpr(expr string) (*Expression, error) {
	openBracket := strings.Index(expr, "[")
	closeBracket := strings.LastIndex(expr, "]")
	
	if openBracket == -1 || closeBracket == -1 || closeBracket < openBracket {
		return nil, fmt.Errorf("invalid array index expression: %s", expr)
	}
	
	basePath := strings.TrimSpace(expr[:openBracket])
	indexStr := strings.TrimSpace(expr[openBracket+1 : closeBracket])
	
	// Parse the index expression (could be a number or a path)
	indexExpr, err := ParseExpression(indexStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse array index: %w", err)
	}
	
	return &Expression{
		Type:  ExprArrayIndex,
		Path:  basePath,
		Index: indexExpr,
	}, nil
}

// tryParseConcatExpr attempts to parse a concatenation expression
func tryParseConcatExpr(expr string) (*Expression, bool) {
	// Look for + operator outside of parentheses and quotes
	// This is for string concatenation like: .spec.prefix + "-" + .spec.suffix
	
	elements := []*Expression{}
	current := ""
	depth := 0
	inQuotes := false
	
	for i := 0; i < len(expr); i++ {
		ch := expr[i]
		
		switch ch {
		case '"':
			inQuotes = !inQuotes
			current += string(ch)
		case '(':
			depth++
			current += string(ch)
		case ')':
			depth--
			current += string(ch)
		case '+':
			if depth == 0 && !inQuotes {
				// This is a concatenation operator
				if current != "" {
					elem, err := parseNonConcatExpression(strings.TrimSpace(current))
					if err != nil {
						return nil, false
					}
					elements = append(elements, elem)
					current = ""
				}
			} else {
				current += string(ch)
			}
		default:
			current += string(ch)
		}
	}
	
	// Add the last element
	if current != "" {
		elem, err := parseNonConcatExpression(strings.TrimSpace(current))
		if err != nil {
			return nil, false
		}
		elements = append(elements, elem)
	}
	
	// If we found multiple elements, it's a concatenation
	if len(elements) > 1 {
		return &Expression{
			Type:     ExprConcat,
			Elements: elements,
		}, true
	}
	
	return nil, false
}

// parseNonConcatExpression parses an expression without checking for concatenation
// This prevents infinite recursion in concat parsing
func parseNonConcatExpression(expr string) (*Expression, error) {
	expr = strings.TrimSpace(expr)
	
	// Check for arithmetic operations (but not +, that's handled by concat)
	for _, op := range []string{"-", "*", "/", "%"} {
		if containsOperatorOutsideParens(expr, op) {
			return parseBinaryExpr(expr, op)
		}
	}
	
	// Check if it's a function call (but not a parenthesized expression)
	if strings.Contains(expr, "(") && strings.HasSuffix(expr, ")") && !strings.Contains(expr, "[") {
		// Check if it's a parenthesized expression (starts with '(')
		if strings.HasPrefix(expr, "(") {
			// It's a parenthesized expression, parse the inner part
			inner := expr[1 : len(expr)-1]
			return parseNonConcatExpression(inner)
		}
		return parseFunctionExpr(expr)
	}
	
	// Check for array indexing
	if strings.Contains(expr, "[") && strings.Contains(expr, "]") {
		return parseArrayIndexExpr(expr)
	}
	
	// Check if it's a comparison expression
	for _, op := range []string{"==", "!=", ">=", "<=", ">", "<"} {
		if containsOperatorOutsideParens(expr, op) {
			return parseBinaryExpr(expr, op)
		}
	}
	
	// It's a simple path expression
	if strings.HasPrefix(expr, ".") {
		return &Expression{
			Type: ExprPath,
			Path: expr,
		}, nil
	}
	
	// It's a literal
	return &Expression{
		Type: ExprLiteral,
		Path: expr,
	}, nil
}

// containsOperatorOutsideParens checks if an operator exists outside of parentheses
func containsOperatorOutsideParens(expr string, operator string) bool {
	return findOperatorPosition(expr, operator) != -1
}

// findOperatorPosition finds the position of an operator outside parentheses
func findOperatorPosition(expr string, operator string) int {
	depth := 0
	inQuotes := false
	
	for i := 0; i < len(expr); i++ {
		ch := expr[i]
		
		if ch == '"' {
			inQuotes = !inQuotes
			continue
		}
		
		if inQuotes {
			continue
		}
		
		if ch == '(' {
			depth++
		} else if ch == ')' {
			depth--
		} else if depth == 0 {
			// Check if this position matches the operator
			if i+len(operator) <= len(expr) && expr[i:i+len(operator)] == operator {
				// Make sure it's not part of a comparison operator
				// e.g., don't match "=" in "=="
				if operator == "=" || operator == "!" || operator == ">" || operator == "<" {
					if i+1 < len(expr) && expr[i+1] == '=' {
						continue
					}
				}
				return i
			}
		}
	}
	
	return -1
}

// ParseForLoop parses a for loop expression like "item in .spec.items"
func ParseForLoop(expr string) (varName string, iterPath string, err error) {
	parts := strings.Split(expr, " in ")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid for loop expression: %s (expected 'var in path')", expr)
	}
	
	varName = strings.TrimSpace(parts[0])
	iterPath = strings.TrimSpace(parts[1])
	
	if !strings.HasPrefix(iterPath, ".") {
		return "", "", fmt.Errorf("iteration path must start with '.': %s", iterPath)
	}
	
	return varName, iterPath, nil
}

