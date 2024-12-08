package parser

type Assignment struct {
	Pattern  Expression // "value", "Type", "(value: Type).method"
	Value    Expression
	Operator Token // '=', ':=', '::', '+='...
}

func (a *Assignment) typeCheck(p *Parser) {
	switch a.Operator.Kind() {
	case Assign:
		typeCheckAssignment(p, a)
	case AddAssign,
		SubAssign,
		MulAssign,
		DivAssign,
		ModAssign,
		ConcatAssign,
		LogicalAndAssign,
		LogicalOrAssign:
		typeCheckOtherAssignment(p, a)
	case Declare:
		typeCheckDeclaration(p, a)
	case Define:
		typeCheckDefinition(p, a)
	default:
		panic("Assignment type check should've been exhaustive!")
	}
}

func (a *Assignment) Loc() Loc {
	loc := a.Operator.Loc()
	if a.Pattern != nil {
		loc.Start = a.Pattern.Loc().Start
	}
	if a.Value != nil {
		loc.End = a.Value.Loc().End
	}
	return loc
}

func (a *Assignment) getChildren() []Node {
	children := []Node{}
	if a.Pattern != nil {
		children = append(children, a.Pattern)
	}
	if a.Value != nil {
		children = append(children, a.Value)
	}
	return children
}

func (p *Parser) parseAssignment() Node {
	expr := p.parseExpression()

	var operator Token
	next := p.Peek()
	switch next.Kind() {
	case Assign,
		AddAssign,
		ConcatAssign,
		SubAssign,
		MulAssign,
		DivAssign,
		ModAssign,
		LogicalAndAssign,
		LogicalOrAssign:
		if !isValidAssignee(expr) {
			p.error(expr, InvalidPattern)
			expr = nil
		}
		operator = p.Consume()
	case Declare, Define:
		operator = p.Consume()
	default:
		return expr
	}
	init := p.parseExpression()
	if c, ok := expr.(*ComputedAccessExpression); ok && operator.Kind() == Define {
		c.Property.Expr = makeTuple(c.Property.Expr)
		validateTypeParams(p, c.Property)
		expr = c
	}
	if b, ok := init.(*Block); ok && operator.Kind() == Define {
		validateObject(p, b)
		reportDuplicatedFields(p, b)
		init = b
	}
	return &Assignment{expr, init, operator}
}

func isValidAssignee(expr Expression) bool {
	valid := true
	Walk(expr, func(n Node, skip func()) {
		switch n := n.(type) {
		case *ComputedAccessExpression:
			// this is valid but we need to skip brackets,
			// which are otherwise invalid
			valid = isValidAssignee(n.Expr)
			skip()
		case *BinaryExpression,
			*Block,
			*BracketedExpression,
			*CallExpression,
			*CatchExpression,
			*Entry,
			*ForExpression,
			*FunctionExpression,
			*FunctionTypeExpression,
			*IfExpression,
			*InstanceExpression,
			*MatchExpression,
			*Param,
			*RangeExpression,
			*SumType,
			*UnaryExpression:
			valid = false
			skip()
		}
	})
	return valid
}

func validateObject(p *Parser, b *Block) {
	if len(b.Statements) == 1 {
		s := b.Statements[0]
		t, ok := s.(*TupleExpression)
		if !ok {
			b.Statements[0] = getValidatedObjectField(p, s)
			return
		}
		b.Statements = make([]Node, len(t.Elements))
		for i := range t.Elements {
			b.Statements[i] = getValidatedObjectField(p, t.Elements[i])
		}
		return
	}
	for i := range b.Statements {
		b.Statements[i] = getValidatedObjectField(p, b.Statements[i])
	}
}

func reportDuplicatedFields(p *Parser, b *Block) {
	declarations := map[string][]*Identifier{}
	for _, s := range b.Statements {
		var identifier *Identifier
		switch s := s.(type) {
		case *Param:
			identifier = s.Identifier
		case *Entry:
			identifier = s.Key.(*Identifier)
		}
		name := identifier.Text()
		if name != "" {
			declarations[name] = append(declarations[name], identifier)
		}
	}
	for _, identifiers := range declarations {
		if len(identifiers) == 1 {
			continue
		}
		for _, identifier := range identifiers {
			p.error(identifier, DuplicateIdentifier)
		}
	}
}

func getValidatedObjectField(p *Parser, node Node) Node {
	switch node := node.(type) {
	case *Param:
		return node
	case *Entry:
		if _, ok := node.Key.(*Identifier); !ok {
			p.error(node.Key, IdentifierExpected)
			return nil
		}
		return node
	default:
		p.error(node, FieldExpected)
		return nil
	}
}

