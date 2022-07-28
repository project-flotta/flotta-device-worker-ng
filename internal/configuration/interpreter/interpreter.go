package interpreter

import (
	"bytes"
	"fmt"
)

// EvaluationError is the type of error returned by interpreter when evaluating errors.
type EvaluationError struct {
	// Source line/column position where the error occurred.
	Expr Expr
	// Error message.
	Message string
}

// Error returns a formatted version of the error, including the line number.
func (e *EvaluationError) Error() string {
	return fmt.Sprintf("expr '%s': %s", e.Expr.String(), e.Message)
}

func newEvaluationError(e Expr, format string, args ...interface{}) error {
	message := fmt.Sprintf(format, args...)
	return &EvaluationError{Expr: e, Message: message}
}

type Interpreter struct {
	expr Expr
}

func NewInterpreter(expression string) (*Interpreter, error) {
	expr, err := parse(bytes.NewBufferString(expression).Bytes())
	if err != nil {
		return nil, err
	}

	return &Interpreter{expr}, nil
}

// evaluate evaluates the expression to bool.
func (i *Interpreter) evaluate(variables map[string]interface{}) (result bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(*EvaluationError)
		}
	}()

	a := newAst(variables)
	v := i.expr.Accept(a)

	fmt.Printf("****** %s\n", i.expr.String())

	if v.typ != typeBool {
		return false, newEvaluationError(i.expr, "expected bool value. actual '%v'", v.typ)
	}

	result = v.b

	return
}
