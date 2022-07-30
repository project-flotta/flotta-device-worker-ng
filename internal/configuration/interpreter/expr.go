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
	Value float32
}

func (p *NumExpr) String() string {
	if p.Value == float32(int(p.Value)) {
		return strconv.Itoa(int(p.Value))
	} else {
		return fmt.Sprintf("%.6g", p.Value)
	}
}

func (l *NumExpr) Accept(a *AST) value {
	return a.visitNumExpr(l)
}

// UnaryExpr
type UnaryExpr struct {
	Op    Token
	Right Expr
}

func (u *UnaryExpr) String() string {
	return u.Op.String() + u.Right.String()
}

func (u *UnaryExpr) Accept(a *AST) value {
	return a.visitUnaryExpr(u)
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

// LogicExpr is an expression like x > 0 && y == 1
type LogicExpr struct {
	Left  Expr
	Op    Token
	Right Expr
}

func (l *LogicExpr) String() string {
	return fmt.Sprintf("( %s %s %s )", l.Left.String(), l.Op.String(), l.Right.String())
}

func (l *LogicExpr) Accept(a *AST) value {
	return a.visitLogicExpr(l)
}

// GroupExpr is an expression like ( x == 2 )
type GroupExpr struct {
	Expr Expr
}

func (g *GroupExpr) String() string {
	return fmt.Sprintf("%s", g.Expr.String())
}

func (l *GroupExpr) Accept(a *AST) value {
	return a.visitGroupExpr(l)
}
