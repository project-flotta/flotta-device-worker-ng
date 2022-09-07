package interpreter

type AST struct {
	// Variables holds the numeric variables used to evaluate the expressions.
	variables map[string]interface{}
}

func newAst(v map[string]interface{}) *AST {
	return &AST{variables: v}
}

func (a *AST) evaluate(e Expr) value {
	return e.Accept(a)
}

func (a *AST) visitLogicExpr(e *LogicExpr) value {
	valueLeft := e.Left.Accept(a)
	valueRight := e.Right.Accept(a)

	switch e.Op {
	case OR:
		return boolean(valueLeft.b || valueRight.b)
	case AND:
		return boolean(valueLeft.b && valueRight.b)
	default:
		panic(newEvaluationError(e, "operator '%s' not supported", e.Op))
	}
}

func (a *AST) visitComprExpr(e *CompExpr) value {
	valueLeft := e.Left.Accept(a)
	valueRight := e.Right.Accept(a)

	if valueRight.typ == typeNull {
		switch e.Op {
		case EQUALS:
			return boolean(valueLeft.typ == typeNull)
		case NOT_EQUALS:
			return boolean(valueLeft.typ != typeNull)
		default:
			panic(newEvaluationError(e, "operator '%s' not supported on nil", e.Op))
		}

	}

	// at this point, we expect to find each expression with values.
	// So, valueLeft need to be different than nil
	if valueLeft.typ == typeNull {
		panic(newEvaluationError(e, "cannot evaluate expression. missing left value"))
	}

	switch e.Op {
	case LESS:
		return boolean(valueLeft.n < valueRight.n)
	case LTE:
		return boolean(valueLeft.n <= valueRight.n)
	case GREATER:
		return boolean(valueLeft.n > valueRight.n)
	case GTE:
		return boolean(valueLeft.n >= valueRight.n)
	case EQUALS:
		return boolean(valueLeft.n == valueRight.n)
	case NOT_EQUALS:
		return boolean(valueLeft.n != valueRight.n)
	default:
		panic(newEvaluationError(e, "operator '%s' not supported", e.Op))
	}
}

func (a *AST) visitNumExpr(e *NumExpr) value {
	return num(e.Value)
}

func (a *AST) visitNilExpr(e *NilExpr) value {
	return null()
}

func (a *AST) visitValueExpr(e *ValueExpr) value {
	numExpr := e.Left.(*NumExpr)
	return num(numExpr.Value)
}

func (a *AST) visitUnaryExpr(e *UnaryExpr) value {
	val := e.Right.(*NumExpr)
	return num(-1 * val.Value)
}

func (a *AST) visitLiteralExpr(e *LiteralExpr) value {
	v, ok := a.variables[e.Name]
	if !ok {
		return null()
	}

	// check the type of the variable
	switch vv := v.(type) {
	case bool:
		return boolean(vv)
	case float32:
		return num(vv)
	case int:
		return num(float32(vv))
	default:
		panic(newEvaluationError(e, "variable '%s' has the wrong type", e.Name))
	}
}

func (a *AST) visitGroupExpr(e *GroupExpr) value {
	return e.Expr.Accept(a)
}
