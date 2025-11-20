package ast

// Visitor defines the interface for visiting AST nodes
type Visitor interface {
	VisitRoot(node *RootNode) (interface{}, error)
	VisitForLoop(node *ForLoopNode) (interface{}, error)
	VisitConditional(node *ConditionalNode) (interface{}, error)
	VisitResource(node *ResourceNode) (interface{}, error)
	VisitField(node *FieldNode) (interface{}, error)
	VisitExpression(node *ExpressionNode) (interface{}, error)
	VisitLiteral(node *LiteralNode) (interface{}, error)
	VisitArray(node *ArrayNode) (interface{}, error)
	VisitMap(node *MapNode) (interface{}, error)
}

// Walk traverses an AST node and all its children
func Walk(node Node, visitor Visitor) error {
	_, err := node.Accept(visitor)
	return err
}
