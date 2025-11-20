package ast

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/zachaller/k8s-client-api-builder/pkg/dsl"
)

// Parser parses YAML data into an AST
type Parser struct {
	currentFile string
	currentLine int
}

// NewParser creates a new template parser
func NewParser() *Parser {
	return &Parser{
		currentFile: "<template>",
		currentLine: 0,
	}
}

// ParseTemplate parses a YAML template into an AST
// The yamlData should be the parsed "resources" field from the template
func ParseTemplate(yamlData interface{}) (*RootNode, error) {
	parser := NewParser()
	return parser.parseRoot(yamlData)
}

// parseRoot parses the root resources node
func (p *Parser) parseRoot(data interface{}) (*RootNode, error) {
	root := &RootNode{
		Resources: []Node{},
		Pos:       p.currentPos(),
	}

	// Handle different possible formats for resources
	switch v := data.(type) {
	case []interface{}:
		// Array of resources
		for _, item := range v {
			node, err := p.parseNode(item)
			if err != nil {
				return nil, err
			}
			root.Resources = append(root.Resources, node)
		}
	case map[string]interface{}:
		// Map with potential control flow keys
		node, err := p.parseNode(v)
		if err != nil {
			return nil, err
		}
		root.Resources = append(root.Resources, node)
	default:
		return nil, fmt.Errorf("unexpected root type: %T", data)
	}

	return root, nil
}

// parseNode parses any node in the AST
func (p *Parser) parseNode(data interface{}) (Node, error) {
	switch v := data.(type) {
	case string:
		// Check if it's an @expr(...) expression
		if strings.HasPrefix(v, "@expr(") && strings.HasSuffix(v, ")") {
			return p.parseExpressionNode(v)
		}
		// Otherwise, it's a literal string
		return &LiteralNode{Value: v, Pos: p.currentPos()}, nil

	case map[string]interface{}:
		// Count control flow keys and regular keys
		controlFlowCount := 0
		regularKeyCount := 0
		var singleControlKey string
		var singleControlValue interface{}

		for key, value := range v {
			if strings.HasPrefix(key, "@for(") || strings.HasPrefix(key, "@if(") {
				controlFlowCount++
				singleControlKey = key
				singleControlValue = value
			} else {
				regularKeyCount++
			}
		}

		// If we have MULTIPLE control flow keys AND no regular keys, treat as a special container
		if controlFlowCount > 1 && regularKeyCount == 0 {
			// Parse all control flow nodes and return a container
			nodes := []Node{}
			for key, value := range v {
				if strings.HasPrefix(key, "@for(") {
					node, err := p.parseForLoop(key, value)
					if err != nil {
						return nil, err
					}
					nodes = append(nodes, node)
				} else if strings.HasPrefix(key, "@if(") {
					node, err := p.parseConditional(key, value)
					if err != nil {
						return nil, err
					}
					nodes = append(nodes, node)
				}
			}
			// Return a special container node that will execute all control flows
			return &MultiControlFlowNode{
				Nodes: nodes,
				Pos:   p.currentPos(),
			}, nil
		}

		// Single control flow key AND no regular keys (backward compatibility)
		if controlFlowCount == 1 && regularKeyCount == 0 {
			if strings.HasPrefix(singleControlKey, "@for(") {
				return p.parseForLoop(singleControlKey, singleControlValue)
			}
			if strings.HasPrefix(singleControlKey, "@if(") {
				return p.parseConditional(singleControlKey, singleControlValue)
			}
		}

		// Regular map node (includes maps with control flow keys mixed with regular keys)
		return p.parseMapNode(v)

	case []interface{}:
		// Array node
		return p.parseArrayNode(v)

	case int, int32, int64, float32, float64, bool:
		// Primitive literals
		return &LiteralNode{Value: v, Pos: p.currentPos()}, nil

	case nil:
		return &LiteralNode{Value: nil, Pos: p.currentPos()}, nil

	default:
		return nil, fmt.Errorf("unexpected node type: %T", data)
	}
}

// parseForLoop parses a @for(...) control structure
func (p *Parser) parseForLoop(key string, value interface{}) (*ForLoopNode, error) {
	// Extract expression from @for(...)
	if !strings.HasPrefix(key, "@for(") || !strings.HasSuffix(key, ")") && !strings.HasSuffix(key, "):") {
		return nil, fmt.Errorf("invalid @for syntax: %s", key)
	}

	// Remove @for( prefix and ) or ): suffix
	exprStr := key[5:] // Remove "@for("
	if strings.HasSuffix(exprStr, "):") {
		exprStr = exprStr[:len(exprStr)-2]
	} else if strings.HasSuffix(exprStr, ")") {
		exprStr = exprStr[:len(exprStr)-1]
	}

	// Parse the for loop expression (e.g., "ws in .spec.webservices where ws.disabled != true")
	varName, iterPath, filterExpr, err := dsl.ParseForLoopWithFilter(exprStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse for loop expression: %w", err)
	}

	// Parse the iterable expression
	iterExpr, err := dsl.ParseExpression(iterPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse iterable expression: %w", err)
	}

	// Parse the where clause if present
	var whereExpr *dsl.Expression
	if filterExpr != "" {
		whereExpr, err = dsl.ParseExpression(filterExpr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse where clause: %w", err)
		}
	}

	// Parse the body
	var body []Node
	switch bodyValue := value.(type) {
	case []interface{}:
		for _, item := range bodyValue {
			node, err := p.parseNode(item)
			if err != nil {
				return nil, err
			}
			body = append(body, node)
		}
	case map[string]interface{}:
		node, err := p.parseNode(bodyValue)
		if err != nil {
			return nil, err
		}
		body = append(body, node)
	default:
		return nil, fmt.Errorf("invalid for loop body type: %T", value)
	}

	return &ForLoopNode{
		Variable:    varName,
		Iterable:    iterExpr,
		WhereClause: whereExpr,
		Body:        body,
		Pos:         p.currentPos(),
	}, nil
}

