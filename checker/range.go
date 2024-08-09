package checker

import (
	"github.com/bmelicque/test-parser/parser"
	"github.com/bmelicque/test-parser/tokenizer"
)

type RangeExpression struct {
	Left     Expression
	Right    Expression
	Operator tokenizer.Token
}

func (r RangeExpression) Loc() tokenizer.Loc {
	var loc tokenizer.Loc
	if r.Left != nil {
		loc.Start = r.Left.Loc().Start
	} else {
		loc.Start = r.Operator.Loc().Start
	}
	if r.Right != nil {
		loc.End = r.Right.Loc().End
	} else {
		loc.End = r.Operator.Loc().End
	}
	return loc
}

func (r RangeExpression) Type() ExpressionType {
	var typing ExpressionType
	if r.Left != nil {
		typing = r.Left.Type()
	} else if r.Right != nil {
		typing = r.Right.Type()
	}
	return Range{typing}
}

func (c *Checker) checkRangeExpression(expr parser.RangeExpression) RangeExpression {
	var left, right Expression
	if expr.Left != nil {
		left = c.CheckExpression(expr.Left)
	}
	if expr.Right != nil {
		right = c.CheckExpression(expr.Right)
	}

	if left != nil && right != nil && !left.Type().Match(right.Type()) {
		c.report("Types don't match", expr.Loc())
	}

	if expr.Operator.Kind() == tokenizer.RANGE_INCLUSIVE && expr.Right == nil {
		c.report("Expected right operand", expr.Operator.Loc())
	}

	return RangeExpression{left, right, expr.Operator}
}