// type check assignment where operator is '='
func typeCheckAssignment(p *Parser, a *Assignment) {
	outer := p.writing
	p.writing = a
	a.Pattern.typeCheck(p)
	p.writing = outer
	a.Value.typeCheck(p)
	reportInvalidVariableType(p, a.Value)

	switch pattern := a.Pattern.(type) {
	case *ComputedAccessExpression:
		if isMap(pattern.Expr.Type()) {
			t := pattern.typing.(TypeAlias).Ref.(Sum).getMember("Some")
			if !t.Extends(a.Value.Type()) {
				p.error(a, CannotAssignType, t, a.Value.Type())
			}
			return
		}
		if isSlice(pattern.Expr.Type()) {
			el := pattern.Expr.Type().(Ref).To.(List).Element
			if !el.Extends(a.Value.Type()) {
				p.error(a, CannotAssignType, el, a.Value.Type())
			}
			return
		}
		p.error(pattern, InvalidAssignmentToEntry)
	case *Identifier:
		if pattern.typing.Extends(a.Value.Type()) {
			return
		}
		p.error(pattern, CannotAssignType, pattern.typing, a.Value)
	case *TupleExpression:
		for _, element := range pattern.Elements {
			if _, ok := element.(*Identifier); !ok {
				p.error(element, IdentifierExpected)
			}
		}
		if !pattern.typing.Extends(a.Value.Type()) {
			p.error(a.Value, CannotAssignType, pattern.typing, a.Value)
		}
	default:
		p.error(a.Pattern, InvalidPattern)
	}
}

func typeCheckOtherAssignment(p *Parser, a *Assignment) {
	outer := p.writing
	p.writing = a
	a.Pattern.typeCheck(p)
	p.writing = outer
	a.Value.typeCheck(p)

	left := a.Pattern.Type()

	switch a.Operator.Kind() {
	case AddAssign, SubAssign, MulAssign, DivAssign, ModAssign:
		if !(Number{}).Extends(left) {
			p.error(a.Pattern, NumberExpected, left)
		}
		if !(Number{}).Extends(a.Value.Type()) {
			p.error(a.Pattern, NumberExpected, left)
		}
	case ConcatAssign:
		init := a.Value.Type()
		var err bool
		if !(String{}).Extends(left) && !(List{Unknown{}}).Extends(left) {
			p.error(a.Pattern, ConcatenableExpected, left)
			err = true
		}
		if !(String{}).Extends(init) && !(List{Unknown{}}).Extends(init) {
			p.error(a.Pattern, ConcatenableExpected, init)
			err = true
		}
		if !err && !left.Extends(init) {
			p.error(a, CannotAssignType, left, init)
		}
	case LogicalAndAssign, LogicalOrAssign:
		if !(Boolean{}).Extends(left) {
			p.error(a.Pattern, BooleanExpected, left)
		}
		if !(Boolean{}).Extends(a.Value.Type()) {
			p.error(a.Pattern, BooleanExpected, left)
		}
	}
}

// type check assignment where operator is ':='
func typeCheckDeclaration(p *Parser, a *Assignment) {
	a.Value.typeCheck(p)
	reportInvalidVariableType(p, a.Value)
	switch pattern := a.Pattern.(type) {
	case *Identifier:
		declareIdentifier(p, pattern, a.Value.Type())
	case *TupleExpression:
		declareTuple(p, pattern, a.Value.Type())
	case *CallExpression:
		if !p.conditionalDeclaration {
			p.error(a.Pattern, InvalidPattern)
			return
		}
		p.typeCheckPattern(a.Pattern, a.Value.Type())
	default:
		p.error(a.Pattern, InvalidPattern)
	}
}

func declareIdentifier(p *Parser, identifier *Identifier, typing ExpressionType) {
	name := identifier.Text()
	if name == "" || name == "_" {
		return
	}
	if name == "Map" || name == "panic" {
		p.error(identifier, ReservedName, name)
		return
	}
	p.scope.Add(name, identifier.Loc(), typing)
}

func declareTuple(p *Parser, pattern *TupleExpression, typing ExpressionType) {
	tuple, ok := typing.(Tuple)
	if !ok {
		p.error(pattern, InvalidTypeForPattern, pattern, typing)
		return
	}
	l := len(pattern.Elements)
	if l > len(tuple.Elements) {
		p.error(pattern, TooManyElements, len(tuple.Elements), len(pattern.Elements))
		l = len(tuple.Elements)
	}
	for i := 0; i < l; i++ {
		identifier, ok := pattern.Elements[i].(*Identifier)
		if !ok {
			p.error(pattern.Elements[i], IdentifierExpected)
			continue
		}
		declareIdentifier(p, identifier, tuple.Elements[i])
	}
}

