// Grammar
//
// expression:		or
// or:				and | and ( "||" and)*										;
// and:				comparison | comparison( "&&" comparison)*					; x < 2 == y > 3 && w == 2 && z == 2
// comparison:		primary ("==" | "!=" | "<" | "<=" | ">" | ">=" ) value		; remark: normally equality has lower precedence than comparison but in our context we don't care about that
//																				; we don't accept expression like x > 2 == y > 2 == z > 3. There are not useful for our usecase.
// value:			unary | unary primary										; used as value + unit of measure like: 2.2Gib
// unary:			( "-" ) primary	| primary									; could be 2, 2.2, -2.2
// primary:			STRING | NUMBER | "( expression )"							; cpu123, cpu_123

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

func parse(src []byte) (expr Expr, err error) {
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
	expr = p.expression()

	return
}

func (p *parser) expression() Expr {
	return p.or()
}

func (p *parser) or() Expr {
	var expr Expr
	expr = p.and()

	for p.matches(OR) {
		p.next()
		right := p.and()
		expr = &LogicExpr{expr, OR, right}
	}

	return expr
}

// Parse and expression
//
func (p *parser) and() Expr {
	var expr Expr

	expr = p.compare()

	for p.matches(AND) {
		p.next()
		right := p.compare()
		expr = &LogicExpr{expr, AND, right}
		return expr
	}

	if !p.matches(OR, RPAREN, EOL) {
		panic(p.errorf("expecting OR RPAREN or EOL instead of '%s'", p.tok))
	}

	return expr
}

// Parse compare expression
//
// compare: primary ("==" | "<" | "<=" | ">" | ">=" ) value
//
func (p *parser) compare() Expr {
	expr := p.primary()

	if p.matches(GREATER, GTE, LESS, LTE, EQUALS, NOT_EQUALS) {
		op := p.tok
		p.next()
		right := p.value()
		expr = &CompExpr{expr, op, right}
	}

	return expr
}

// Parse a value
func (p *parser) value() Expr {
	expr := p.unary()

	if p.matches(STRING) {
		e := &ValueExpr{Left: expr}
		e.Right = p.primary()

		return e
	}

	return expr
}

func (p *parser) unary() Expr {
	if p.matches(DEC) {
		expr := &UnaryExpr{Op: p.tok}
		p.next()

		p.expect(NUMBER)
		expr.Right = p.primary()

		return expr
	}

	p.expect(NUMBER)

	return p.primary()
}

func (p *parser) primary() Expr {
	switch p.tok {
	case STRING:
		val := p.val
		p.next()
		return &LiteralExpr{val}
	case NUMBER:
		value, err := strconv.ParseFloat(p.val, 32)
		if err != nil {
			panic(p.errorf("expected number instead of '%s'", p.val))
		}
		p.next()
		return &NumExpr{float32(value)}
	case LPAREN:
		p.next()
		expr := p.expression()
		p.consume(RPAREN, "expect ')' after expression")
		return &GroupExpr{expr}
	default:
		panic(p.errorf("unknown token '%s'", p.tok))
	}
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

func (p *parser) check(tok Token) bool {
	return p.tok == tok
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
