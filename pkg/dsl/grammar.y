%{
package dsl

import (
	"fmt"
)
%}

%union {
	expr     *Expression
	exprs    []*Expression
	str      string
	num      float64
}

%token <str> IDENTIFIER STRING
%token <num> NUMBER
%token DOT LPAREN RPAREN LBRACKET RBRACKET COMMA
%token PLUS MINUS MULTIPLY DIVIDE MODULO
%token EQ NE LT LE GT GE
%token AND OR NOT
%token TRUE FALSE

%type <expr> expression primary binary unary call array_index literal path
%type <exprs> argument_list argument_list_opt

%left OR
%left AND
%left EQ NE
%left LT LE GT GE
%left PLUS MINUS
%left MULTIPLY DIVIDE MODULO
%right NOT
%right UMINUS

%%

start:
	expression
	{
		yylex.(*Lexer).result = $1
	}
	;

expression:
	binary
	| unary
	| primary
	;

binary:
	expression PLUS expression
	{
		// Check if it's string concatenation or arithmetic
		$$ = &Expression{
			Type:     ExprBinary,
			Operator: "+",
			Left:     $1,
			Right:    $3,
		}
	}
	| expression MINUS expression
	{
		$$ = &Expression{
			Type:     ExprBinary,
			Operator: "-",
			Left:     $1,
			Right:    $3,
		}
	}
	| expression MULTIPLY expression
	{
		$$ = &Expression{
			Type:     ExprBinary,
			Operator: "*",
			Left:     $1,
			Right:    $3,
		}
	}
	| expression DIVIDE expression
	{
		$$ = &Expression{
			Type:     ExprBinary,
			Operator: "/",
			Left:     $1,
			Right:    $3,
		}
	}
	| expression MODULO expression
	{
		$$ = &Expression{
			Type:     ExprBinary,
			Operator: "%",
			Left:     $1,
			Right:    $3,
		}
	}
	| expression EQ expression
	{
		$$ = &Expression{
			Type:     ExprBinary,
			Operator: "==",
			Left:     $1,
			Right:    $3,
		}
	}
	| expression NE expression
	{
		$$ = &Expression{
			Type:     ExprBinary,
			Operator: "!=",
			Left:     $1,
			Right:    $3,
		}
	}
	| expression LT expression
	{
		$$ = &Expression{
			Type:     ExprBinary,
			Operator: "<",
			Left:     $1,
			Right:    $3,
		}
	}
	| expression LE expression
	{
		$$ = &Expression{
			Type:     ExprBinary,
			Operator: "<=",
			Left:     $1,
			Right:    $3,
		}
	}
	| expression GT expression
	{
		$$ = &Expression{
			Type:     ExprBinary,
			Operator: ">",
			Left:     $1,
			Right:    $3,
		}
	}
	| expression GE expression
	{
		$$ = &Expression{
			Type:     ExprBinary,
			Operator: ">=",
			Left:     $1,
			Right:    $3,
		}
	}
	| expression AND expression
	{
		$$ = &Expression{
			Type:     ExprBinary,
			Operator: "&&",
			Left:     $1,
			Right:    $3,
		}
	}
	| expression OR expression
	{
		$$ = &Expression{
			Type:     ExprBinary,
			Operator: "||",
			Left:     $1,
			Right:    $3,
		}
	}
	;

unary:
	NOT expression
	{
		$$ = &Expression{
			Type:     ExprUnary,
			Operator: "!",
			Operand:  $2,
		}
	}
	| MINUS expression %prec UMINUS
	{
		$$ = &Expression{
			Type:     ExprUnary,
			Operator: "-",
			Operand:  $2,
		}
	}
	;

primary:
	literal
	| path
	| call
	| array_index
	| LPAREN expression RPAREN
	{
		$$ = $2
	}
	;

path:
	DOT IDENTIFIER
	{
		$$ = &Expression{
			Type: ExprPath,
			Path: "." + $2,
		}
	}
	| path DOT IDENTIFIER
	{
		$$ = &Expression{
			Type: ExprPath,
			Path: $1.Path + "." + $3,
		}
	}
	| IDENTIFIER
	{
		$$ = &Expression{
			Type: ExprPath,
			Path: $1,
		}
	}
	| IDENTIFIER DOT IDENTIFIER
	{
		$$ = &Expression{
			Type: ExprPath,
			Path: $1 + "." + $3,
		}
	}
	;

call:
	IDENTIFIER LPAREN argument_list_opt RPAREN
	{
		args := make([]string, len($3))
		for i, expr := range $3 {
			// Convert expression back to string for compatibility
			args[i] = exprToString(expr)
		}
		$$ = &Expression{
			Type:     ExprFunction,
			Function: $1,
			Args:     args,
		}
	}
	;

array_index:
	path LBRACKET expression RBRACKET
	{
		$$ = &Expression{
			Type:  ExprArrayIndex,
			Path:  $1.Path,
			Index: $3,
		}
	}
	| IDENTIFIER LBRACKET expression RBRACKET
	{
		$$ = &Expression{
			Type:  ExprArrayIndex,
			Path:  $1,
			Index: $3,
		}
	}
	;

literal:
	STRING
	{
		$$ = &Expression{
			Type: ExprLiteral,
			Path: $1,
		}
	}
	| NUMBER
	{
		$$ = &Expression{
			Type: ExprLiteral,
			Path: fmt.Sprintf("%v", $1),
		}
	}
	| TRUE
	{
		$$ = &Expression{
			Type: ExprLiteral,
			Path: "true",
		}
	}
	| FALSE
	{
		$$ = &Expression{
			Type: ExprLiteral,
			Path: "false",
		}
	}
	;

argument_list_opt:
	/* empty */
	{
		$$ = []*Expression{}
	}
	| argument_list
	{
		$$ = $1
	}
	;

argument_list:
	expression
	{
		$$ = []*Expression{$1}
	}
	| argument_list COMMA expression
	{
		$$ = append($1, $3)
	}
	;

%%

// Helper function to convert expression to string for Args field
// This maintains compatibility with the existing Expression struct
func exprToString(expr *Expression) string {
	if expr == nil {
		return ""
	}
	
	switch expr.Type {
	case ExprLiteral:
		return expr.Path
		
	case ExprPath:
		return expr.Path
		
	case ExprBinary:
		left := exprToString(expr.Left)
		right := exprToString(expr.Right)
		return left + " " + expr.Operator + " " + right
		
	case ExprUnary:
		operand := exprToString(expr.Operand)
		return expr.Operator + operand
		
	case ExprFunction:
		args := ""
		for i, arg := range expr.Args {
			if i > 0 {
				args += ", "
			}
			args += arg
		}
		return expr.Function + "(" + args + ")"
		
	case ExprArrayIndex:
		index := exprToString(expr.Index)
		return expr.Path + "[" + index + "]"
		
	default:
		return fmt.Sprintf("<expr:%d>", expr.Type)
	}
}

