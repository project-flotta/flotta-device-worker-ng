// Grammar
//
// expression: comparison | comparison ( ("&&" | "||") comparison)*					;
// comparison: literal ( ("==" | "<" | "<=" | ">" | ">=" ) value )*					;
// value: numerical literal																;
// numerical: "-"? NUMBER																;
// literal: STRING																	;

package interpreter

import (
	"fmt"
	"strconv"
)

// ParseError (actually *ParseError) is the type of error returned by parse.
type ParseError struct {
	// Source line/column position where the error occurred.
	Position int
	// Error message.
	Message string
}

// Error returns a formatted version of the error, including the line number.
func (e *ParseError) Error() string {
	return fmt.Sprintf("parse error at column %d: %s", e.Position, e.Message)
}

type parser struct {
	// Lexer instance and current token values
	lexer *lexer
	pos   int    // position of last token (tok)
	tok   Token  // last lexed token
	val   string // string value of last token (or "")
}

func parse(src []byte) (expr *CompExpr, err error) {
	defer func() {
		if r := recover(); r != nil {
			// Convert to ParseError or re-panic
			err = r.(*ParseError)
		}
	}()

	lexer := newLexer(src)
	p := parser{lexer: lexer}
	p.next() // initialize p.tok

	// parse the expression
	expr = p.expression().(*CompExpr)

	return
}

// Parse an expression
//
// expression: comparison | comparison ( ("&&" | "||") comparison)*
//
func (p *parser) expression() Expr {
	var expr Expr

	if p.matches(LPAREN) {
		p.next()
		expr = p.expression()
		p.consume(RPAREN, "expected ')' after expression")
	} else {
		expr = p.comparison()
	}

	if !p.matches(AND, OR) && !p.matches(EOL, RPAREN) {
		panic(p.errorf("unexpected expression after '%s'", p.tok))
	}

	for p.matches(AND, OR) {
		op := p.tok
		p.next()

		var right Expr
		if p.matches(LPAREN) {
			p.next()
			right = p.expression()
			p.consume(RPAREN, "expected ')' after expression")
		} else {
			right = p.comparison()
		}

		expr = &CompExpr{Left: expr, Op: op, Right: right}
	}

	return expr
}

// Parse equality expression
//
// comparison: literal ( ("==" | "<" | "<=" | ">" | ">=" ) value )*
//
func (p *parser) comparison() Expr {
	expr := &CompExpr{Left: p.literal()}

	switch p.tok {
	case GREATER, GTE, LESS, LTE, EQUALS, NOT_EQUALS:
		expr.Op = p.tok
		p.next()
	default:
		panic(p.errorf("expected comparison operator instead of %s", p.tok))

	}

	expr.Right = p.value()

	return expr
}

// Parse a value
func (p *parser) value() Expr {
	left := p.numerical()

	var right Expr
	if p.tok == STRING {
		right = p.literal()
	}

	return &ValueExpr{left, right}
}

// Parse literal
//
// STRING
//
func (p *parser) literal() Expr {
	var expr Expr

	p.expect(STRING)

	expr = &LiteralExpr{p.val}

	p.next()

	return expr
}

func (p *parser) numerical() Expr {
	if p.tok != DEC && p.tok != NUMBER {
		panic(p.errorf("expected '-' or number instead of %s", p.tok))
	}

	isNegative := false
	if p.tok == DEC {
		isNegative = true
		p.next()
	}

	p.expect(NUMBER)

	value, err := strconv.ParseFloat(p.val, 64)
	if err != nil {
		panic(p.errorf("expected number instead of '%s'", p.val))
	}

	if isNegative {
		value = -1 * value
	}

	p.next()

	return &NumExpr{value}
}

// Parse next token into p.tok (and set p.pos and p.val).
func (p *parser) next() {
	p.pos, p.tok, p.val = p.lexer.Scan()
	if p.tok == ILLEGAL {
		panic(p.errorf("%s", p.val))
	}
}

// Return true iff current token matches one of the given operators,
// but don't parse next token.
func (p *parser) matches(operators ...Token) bool {
	for _, operator := range operators {
		if p.tok == operator {
			return true
		}
	}
	return false
}

// Ensure current token is tok, and parse next token into p.tok.
func (p *parser) expect(tok Token) {
	if p.tok != tok {
		panic(p.errorf("expected %s instead of %s", tok, p.tok))
	}
}

func (p *parser) consume(tok Token, msg string) {
	if !p.matches(tok) {
		panic(p.errorf(msg))
	}
	p.next()
}

// Format given string and args with Sprintf and return an error
// with that message and the current position.
func (p *parser) errorf(format string, args ...interface{}) error {
	message := fmt.Sprintf(format, args...)
	return &ParseError{p.pos, message}
}
