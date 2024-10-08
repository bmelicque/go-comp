package parser

import (
	"slices"
)

type BinaryExpression struct {
	Left     Expression
	Right    Expression
	Operator Token
}

func (expr BinaryExpression) Loc() Loc {
	loc := Loc{}
	if expr.Left != nil {
		loc.Start = expr.Left.Loc().Start
	} else {
		loc.Start = expr.Operator.Loc().Start
	}

	if expr.Right != nil {
		loc.End = expr.Right.Loc().End
	} else {
		loc.End = expr.Operator.Loc().End
	}
	return loc
}

/******************************
 *  PARSING HELPER FUNCTIONS  *
 ******************************/
func (BinaryExpression) Parse(p *Parser) Expression {
	return parseLogicalOr(p)
}
func parseBinary(p *Parser, operators []TokenKind, fallback func(p *Parser) Expression) Expression {
	expression := fallback(p)
	next := p.Peek()
	for slices.Contains(operators, next.Kind()) {
		operator := p.Consume()
		right := fallback(p)
		expression = BinaryExpression{expression, right, operator}
		next = p.Peek()
	}
	return expression
}
func parseLogicalOr(p *Parser) Expression {
	return parseBinary(p, []TokenKind{LogicalOr}, parseLogicalAnd)
}
func parseLogicalAnd(p *Parser) Expression {
	return parseBinary(p, []TokenKind{LogicalAnd}, parseEquality)
}
func parseEquality(p *Parser) Expression {
	return parseBinary(p, []TokenKind{Equal, NotEqual}, parseComparison)
}
func parseComparison(p *Parser) Expression {
	return parseBinary(p, []TokenKind{Less, LessEqual, GreaterEqual, Greater}, parseAddition)
}
func parseAddition(p *Parser) Expression {
	return parseBinary(p, []TokenKind{Add, Concat, Sub}, parseMultiplication)
}
func parseMultiplication(p *Parser) Expression {
	return parseBinary(p, []TokenKind{Mul, Div, Mod}, parseExponentiation)
}
func parseExponentiation(p *Parser) Expression {
	expression := p.parseAccessExpression()
	next := p.Peek()
	for next.Kind() == Pow {
		operator := p.Consume()
		right := parseExponentiation(p)
		expression = BinaryExpression{expression, right, operator}
		next = p.Peek()
	}
	return expression
}
