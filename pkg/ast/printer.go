package ast

import (
	"fmt"
	"strings"
)

// Printer is a visitor that prints the AST for debugging
type Printer struct {
	indent int
	output strings.Builder
}

// NewPrinter creates a new AST printer
func NewPrinter() *Printer {
	return &Printer{indent: 0}
}

// Print prints the AST and returns the string representation
func (p *Printer) Print(node Node) (string, error) {
	p.output.Reset()
	_, err := node.Accept(p)
	if err != nil {
		return "", err
	}
	return p.output.String(), nil
}

func (p *Printer) writeIndent() {
	p.output.WriteString(strings.Repeat("  ", p.indent))
}

func (p *Printer) VisitRoot(node *RootNode) (interface{}, error) {
	p.writeIndent()
	p.output.WriteString("RootNode:\n")
	p.indent++
	for i, resource := range node.Resources {
		p.writeIndent()
		p.output.WriteString(fmt.Sprintf("Resource[%d]:\n", i))
		p.indent++
		resource.Accept(p)
		p.indent--
	}
	p.indent--
	return nil, nil
}

func (p *Printer) VisitForLoop(node *ForLoopNode) (interface{}, error) {
	p.writeIndent()
	p.output.WriteString(fmt.Sprintf("ForLoopNode(var=%s, iterable=%v", node.Variable, node.Iterable))
	if node.WhereClause != nil {
		p.output.WriteString(fmt.Sprintf(", where=%v", node.WhereClause))
	}
	p.output.WriteString("):\n")

	p.indent++
	for _, child := range node.Body {
		child.Accept(p)
	}
	p.indent--
	return nil, nil
}

func (p *Printer) VisitConditional(node *ConditionalNode) (interface{}, error) {
	p.writeIndent()
	p.output.WriteString(fmt.Sprintf("ConditionalNode(condition=%v):\n", node.Condition))

	p.indent++
	p.writeIndent()
	p.output.WriteString("Then:\n")
	p.indent++
	for _, child := range node.ThenBranch {
		child.Accept(p)
	}
	p.indent--

	if len(node.ElseBranch) > 0 {
		p.writeIndent()
		p.output.WriteString("Else:\n")
		p.indent++
		for _, child := range node.ElseBranch {
			child.Accept(p)
		}
		p.indent--
	}
	p.indent--
	return nil, nil
}

func (p *Printer) VisitResource(node *ResourceNode) (interface{}, error) {
	p.writeIndent()
	p.output.WriteString("ResourceNode:\n")
	p.indent++
	for key, value := range node.Fields {
		p.writeIndent()
		p.output.WriteString(fmt.Sprintf("%s:\n", key))
		p.indent++
		value.Accept(p)
		p.indent--
	}
	p.indent--
	return nil, nil
}

func (p *Printer) VisitField(node *FieldNode) (interface{}, error) {
	p.writeIndent()
	p.output.WriteString(fmt.Sprintf("FieldNode(key=%s):\n", node.Key))
	p.indent++
	node.Value.Accept(p)
	p.indent--
	return nil, nil
}

func (p *Printer) VisitExpression(node *ExpressionNode) (interface{}, error) {
	p.writeIndent()
	p.output.WriteString(fmt.Sprintf("ExpressionNode(%v)\n", node.Expr))
	return nil, nil
}

func (p *Printer) VisitLiteral(node *LiteralNode) (interface{}, error) {
	p.writeIndent()
	p.output.WriteString(fmt.Sprintf("LiteralNode(%v)\n", node.Value))
	return nil, nil
}

func (p *Printer) VisitArray(node *ArrayNode) (interface{}, error) {
	p.writeIndent()
	p.output.WriteString("ArrayNode:\n")
	p.indent++
	for i, elem := range node.Elements {
		p.writeIndent()
		p.output.WriteString(fmt.Sprintf("[%d]:\n", i))
		p.indent++
		elem.Accept(p)
		p.indent--
	}
	p.indent--
	return nil, nil
}

func (p *Printer) VisitMap(node *MapNode) (interface{}, error) {
	p.writeIndent()
	p.output.WriteString("MapNode:\n")
	p.indent++
	for key, value := range node.Fields {
		p.writeIndent()
		p.output.WriteString(fmt.Sprintf("%s:\n", key))
		p.indent++
		value.Accept(p)
		p.indent--
	}
	p.indent--
	return nil, nil
}

func (p *Printer) VisitMultiControlFlow(node *MultiControlFlowNode) (interface{}, error) {
	p.writeIndent()
	p.output.WriteString("MultiControlFlowNode:\n")
	p.indent++
	for i, controlNode := range node.Nodes {
		p.writeIndent()
		p.output.WriteString(fmt.Sprintf("Control[%d]:\n", i))
		p.indent++
		controlNode.Accept(p)
		p.indent--
	}
	p.indent--
	return nil, nil
}