func typeCheckDefinition(p *Parser, a *Assignment) {
	switch pattern := a.Pattern.(type) {
	case *ComputedAccessExpression:
		typeCheckGenericTypeDefinition(p, a)
	case *Identifier:
		if a.Value == nil {
			return
		}
		a.Value.typeCheck(p)
		reportInvalidVariableType(p, a.Value)
		if pattern.IsType() {
			typeCheckTypeDefinition(p, a)
		} else {
			typeCheckFunctionDefinition(p, a)
		}
	case *PropertyAccessExpression:
		typeCheckMethod(p, pattern, a.Value)
	default:
		a.Value.typeCheck(p)
		reportInvalidVariableType(p, a.Value)
		p.error(pattern, InvalidPattern)
	}
}

func typeCheckGenericTypeDefinition(p *Parser, a *Assignment) {
	pattern := a.Pattern.(*ComputedAccessExpression)

	p.pushScope(NewScope(ProgramScope))
	typeCheckTypeParams(p, pattern.Property)
	a.Value.typeCheck(p)
	p.dropScope()

	identifier, ok := pattern.Expr.(*Identifier)
	if !ok || !identifier.IsType() {
		p.error(pattern.Expr, TypeIdentifierExpected)
		return
	}
	t := Type{TypeAlias{
		Name:   identifier.Text(),
		Params: pattern.Property.getGenerics(),
		Ref:    getInitType(p, a.Value),
	}}
	p.scope.Add(identifier.Text(), pattern.Loc(), t)
}

func typeCheckTypeDefinition(p *Parser, a *Assignment) {
	identifier := a.Pattern.(*Identifier)
	t := Type{TypeAlias{
		Name: identifier.Text(),
		Ref:  getInitType(p, a.Value),
	}}
	declareIdentifier(p, identifier, t)
}

func getInitType(p *Parser, expr Expression) ExpressionType {
	if expr == nil {
		return Unknown{}
	}
	if b, ok := expr.(*Block); ok {
		return getObjectDefinedType(p, b)
	}
	t, ok := expr.Type().(Type)
	if !ok {
		p.error(expr, TypeExpected)
		return Unknown{}
	}
	return t.Value
}

func getObjectDefinedType(p *Parser, b *Block) ExpressionType {
	o := newObject()
	for _, s := range b.Statements {
		switch s := s.(type) {
		case *Param:
			key := s.Identifier.Text()
			t, ok := s.Complement.Type().(Type)
			if !ok {
				p.error(s.Complement, TypeExpected)
				o.addMember(key, Unknown{})
				continue
			}
			if len(o.Defaults) > 0 {
				p.error(s, MandatoryAfterOptional)
			}
			o.addMember(key, t.Value)
		case *Entry:
			key := s.Key.(*Identifier).Text()
			if _, ok := s.Value.Type().(Type); ok {
				p.error(s.Value, ValueExpected)
			}
			o.addDefault(key, s.Value.Type())
		}
	}
	return o
}

func typeCheckFunctionDefinition(p *Parser, a *Assignment) {
	identifier := a.Pattern.(*Identifier)
	t, ok := a.Value.Type().(Function)
	if !ok {
		p.error(a.Value, FunctionExpressionExpected)
		return
	}
	declareIdentifier(p, identifier, t)
}

func typeCheckMethod(p *Parser, expr *PropertyAccessExpression, init Expression) {
	p.pushScope(NewScope(ProgramScope))
	defer p.dropScope()

	typeIdentifier := declareMethodReceiver(p, expr.Expr)

	method, ok := expr.Property.(*Identifier)
	if !ok {
		p.error(expr.Property, IdentifierExpected)
	}

	init.typeCheck(p)

	if _, ok := init.Type().(Function); !ok {
		p.error(init, FunctionExpressionExpected)
		return
	}
	if !ok || typeIdentifier == nil {
		return
	}

	t, ok := typeIdentifier.Type().(Type)
	if !ok {
		return
	}
	alias, ok := t.Value.(TypeAlias)
	if !ok {
		return
	}

	p.scope.AddMethod(method.Text(), alias, init.Type().(Function))
}

func declareMethodReceiver(p *Parser, receiver Expression) *Identifier {
	paren, ok := receiver.(*ParenthesizedExpression)
	if !ok || paren.Expr == nil {
		p.error(receiver, ReceiverExpected)
		return nil
	}
	param, ok := paren.Expr.(*Param)
	if !ok {
		p.error(paren.Expr, ReceiverExpected)
		return nil
	}

	typeIdentifier, ok := param.Complement.(*Identifier)
	if !ok || !typeIdentifier.IsType() {
		p.error(param.Complement, TypeIdentifierExpected)
		return nil
	}
	typeIdentifier.typeCheck(p)

	if t, ok := typeIdentifier.Type().(Type); ok {
		p.scope.Add(
			param.Identifier.Text(),
			param.Identifier.Loc(),
			t.Value,
		)
	}
	return typeIdentifier
}

func reportInvalidVariableType(p *Parser, value Expression) {
	switch t := value.Type().(type) {
	case TypeAlias:
		if t.Name == "!" {
			p.error(value, ResultDeclaration)
		}
	case Nil:
		p.error(value, NilDeclaration)
	}
}
