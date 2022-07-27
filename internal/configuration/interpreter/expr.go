package interpreter

import (
	"fmt"
	"strconv"
)

type Expr interface {
	String() string
	Accept(a *AST) value // visitor pattern
}

// LiteralExpr is an expression like 'cpu123'
type LiteralExpr struct {
	Name string
}

func (l *LiteralExpr) String() string {
	return l.Name
}

// Accept looks into AST variables map and return the numberic value of the variable or it panics.
func (l *LiteralExpr) Accept(a *AST) value {
	return a.visitLiteralExpr(l)
}

// NumExpr is an expression like 1234.
type NumExpr struct {
	Value float64
}

func (p *NumExpr) String() string {
	if p.Value == float64(int(p.Value)) {
		return strconv.Itoa(int(p.Value))
	} else {
		return fmt.Sprintf("%.6g", p.Value)
	}
}

func (l *NumExpr) Accept(a *AST) value {
	return a.visitNumExpr(l)
}

// ValueExpr is an expression like 100Gib
type ValueExpr struct {
	Left  Expr
	Right Expr
}

func (v *ValueExpr) String() string {
	if v.Right != nil {
		return fmt.Sprintf("%s %s", v.Left.String(), v.Right.String())
	}
	return fmt.Sprintf("%s", v.Left.String())
}

func (l *ValueExpr) Accept(a *AST) value {
	return a.visitValueExpr(l)
}

// CompExpr is an expression like cpu < 23%
type CompExpr struct {
	Left  Expr
	Op    Token
	Right Expr
}

func (c *CompExpr) String() string {
	return fmt.Sprintf("( %s %s %s )", c.Left.String(), c.Op.String(), c.Right.String())
}

func (c *CompExpr) Accept(a *AST) value {
	return a.visitComprExpr(c)
}
