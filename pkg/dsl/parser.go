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
	Type        ExprType
	Path        string
	Index       *Expression // For array indexing
	Function    string
	Args        []string
	Operator    string
	Left        *Expression
	Right       *Expression
	Elements    []*Expression      // For concatenation
	ResourceRef *ResourceReference // For resource references
	Operand     *Expression        // For unary operations
}

// ExpressionVisitor defines the visitor interface for expressions
type ExpressionVisitor interface {
	VisitPath(expr *Expression) (interface{}, error)
	VisitFunction(expr *Expression) (interface{}, error)
	VisitBinary(expr *Expression) (interface{}, error)
	VisitLiteral(expr *Expression) (interface{}, error)
	VisitArrayIndex(expr *Expression) (interface{}, error)
	VisitConcat(expr *Expression) (interface{}, error)
	VisitResourceRef(expr *Expression) (interface{}, error)
	VisitUnary(expr *Expression) (interface{}, error)
}

// Accept allows a visitor to visit this expression
func (e *Expression) Accept(visitor ExpressionVisitor) (interface{}, error) {
	switch e.Type {
	case ExprPath:
		return visitor.VisitPath(e)
	case ExprFunction:
		return visitor.VisitFunction(e)
	case ExprBinary:
		return visitor.VisitBinary(e)
	case ExprLiteral:
		return visitor.VisitLiteral(e)
	case ExprArrayIndex:
		return visitor.VisitArrayIndex(e)
	case ExprConcat:
		return visitor.VisitConcat(e)
	case ExprResourceRef:
		return visitor.VisitResourceRef(e)
	case ExprUnary:
		return visitor.VisitUnary(e)
	default:
		return nil, fmt.Errorf("unknown expression type: %d", e.Type)
	}
}

// Walk traverses an expression tree with a visitor
func Walk(expr *Expression, visitor ExpressionVisitor) error {
	_, err := expr.Accept(visitor)
	return err
}

// ResourceReference represents a reference to another resource
type ResourceReference struct {
	APIVersion string
	Kind       string
	Name       *Expression // Name can be an expression
	FieldPath  string
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
	ExprResourceRef
	ExprUnary
)

// ParseExpression parses a DSL expression
// Uses yacc parser for standard expressions, legacy parser only for resource()
func ParseExpression(expr string) (*Expression, error) {
	expr = strings.TrimSpace(expr)

	// Special case: resource() function requires custom parsing
	// because it has special syntax: resource(...).field.path
	if strings.HasPrefix(expr, "resource(") {
		return parseResourceRef(expr)
	}

	// Use yacc parser for all other expressions
	return ParseExpressionWithYacc(expr)
}

// ParseForLoop parses a for loop expression like "item in .spec.items" or "port in container.ports"
func ParseForLoop(expr string) (varName string, iterPath string, err error) {
	parts := strings.Split(expr, " in ")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid for loop expression: %s (expected 'var in path')", expr)
	}

	varName = strings.TrimSpace(parts[0])
	iterPath = strings.TrimSpace(parts[1])

	// Iteration path can start with '.' (root path) or be a loop variable reference
	// Examples: ".spec.items" or "container.ports"
	if !strings.HasPrefix(iterPath, ".") && !isIdentifier(strings.Split(iterPath, ".")[0]) {
		return "", "", fmt.Errorf("iteration path must start with '.' or be a variable reference: %s", iterPath)
	}

	return varName, iterPath, nil
}

// ParseForLoopWithFilter parses a for loop expression with optional where clause
// Supports: "item in .path" or "item in .path where item.field != value"
func ParseForLoopWithFilter(expr string) (varName string, iterPath string, filterExpr string, err error) {
	// Check for "where" clause
	whereIndex := strings.Index(expr, " where ")
	if whereIndex > 0 {
		// Split into loop part and filter part
		loopPart := strings.TrimSpace(expr[:whereIndex])
		filterExpr = strings.TrimSpace(expr[whereIndex+7:]) // +7 for " where "

		// Parse the loop part
		varName, iterPath, err = ParseForLoop(loopPart)
		return varName, iterPath, filterExpr, err
	}

	// No where clause, use regular parsing
	varName, iterPath, err = ParseForLoop(expr)
	return varName, iterPath, "", err
}

// parseResourceRef parses a resource reference like resource("v1", "Service", "my-app").spec.clusterIP
func parseResourceRef(expr string) (*Expression, error) {
	// Find the closing parenthesis of resource()
	depth := 0
	closeParen := -1
	for i, ch := range expr {
		if ch == '(' {
			depth++
		} else if ch == ')' {
			depth--
			if depth == 0 {
				closeParen = i
				break
			}
		}
	}

	if closeParen == -1 {
		return nil, fmt.Errorf("invalid resource reference: missing closing parenthesis")
	}

	// Extract arguments: resource(args)
	argsStr := expr[len("resource("):closeParen]

	// Extract field path after the function call
	fieldPath := ""
	if closeParen+1 < len(expr) {
		remainder := expr[closeParen+1:]
		if strings.HasPrefix(remainder, ".") {
			fieldPath = remainder[1:] // Remove leading dot
		} else if remainder != "" {
			return nil, fmt.Errorf("invalid resource reference: expected '.' after resource(), got %s", remainder)
		}
	}

	// Parse arguments: "apiVersion", "kind", name_expression
	args, err := parseResourceRefArgs(argsStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse resource reference arguments: %w", err)
	}

	if len(args) != 3 {
		return nil, fmt.Errorf("resource() requires 3 arguments (apiVersion, kind, name), got %d", len(args))
	}

	// Parse the name argument (could be an expression)
	nameExpr, err := ParseExpression(args[2])
	if err != nil {
		return nil, fmt.Errorf("failed to parse resource name: %w", err)
	}

	return &Expression{
		Type: ExprResourceRef,
		ResourceRef: &ResourceReference{
			APIVersion: strings.Trim(args[0], "\""),
			Kind:       strings.Trim(args[1], "\""),
			Name:       nameExpr,
			FieldPath:  fieldPath,
		},
	}, nil
}

// parseResourceRefArgs parses the arguments of resource() function
// Handles quoted strings and expressions
func parseResourceRefArgs(argsStr string) ([]string, error) {
	args := []string{}
	current := ""
	inDoubleQuotes := false
	inSingleQuotes := false
	depth := 0

	for i := 0; i < len(argsStr); i++ {
		ch := argsStr[i]

		switch ch {
		case '"':
			if !inSingleQuotes {
				inDoubleQuotes = !inDoubleQuotes
			}
			current += string(ch)
		case '\'':
			if !inDoubleQuotes {
				inSingleQuotes = !inSingleQuotes
			}
			current += string(ch)
		case '(':
			depth++
			current += string(ch)
		case ')':
			depth--
			current += string(ch)
		case ',':
			if !inDoubleQuotes && !inSingleQuotes && depth == 0 {
				// End of argument
				args = append(args, strings.TrimSpace(current))
				current = ""
			} else {
				current += string(ch)
			}
		default:
			current += string(ch)
		}
	}

	// Add last argument
	if current != "" {
		args = append(args, strings.TrimSpace(current))
	}

	return args, nil
}

// isIdentifier checks if a string is a valid identifier
func isIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}
	// Must start with a letter or underscore
	firstCh := rune(s[0])
	if !((firstCh >= 'a' && firstCh <= 'z') || (firstCh >= 'A' && firstCh <= 'Z') || firstCh == '_') {
		return false
	}
	// Rest can be letters, numbers, underscores, or hyphens
	for _, ch := range s {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '-') {
			return false
		}
	}
	return true
}
