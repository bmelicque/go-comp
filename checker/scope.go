package checker

import (
	"github.com/bmelicque/test-parser/tokenizer"
)

type Variable struct {
	declaredAt tokenizer.Loc
	typing     ExpressionType
	writes     []tokenizer.Loc
	reads      []tokenizer.Loc
}

type Method struct {
	self       ExpressionType
	signature  Function
	declaredAt tokenizer.Loc
	reads      []tokenizer.Loc
}

type Scope struct {
	variables  map[string]*Variable
	methods    map[string][]Method
	returnType ExpressionType // The expected type for a return statement (if any)
	outer      *Scope
	shadow     bool
}

func NewScope() *Scope {
	return &Scope{
		variables: map[string]*Variable{},
		methods:   map[string][]Method{},
	}
}

func NewShadowScope() *Scope {
	return &Scope{
		variables: map[string]*Variable{},
		methods:   map[string][]Method{},
		shadow:    true,
	}
}

func (s Scope) Find(name string) (*Variable, bool) {
	variable, ok := s.variables[name]
	if ok {
		return variable, true
	}
	if s.outer != nil {
		return s.outer.Find(name)
	}
	return nil, false
}

func (s Scope) FindMethod(name string, typing ExpressionType) (*Method, bool) {
	methods, ok := s.methods[name]
	if !ok {
		return nil, false
	}
	for _, method := range methods {
		found := method.self
		if f, ok := found.(Type); ok {
			found = f.Value
		}
		if found.Match(typing) {
			return &method, true
		}
	}
	if s.outer != nil {
		return s.outer.FindMethod(name, typing)
	}
	return nil, false
}

func (s Scope) Has(name string) bool {
	_, ok := s.variables[name]
	if ok {
		return true
	}
	if s.outer != nil && s.outer.shadow {
		return s.outer.Has(name)
	}
	return false
}

func (s *Scope) Add(name string, declaredAt tokenizer.Loc, typing ExpressionType) {
	s.variables[name] = &Variable{
		declaredAt: declaredAt,
		typing:     typing,
		writes:     []tokenizer.Loc{},
		reads:      []tokenizer.Loc{},
	}
}

func (s *Scope) AddMethod(name string, declaredAt tokenizer.Loc, self ExpressionType, signature Function) {
	s.methods[name] = append(s.methods[name], Method{self, signature, declaredAt, []tokenizer.Loc{}})
}

func (s *Scope) WriteAt(name string, loc tokenizer.Loc) {
	variable, ok := s.Find(name)
	// TODO: panic on error
	if ok {
		variable.writes = append(variable.writes, loc)
	}
}

func (s *Scope) ReadAt(name string, loc tokenizer.Loc) {
	variable, ok := s.Find(name)
	// TODO: panic on error
	if ok {
		variable.reads = append(variable.reads, loc)
	}
}

func (s Scope) GetReturnType() ExpressionType {
	return s.returnType
}
