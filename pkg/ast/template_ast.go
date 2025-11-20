package ast

import (
	"github.com/zachaller/k8s-client-api-builder/pkg/dsl"
)

// Position represents a location in the source file
type Position struct {
	Line   int
	Column int
	File   string
}

// Node is the base interface for all AST nodes
type Node interface {
	Accept(visitor Visitor) (interface{}, error)
	Position() Position
}

// RootNode represents the root of the template AST
type RootNode struct {
	Resources []Node
	Pos       Position
}

func (n *RootNode) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitRoot(n)
}

func (n *RootNode) Position() Position {
	return n.Pos
}

// ForLoopNode represents a for loop iteration
type ForLoopNode struct {
	Variable    string          // Loop variable name (e.g., "ws")
	Iterable    *dsl.Expression // Expression to iterate over
	WhereClause *dsl.Expression // Optional filter condition
	Body        []Node          // Loop body nodes
	Pos         Position
}

func (n *ForLoopNode) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitForLoop(n)
}

func (n *ForLoopNode) Position() Position {
	return n.Pos
}

// ConditionalNode represents an if/else conditional
type ConditionalNode struct {
	Condition  *dsl.Expression // Condition expression
	ThenBranch []Node          // Nodes to execute if true
	ElseBranch []Node          // Optional nodes to execute if false
	Pos        Position
}

func (n *ConditionalNode) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitConditional(n)
}

func (n *ConditionalNode) Position() Position {
	return n.Pos
}

// ResourceNode represents a Kubernetes resource
type ResourceNode struct {
	Fields map[string]Node // Resource fields (apiVersion, kind, metadata, spec, etc.)
	Pos    Position
}

func (n *ResourceNode) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitResource(n)
}

func (n *ResourceNode) Position() Position {
	return n.Pos
}

// FieldNode represents a field with a key and value
type FieldNode struct {
	Key   string // Field key
	Value Node   // Field value (can be any node type)
	Pos   Position
}

func (n *FieldNode) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitField(n)
}

func (n *FieldNode) Position() Position {
	return n.Pos
}

// ExpressionNode wraps a DSL expression for evaluation
type ExpressionNode struct {
	Expr *dsl.Expression // The expression to evaluate
	Pos  Position
}

func (n *ExpressionNode) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitExpression(n)
}

func (n *ExpressionNode) Position() Position {
	return n.Pos
}

// LiteralNode represents a static literal value
type LiteralNode struct {
	Value interface{} // The literal value (string, number, bool, etc.)
	Pos   Position
}

func (n *LiteralNode) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitLiteral(n)
}

func (n *LiteralNode) Position() Position {
	return n.Pos
}

// ArrayNode represents an array of nodes
type ArrayNode struct {
	Elements []Node // Array elements
	Pos      Position
}

func (n *ArrayNode) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitArray(n)
}

func (n *ArrayNode) Position() Position {
	return n.Pos
}

// MapNode represents a map of key-value pairs
type MapNode struct {
	Fields map[string]Node // Map fields
	Pos    Position
}

func (n *MapNode) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitMap(n)
}

func (n *MapNode) Position() Position {
	return n.Pos
}

// MultiControlFlowNode represents multiple control flow nodes at the same level
// This is used when a map contains only @for and/or @if keys
type MultiControlFlowNode struct {
	Nodes []Node // The control flow nodes to execute
	Pos   Position
}

func (n *MultiControlFlowNode) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitMultiControlFlow(n)
}

func (n *MultiControlFlowNode) Position() Position {
	return n.Pos
}
