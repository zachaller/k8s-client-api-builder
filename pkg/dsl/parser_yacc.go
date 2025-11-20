package dsl

import "fmt"

// ParseExpressionWithYacc parses a DSL expression using the yacc-generated parser
func ParseExpressionWithYacc(expr string) (*Expression, error) {
	lexer := NewLexer(expr)

	// Run the parser
	result := yyParse(lexer)

	if result != 0 || lexer.err != nil {
		if lexer.err != nil {
			return nil, lexer.err
		}
		return nil, fmt.Errorf("parse error")
	}

	return lexer.result, nil
}
