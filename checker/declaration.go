package checker

import (
	"errors"
	"unicode"

	"github.com/bmelicque/test-parser/parser"
	"github.com/bmelicque/test-parser/tokenizer"
)

type VariableDeclaration struct {
	Pattern     Expression
	Initializer Expression
	loc         tokenizer.Loc
	Constant    bool
}

func (vd VariableDeclaration) Loc() tokenizer.Loc { return vd.loc }

func (c *Checker) declareIdentifier(declared parser.Node, typing ExpressionType) (Identifier, error) {
	token, ok := declared.(parser.TokenExpression)
	if !ok {
		return Identifier{}, errors.New("identifier expected")
	}
	identifier, ok := c.checkToken(token, false).(Identifier)
	if !ok {
		return Identifier{}, errors.New("identifier expected")
	}
	name := identifier.Token.Text()

	isTypeIdentifier := unicode.IsUpper(rune(name[0]))
	_, isTypeTyping := typing.(Type)

	if isTypeIdentifier != isTypeTyping {
		return Identifier{}, errors.New("types don't match")
	}

	c.scope.Add(name, declared.Loc(), typing)
	return identifier, nil
}

func (c *Checker) checkVariableDeclaration(a parser.Assignment) VariableDeclaration {
	var pattern Expression
	var err error
	init := c.checkExpression(a.Initializer)
	constant := a.Operator.Kind() == tokenizer.DEFINE

	declared := a.Declared
	if d, ok := declared.(parser.ParenthesizedExpression); ok {
		declared = d.Expr
	}
	switch declared := declared.(type) {
	case parser.TokenExpression:
		pattern, err = c.declareIdentifier(declared, init.Type())
		if err != nil {
			c.report(err.Error(), declared.Loc())
		}
	case parser.TupleExpression:
		tuple, ok := init.Type().(Tuple)
		if !ok {
			c.report("Pattern doesn't match initializer type", declared.Loc())
		}
		elements := make([]Expression, len(declared.Elements))
		for i, element := range declared.Elements {
			identifier, err := c.declareIdentifier(element, tuple.elements[i])
			if err != nil {
				c.report(err.Error(), declared.Loc())
			}
			elements[i] = identifier
		}
		pattern = TupleExpression{elements, declared.Loc()}
	default:
		c.report("Invalid pattern", declared.Loc())
	}
	return VariableDeclaration{pattern, init, a.Loc(), constant}
}