// parseConditional parses a @if(...) control structure
func (p *Parser) parseConditional(key string, value interface{}) (*ConditionalNode, error) {
	// Extract expression from @if(...)
	if !strings.HasPrefix(key, "@if(") || !strings.HasSuffix(key, ")") && !strings.HasSuffix(key, "):") {
		return nil, fmt.Errorf("invalid @if syntax: %s", key)
	}

	// Remove @if( prefix and ) or ): suffix
	exprStr := key[4:] // Remove "@if("
	if strings.HasSuffix(exprStr, "):") {
		exprStr = exprStr[:len(exprStr)-2]
	} else if strings.HasSuffix(exprStr, ")") {
		exprStr = exprStr[:len(exprStr)-1]
	}

	// Parse the condition expression
	condExpr, err := dsl.ParseExpression(exprStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse condition expression: %w", err)
	}

	// Parse the then branch
	var thenBranch []Node
	switch thenValue := value.(type) {
	case []interface{}:
		for _, item := range thenValue {
			node, err := p.parseNode(item)
			if err != nil {
				return nil, err
			}
			thenBranch = append(thenBranch, node)
		}
	case map[string]interface{}:
		node, err := p.parseNode(thenValue)
		if err != nil {
			return nil, err
		}
		thenBranch = append(thenBranch, node)
	default:
		return nil, fmt.Errorf("invalid if branch type: %T", value)
	}

	return &ConditionalNode{
		Condition:  condExpr,
		ThenBranch: thenBranch,
		ElseBranch: []Node{}, // TODO: Support else branches
		Pos:        p.currentPos(),
	}, nil
}

// parseExpressionNode parses an @expr(...) expression
func (p *Parser) parseExpressionNode(exprStr string) (*ExpressionNode, error) {
	// Remove @expr( prefix and ) suffix
	if !strings.HasPrefix(exprStr, "@expr(") || !strings.HasSuffix(exprStr, ")") {
		return nil, fmt.Errorf("invalid @expr syntax: %s", exprStr)
	}

	inner := exprStr[6 : len(exprStr)-1] // Remove "@expr(" and ")"

	// Parse the expression
	expr, err := dsl.ParseExpression(inner)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expression: %w", err)
	}

	return &ExpressionNode{
		Expr: expr,
		Pos:  p.currentPos(),
	}, nil
}

// parseMapNode parses a regular map (not a control structure)
func (p *Parser) parseMapNode(data map[string]interface{}) (*MapNode, error) {
	fields := make(map[string]Node)

	for key, value := range data {
		// Check if the key itself is a control structure
		if strings.HasPrefix(key, "@for(") {
			// This is a for loop that should add fields to the parent map
			forNode, err := p.parseForLoop(key, value)
			if err != nil {
				return nil, err
			}
			// For loops in maps need special handling - store as a special field
			fields[key] = forNode
			continue
		}
		if strings.HasPrefix(key, "@if(") {
			// This is a conditional that should add fields to the parent map
			ifNode, err := p.parseConditional(key, value)
			if err != nil {
				return nil, err
			}
			// Conditionals in maps need special handling
			fields[key] = ifNode
			continue
		}

		// Regular field
		node, err := p.parseNode(value)
		if err != nil {
			return nil, fmt.Errorf("failed to parse field %s: %w", key, err)
		}
		fields[key] = node
	}

	return &MapNode{
		Fields: fields,
		Pos:    p.currentPos(),
	}, nil
}

// parseArrayNode parses an array
func (p *Parser) parseArrayNode(data []interface{}) (*ArrayNode, error) {
	elements := make([]Node, 0, len(data))

	for _, item := range data {
		node, err := p.parseNode(item)
		if err != nil {
			return nil, err
		}
		elements = append(elements, node)
	}

	return &ArrayNode{
		Elements: elements,
		Pos:      p.currentPos(),
	}, nil
}

// detectControlFlow checks if a string is a control flow marker
func detectControlFlow(key string) (nodeType string, expr string, ok bool) {
	// Check for @for(...)
	forPattern := regexp.MustCompile(`^@for\((.+)\):?$`)
	if match := forPattern.FindStringSubmatch(key); match != nil {
		return "for", match[1], true
	}

	// Check for @if(...)
	ifPattern := regexp.MustCompile(`^@if\((.+)\):?$`)
	if match := ifPattern.FindStringSubmatch(key); match != nil {
		return "if", match[1], true
	}

	return "", "", false
}

// currentPos returns the current position in the file
func (p *Parser) currentPos() Position {
	return Position{
		Line:   p.currentLine,
		Column: 0,
		File:   p.currentFile,
	}
}
