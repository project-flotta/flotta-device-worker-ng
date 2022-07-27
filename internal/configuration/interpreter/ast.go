package interpreter

type AST struct {
	// Variables holds the numeric variables used to evaluate the expressions.
	variables map[string]interface{}
}

func newAst(v map[string]interface{}) *AST {
	return &AST{variables: v}
}

func (a *AST) visitComprExpr(e *CompExpr) value {
	valueLeft := e.Left.Accept(a)
	valueRight := e.Right.Accept(a)

	if valueLeft.typ != valueRight.typ {
		panic(newEvaluationError(e, "type mismatch between left and right expression"))
	}

	// bool type does not support operators: '>', '>=', '>' and '>='
	if valueLeft.typ == typeBool {
		switch e.Op {
		case EQUALS:
			return boolean(valueLeft.b == valueRight.b)
		case NOT_EQUALS:
			return boolean(valueLeft.b != valueRight.b)
		case AND:
			return boolean(valueLeft.b && valueRight.b)
		case OR:
			return boolean(valueLeft.b || valueRight.b)
		default:
			panic(newEvaluationError(e, "bool type does not support '%s' operator", e.Op))
		}
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

func (a *AST) visitValueExpr(e *ValueExpr) value {
	numExpr := e.Left.(*NumExpr)
	return num(numExpr.Value)
}

func (a *AST) visitLiteralExpr(e *LiteralExpr) value {
	v, ok := a.variables[e.Name]
	if !ok {
		panic(newEvaluationError(e, "cannot find variable %s", e.Name))
	}

	// check the type of the variable
	switch vv := v.(type) {
	case bool:
		return boolean(vv)
	case float64:
		return num(vv)
	case int:
		return num(float64(vv))
	default:
		panic(newEvaluationError(e, "variable '%s' has the wrong type", e.Name))
	}
}
