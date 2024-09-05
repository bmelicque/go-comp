package checker

import (
	"github.com/bmelicque/test-parser/parser"
	"github.com/bmelicque/test-parser/tokenizer"
)

type CallExpression struct {
	Callee   Expression
	TypeArgs *TupleExpression
	Args     *TupleExpression
	Typing   ExpressionType
}

func (c CallExpression) Loc() tokenizer.Loc {
	loc := c.Args.loc
	if c.Callee != nil {
		loc.Start = c.Callee.Loc().Start
	}
	return loc
}

// FIXME:
func (c CallExpression) Type() ExpressionType {
	callee := c.Callee
	if callee == nil {
		return nil
	}

	if calleeType, ok := callee.Type().(Function); ok {
		return calleeType.Returned
	} else {
		return nil
	}
}

func (c *Checker) checkCallExpression(expr parser.CallExpression) Expression {
	callee := c.checkExpression(expr.Callee)
	if expr.Args == nil && expr.TypeArgs == nil {
		return callee
	}

	typeArgs := checkTypeArgs(c, expr.TypeArgs)

	var args *TupleExpression
	if expr.Args != nil {
		args = &TupleExpression{loc: expr.Args.Loc()}
	}
	if expr.Args != nil && expr.Args.Expr != nil {
		ex := c.checkExpression(expr.Args.Expr)
		if e, ok := ex.(TupleExpression); ok {
			args = &e
		} else {
			args = &TupleExpression{[]Expression{ex}, ex.Loc()}
		}
	}

	returned := c.checkFunctionCallee(callee, typeArgs, args)
	return CallExpression{callee, typeArgs, args, returned}
}

func (c *Checker) checkFunctionCallee(callee Expression, typeArgs *TupleExpression, args *TupleExpression) ExpressionType {
	function, ok := callee.Type().(Function)
	if !ok {
		c.report("Function type expected", callee.Loc())
		return Primitive{UNKNOWN}
	}

	c.pushScope(NewScope())
	defer c.dropScope()
	c.addTypeArgsToScope(typeArgs, function.TypeParams)

	params := function.Params.elements
	checkFunctionArgsNumber(c, args, params, callee.Loc())
	checkFunctionArgs(c, args, params)
	return function.Returned.build(c.scope, nil)
}

func checkFunctionArgsNumber(c *Checker, args *TupleExpression, params []ExpressionType, loc tokenizer.Loc) {
	if args == nil {
		c.report("Expected arguments", loc)
		return
	}

	if len(params) < len(args.Elements) {
		loc := args.Elements[len(params)].Loc()
		loc.End = args.Elements[len(args.Elements)-1].Loc().End
		c.report("Too many arguments", loc)
	}
	if len(params) > len(args.Elements) {
		c.report("Missing argument(s)", args.Loc())
	}
}

func checkFunctionArgs(c *Checker, args *TupleExpression, params []ExpressionType) {
	if args == nil {
		return
	}
	l := len(params)
	if len(args.Elements) < len(params) {
		l = len(args.Elements)
	}
	for i := 0; i < l; i++ {
		element := args.Elements[i]
		received := element.Type()
		built := params[i].build(c.scope, received)
		if !built.Extends(received) {
			c.report("Types don't match", element.Loc())
		}
	}
}
