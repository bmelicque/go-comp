package parser

type ForExpression struct {
	Keyword   Token
	Statement Node // Expression (boolean) | Assignment from range
	Body      *Block
	typing    ExpressionType
}

func (f *ForExpression) getChildren() []Node {
	children := []Node{}
	if f.Statement != nil {
		children = append(children, f.Statement)
	}
	if f.Body != nil {
		children = append(children, f.Body)
	}
	return children
}

func (f *ForExpression) typeCheck(p *Parser) {
	switch s := f.Statement.(type) {
	case *Assignment:
		r := getLoopRangeType(s.Value)
		if r == nil {
			p.report("Range expected", s.Value.Loc())
		}
		switch pattern := s.Pattern.(type) {
		case *Identifier:
			p.scope.Add(pattern.Text(), pattern.Loc(), r)
		case *TupleExpression:
			// TODO: FIXME:
			// index, value := getValidatedRangeTuplePattern(p, pattern)
		}
	case Expression:
		// s.Expr == nil already reported when parsing expression
		if s == nil {
			break
		}
		if _, ok := s.Type().(Boolean); !ok {
			p.report("Boolean expected in loop condition", s.Loc())
		}
	}
	f.Body.typeCheck(p)
	f.typing = getLoopType(p, f.Body)
}

func (f *ForExpression) Loc() Loc {
	loc := f.Keyword.Loc()
	if f.Body != nil {
		loc.End = f.Body.Loc().End
	} else if f.Statement != nil {
		loc.End = f.Statement.Loc().End
	}
	return loc
}
func (f *ForExpression) Type() ExpressionType { return f.typing }

func (p *Parser) parseForExpression() *ForExpression {
	p.pushScope(NewScope(LoopScope))
	defer p.dropScope()

	keyword := p.Consume()

	outer := p.allowBraceParsing
	p.allowBraceParsing = false
	statement := p.parseAssignment()
	p.allowBraceParsing = outer
	validateForCondition(p, statement)

	body := p.parseBlock()

	return &ForExpression{keyword, statement, body, nil}
}

func validateForCondition(p *Parser, s Node) {
	switch s := s.(type) {
	case *Assignment:
		switch pattern := s.Pattern.(type) {
		case *Identifier, *TupleExpression:
		default:
			p.report("Invalid pattern", pattern.Loc())
		}
	case Expression:
	default:
		p.report("Assignment from range or condition expected", s.Loc())
	}
}

func getLoopRangeType(expr Expression) ExpressionType {
	if expr == nil {
		return nil
	}
	r, ok := expr.Type().(Range)
	if !ok {
		return nil
	}
	return r.operands
}

func getValidatedRangeTuplePattern(p *Parser, tuple *TupleExpression) (*Identifier, *Identifier) {
	// Parsing ensures that length is >= 2
	if len(tuple.Elements) != 2 {
		p.report("Only 2 elements expected", tuple.Loc())
	}

	index, ok := tuple.Elements[0].(*Identifier)
	if !ok {
		p.report("Identifier expected", tuple.Elements[0].Loc())
	}
	value, ok := tuple.Elements[0].(*Identifier)
	if !ok {
		p.report("Identifier expected", tuple.Elements[1].Loc())
	}
	return index, value
}

func getLoopType(p *Parser, body *Block) ExpressionType {
	breaks := []*Exit{}
	findBreakStatements(body, &breaks)
	if len(breaks) == 0 {
		return Nil{}
	}
	var t ExpressionType
	if breaks[0].Value != nil {
		t = breaks[0].Value.Type()
	} else {
		t = Nil{}
	}
	for _, b := range breaks[1:] {
		if t == (Nil{}) && b.Value != nil {
			p.report("No value expected", b.Value.Loc())
		}
		if t != (Nil{}) && !t.Extends(b.Value.Type()) {
			p.report("Type doesn't match the type inferred from first break", b.Value.Loc())
		}
	}
	return t
}

func findBreakStatements(node Node, results *[]*Exit) {
	if node == nil {
		return
	}
	if n, ok := node.(*Exit); ok {
		if n.Operator.Kind() == BreakKeyword {
			*results = append(*results, n)
		}
		return
	}
	switch node := node.(type) {
	case *Block:
		for _, statement := range node.Statements {
			findBreakStatements(statement, results)
		}
	case *IfExpression:
		findBreakStatements(node.Body, results)
		findBreakStatements(node.Alternate, results)
	}
}
