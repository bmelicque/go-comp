package parser

import (
	"fmt"
	"slices"
)

type BinaryExpression struct {
	Left     Expression
	Right    Expression
	Operator Token
}

func (b *BinaryExpression) getChildren() []Node {
	children := []Node{}
	if b.Left != nil {
		children = append(children, b.Left)
	}
	if b.Right != nil {
		children = append(children, b.Right)
	}
	return children
}

func (expr *BinaryExpression) Loc() Loc {
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

func (expr *BinaryExpression) Type() ExpressionType {
	switch expr.Operator.Kind() {
	case
		Add,
		Sub,
		Mul,
		Pow,
		Div,
		Mod:
		return Number{}
	case Concat:
		return expr.Left.Type()
	case
		LogicalAnd,
		LogicalOr,
		Less,
		Greater,
		LessEqual,
		GreaterEqual,
		Equal,
		NotEqual:
		return Boolean{}
	case Bang:
		left := expr.Left.Type()
		if t, ok := left.(Type); ok {
			left = t.Value
		} else {
			left = Unknown{}
		}

		right := expr.Right.Type()
		if t, ok := right.(Type); ok {
			right = t.Value
		} else {
			right = Unknown{}
		}
		return Type{makeResultType(right, left)}
	default:
		panic(fmt.Sprintf("operator '%v' not implemented", expr.Operator.Kind()))
	}
}

/******************************
 *  PARSING HELPER FUNCTIONS  *
 ******************************/
func (p *Parser) parseBinaryExpression() Expression {
	return parseBinaryErrorType(p)
}
func parseBinary(p *Parser, operators []TokenKind, fallback func(p *Parser) Expression) Expression {
	expression := fallback(p)
	next := p.Peek()
	for slices.Contains(operators, next.Kind()) {
		operator := p.Consume()
		right := fallback(p)
		expression = &BinaryExpression{expression, right, operator}
		next = p.Peek()
	}
	return expression
}
func parseBinaryErrorType(p *Parser) Expression {
	return parseBinary(p, []TokenKind{Bang}, parseLogicalOr)
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
	expression := p.parseCatchExpression()
	next := p.Peek()
	for next.Kind() == Pow {
		operator := p.Consume()
		right := parseExponentiation(p)
		expression = &BinaryExpression{expression, right, operator}
		next = p.Peek()
	}
	return expression
}

func (b *BinaryExpression) typeCheck(p *Parser) {
	b.Left.typeCheck(p)
	b.Right.typeCheck(p)
	switch b.Operator.Kind() {
	case
		Add,
		Sub,
		Mul,
		Pow,
		Div,
		Mod,
		Less,
		Greater,
		LessEqual,
		GreaterEqual:
		p.typeCheckArithmeticExpression(b.Left, b.Right)
	case Concat:
		p.typeCheckConcatExpression(b.Left, b.Right)
	case
		LogicalAnd,
		LogicalOr:
		p.typeCheckLogicalExpression(b.Left, b.Right)
	case
		Equal,
		NotEqual:
		p.typeCheckComparisonExpression(b.Left, b.Right)
	case Bang:
		typeCheckBinaryErrorType(p, b.Left, b.Right)
	default:
		panic(fmt.Sprintf("operator '%v' not implemented", b.Operator.Kind()))
	}
}

func (p *Parser) typeCheckLogicalExpression(left Expression, right Expression) {
	if left != nil && !(Boolean{}).Extends(left.Type()) {
		p.report("The left-hand side of a logical operation must be a boolean", left.Loc())
	}
	if right != nil && !(Boolean{}).Extends(right.Type()) {
		p.report("The right-hand side of a logical operation must be a boolean", right.Loc())
	}
}

func (p *Parser) typeCheckComparisonExpression(left Expression, right Expression) {
	if left == nil || right == nil {
		return
	}
	leftType := left.Type()
	rightType := right.Type()
	if !Match(leftType, rightType) {
		p.report("Types don't match", Loc{Start: left.Loc().Start, End: right.Loc().End})
	}
}
func (p *Parser) typeCheckConcatExpression(left Expression, right Expression) {
	var leftType ExpressionType
	if left != nil {
		leftType = left.Type()
	}
	var rightType ExpressionType
	if right != nil {
		rightType = right.Type()
	}
	if leftType != nil && !(String{}).Extends(leftType) && !(List{Unknown{}}).Extends(leftType) {
		p.report("The left-hand side of concatenation must be a string or a list", left.Loc())
	}
	if rightType != nil && !(String{}).Extends(rightType) && !(List{Unknown{}}).Extends(rightType) {
		p.report("The right-hand side of concatenation must be a string or a list", right.Loc())
	}

	rightList, ok := rightType.(List)
	if !ok {
		return
	}
	leftList, ok := leftType.(List)
	if !ok {
		return
	}
	if !leftList.Element.Extends(rightList.Element) {
		p.report("Element type doesn't match lhs", right.Loc())
	}
}
func (p *Parser) typeCheckArithmeticExpression(left Expression, right Expression) {
	if left != nil && !(Number{}).Extends(left.Type()) {
		p.report("The left-hand side of an arithmetic operation must be a number", left.Loc())
	}
	if right != nil && !(Number{}).Extends(right.Type()) {
		p.report("The right-hand side of an arithmetic operation must be a number", right.Loc())
	}
}
func typeCheckBinaryErrorType(p *Parser, left Expression, right Expression) {
	if _, ok := left.Type().(Type); !ok {
		p.report("Type expected", left.Loc())
	}
	if _, ok := right.Type().(Type); !ok {
		p.report("Type expected", right.Loc())
	}
}
