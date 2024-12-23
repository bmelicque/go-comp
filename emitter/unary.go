package emitter

import "github.com/bmelicque/test-parser/parser"

func (e *Emitter) emitUnaryExpression(u *parser.UnaryExpression) {
	switch u.Operator.Kind() {
	case parser.AsyncKeyword:
		e.emitCallExpression(u.Operand.(*parser.CallExpression), false)
	case parser.AwaitKeyword:
		e.write("await ")
		e.emitExpression(u.Operand)
	case parser.Bang:
		e.write("!")
		e.emitExpression(u.Operand)
	case parser.TryKeyword:
		e.emitExpression(u.Operand)
	case parser.BinaryAnd:
		e.emitReference(u.Operand)
	case parser.Mul:
		e.emitExpression(u.Operand)
		if _, ok := u.Operand.Type().(parser.Ref).To.(parser.List); ok {
			e.write(".clone()")
		} else {
			e.write("(1)")
		}
	}
}

func (e *Emitter) emitReference(expr parser.Expression) {
	switch expr.(type) {
	case *parser.PropertyAccessExpression, *parser.ComputedAccessExpression:
		e.emitObjectReference(expr)
	default:
		e.emitPrimitiveReference(expr)
	}
}

func (e *Emitter) emitPrimitiveReference(expr parser.Expression) {
	e.write("(a,p)=>(a&4?__s:a&2?\"")
	e.emitExpression(expr)
	e.write("\":a?")
	e.emitExpression(expr)
	e.write(":void (")
	e.emitExpression(expr)
	e.write("=p))")
}
func (e *Emitter) emitObjectReference(expr parser.Expression) {
	e.write("((o,k)=>(a,p)=>(a&4?o:a&2?k:a?o[k]:void (o[k]=p)))(")
	switch expr := expr.(type) {
	case *parser.PropertyAccessExpression:
		e.emitExpression(expr.Expr)
		e.write(",\"")
		e.emitExpression(expr.Property)
		e.write("\")")
	case *parser.ComputedAccessExpression:
		e.emitExpression(expr.Expr)
		e.write(",")
		e.emitExpression(expr.Property)
		e.write(")")
	}
}
