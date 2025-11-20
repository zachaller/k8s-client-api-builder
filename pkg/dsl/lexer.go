package dsl

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Lexer tokenizes DSL expressions
type Lexer struct {
	input  string
	pos    int
	result *Expression
	err    error
}

// NewLexer creates a new lexer
func NewLexer(input string) *Lexer {
	return &Lexer{
		input: strings.TrimSpace(input),
		pos:   0,
	}
}

// Lex returns the next token for the parser
func (l *Lexer) Lex(lval *yySymType) int {
	// Skip whitespace
	for l.pos < len(l.input) && unicode.IsSpace(rune(l.input[l.pos])) {
		l.pos++
	}

	if l.pos >= len(l.input) {
		return 0 // EOF
	}

	ch := l.input[l.pos]

	// Single character tokens
	switch ch {
	case '.':
		l.pos++
		return DOT
	case '(':
		l.pos++
		return LPAREN
	case ')':
		l.pos++
		return RPAREN
	case '[':
		l.pos++
		return LBRACKET
	case ']':
		l.pos++
		return RBRACKET
	case ',':
		l.pos++
		return COMMA
	case '+':
		l.pos++
		return PLUS
	case '-':
		// Could be minus operator or negative number
		if l.pos+1 < len(l.input) && unicode.IsDigit(rune(l.input[l.pos+1])) {
			return l.lexNumber(lval)
		}
		l.pos++
		return MINUS
	case '*':
		l.pos++
		return MULTIPLY
	case '/':
		l.pos++
		return DIVIDE
	case '%':
		l.pos++
		return MODULO
	case '<':
		if l.pos+1 < len(l.input) && l.input[l.pos+1] == '=' {
			l.pos += 2
			return LE
		}
		l.pos++
		return LT
	case '>':
		if l.pos+1 < len(l.input) && l.input[l.pos+1] == '=' {
			l.pos += 2
			return GE
		}
		l.pos++
		return GT
	case '!':
		if l.pos+1 < len(l.input) && l.input[l.pos+1] == '=' {
			l.pos += 2
			return NE
		}
		l.pos++
		return NOT
	case '=':
		if l.pos+1 < len(l.input) && l.input[l.pos+1] == '=' {
			l.pos += 2
			return EQ
		}
		l.Error("unexpected '=' (use '==' for equality)")
		return 0
	case '&':
		if l.pos+1 < len(l.input) && l.input[l.pos+1] == '&' {
			l.pos += 2
			return AND
		}
		l.Error("unexpected '&' (use '&&' for logical AND)")
		return 0
	case '|':
		if l.pos+1 < len(l.input) && l.input[l.pos+1] == '|' {
			l.pos += 2
			return OR
		}
		l.Error("unexpected '|' (use '||' for logical OR)")
		return 0
	}

	// String literals
	if ch == '"' || ch == '\'' {
		return l.lexString(lval, ch)
	}

	// Numbers
	if unicode.IsDigit(rune(ch)) {
		return l.lexNumber(lval)
	}

	// Identifiers and keywords
	if unicode.IsLetter(rune(ch)) || ch == '_' {
		return l.lexIdentifier(lval)
	}

	l.Error(fmt.Sprintf("unexpected character: %c", ch))
	return 0
}

func (l *Lexer) lexString(lval *yySymType, quote byte) int {
	l.pos++ // skip opening quote
	start := l.pos
	escaped := false

	for l.pos < len(l.input) {
		if escaped {
			escaped = false
			l.pos++
			continue
		}
		if l.input[l.pos] == '\\' {
			escaped = true
			l.pos++
			continue
		}
		if l.input[l.pos] == quote {
			break
		}
		l.pos++
	}

	if l.pos >= len(l.input) {
		l.Error(fmt.Sprintf("unterminated string starting with %c", quote))
		return 0
	}

	// Store the string value with quotes (for compatibility with existing code)
	lval.str = string(quote) + l.input[start:l.pos] + string(quote)
	l.pos++ // skip closing quote
	return STRING
}

func (l *Lexer) lexNumber(lval *yySymType) int {
	start := l.pos
	hasDecimal := false

	// Handle negative sign
	if l.input[l.pos] == '-' {
		l.pos++
	}

	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if unicode.IsDigit(rune(ch)) {
			l.pos++
		} else if ch == '.' && !hasDecimal {
			hasDecimal = true
			l.pos++
		} else {
			break
		}
	}

	numStr := l.input[start:l.pos]
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		l.Error(fmt.Sprintf("invalid number: %s", numStr))
		return 0
	}

	lval.num = num
	return NUMBER
}

func (l *Lexer) lexIdentifier(lval *yySymType) int {
	start := l.pos

	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if unicode.IsLetter(rune(ch)) || unicode.IsDigit(rune(ch)) || ch == '_' || ch == '-' {
			l.pos++
		} else {
			break
		}
	}

	id := l.input[start:l.pos]

	// Check for keywords
	switch id {
	case "true":
		return TRUE
	case "false":
		return FALSE
	case "and":
		return AND
	case "or":
		return OR
	case "not":
		return NOT
	default:
		lval.str = id
		return IDENTIFIER
	}
}

// Error is called by the parser when an error occurs
func (l *Lexer) Error(s string) {
	l.err = fmt.Errorf("parse error at position %d: %s", l.pos, s)
}
