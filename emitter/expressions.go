package emitter

import (
	"fmt"

	"github.com/bmelicque/test-parser/parser"
)

func (e *Emitter) emitCallExpression(expr *parser.CallExpression, await bool) {
	if expr.Callee.Type().(parser.Function).Async && await {
		e.write("await ")
	}
	e.emit(expr.Callee)
	e.write("(")
	defer e.write(")")

	args := expr.Args.Expr.(*parser.TupleExpression).Elements
	max := len(args) - 1
	for i := range args[:max] {
		e.emit(args[i])
		e.write(", ")
	}
	e.emit(args[max])
}

func (e *Emitter) emitListAccess(expr *parser.ComputedAccessExpression) {
	switch prop := expr.Property.Expr.(type) {
	case *parser.RangeExpression:
		e.write(".slice(")
		if prop.Left != nil {
			e.emitExpression(prop.Left)
		} else {
			e.write("0")
		}
		if prop.Right == nil {
			e.write(")")
			return
		}
		e.write(", ")
		e.emitExpression(prop.Right)
		if prop.Operator.Kind() == parser.InclusiveRange {
			e.write("+1")
		}
		e.write(")")
	default:
		e.write("[")
		e.emit(expr.Property.Expr)
		e.write("]")
	}
}
func (e *Emitter) emitComputedAccessExpression(expr *parser.ComputedAccessExpression) {
	switch left := expr.Expr.Type().(type) {
	case parser.TypeAlias:
		if left.Name == "Map" {
			emitMapElementAccess(e, expr)
		} else {
			e.emit(expr.Expr)
		}
	case parser.Ref:
		if _, ok := left.To.(parser.List); !ok {
			panic("unexpected typing (expected &[]any)")
		}
		e.emitExpression(expr.Expr)
		e.write("(")
		e.emitExpression(expr.Property.Expr)
		e.write(")")
	case parser.List:
		e.emitListAccess(expr)
	default:
		e.emit(expr.Expr)
	}
}
func emitMapElementAccess(e *Emitter, c *parser.ComputedAccessExpression) {
	e.emitExpression(c.Expr)
	e.write(".get(")
	e.emitExpression(c.Property.Expr)
	e.write(")")
}

func (e *Emitter) emitFunctionBody(b *parser.Block, params *parser.TupleExpression) {
	e.write("{")
	if len(b.Statements) == 0 {
		e.write("}")
		return
	}
	e.write("\n")
	e.depth++

	for _, param := range params.Elements {
		if _, ok := param.Type().(parser.Ref); ok {
			continue
		}
		name := param.(*parser.Param).Identifier.Text()
		v, ok := b.Scope().Find(name)
		if !ok {
			panic("variable should be found in current scope...")
		}
		if isMutated(v) {
			e.indent()
			e.write(fmt.Sprintf("%v = structuredClone(%v);\n", name, name))
		}
	}
	max := len(b.Statements) - 1
	for _, statement := range b.Statements[:max] {
		e.indent()
		e.emit(statement)
	}
	e.indent()
	e.write("return ")
	e.emit(b.Statements[max])
	e.write(";\n")
	e.depth--
	e.indent()
	e.write("}\n")
}

func (e *Emitter) emitFunctionExpression(f *parser.FunctionExpression) {
	if f.Type().(parser.Function).Async {
		e.write("async ")
	}
	e.write("(")
	args := f.Params.Expr.(*parser.TupleExpression).Elements
	max := len(args) - 1
	for i := range args[:max] {
		param := args[i].(*parser.Param)
		e.emit(param.Identifier)
		e.write(", ")
	}
	e.emit(args[max].(*parser.Param).Identifier)
	e.write(") => ")

	params := f.Params.Expr.(*parser.TupleExpression)
	e.emitFunctionBody(f.Body, params)
}

func (e *Emitter) emitIdentifier(i *parser.Identifier) {
	text := i.Token.Text()
	if text == e.thisName {
		e.write("this")
		return
	}
	e.write(getSanitizedName(text))
}

func (e *Emitter) emitPropertyAccessExpression(p *parser.PropertyAccessExpression) {
	e.emit(p.Expr)
	if _, ok := p.Expr.Type().(parser.Tuple); ok {
		e.write("[")
		e.emit(p.Property)
		e.write("]")
	} else {
		e.write(".")
		e.emit(p.Property)
	}
}

func (e *Emitter) emitRangeExpression(r *parser.RangeExpression) {
	e.addFlag(RangeFlag)

	e.write("_range(")

	if r.Left != nil {
		e.emit(r.Left)
	} else {
		e.write("0")
	}

	e.write(", ")

	if r.Right != nil {
		e.emit(r.Right)
		if r.Operator.Kind() == parser.InclusiveRange {
			e.write(" + 1")
		}
	} else {
		e.write("1")
	}

	e.write(")")
}

func (e *Emitter) emitTupleExpression(t *parser.TupleExpression) {
	if len(t.Elements) == 1 {
		e.emit(t.Elements[0])
		return
	}
	e.write("[")
	length := len(t.Elements)
	for i, el := range t.Elements {
		e.emit(el)
		if i != length-1 {
			e.write(", ")
		}
	}
	e.write("]")
}

func (e *Emitter) emitReference(expr parser.Expression) {
	e.write("function (_) { return arguments.length ? void (")
	e.emit(expr)
	e.write(" = _) : ")
	e.emit(expr)
	e.write(" }")
}
func (e *Emitter) emitSlice(expr parser.Expression) {
	var r *parser.RangeExpression
	if computed, ok := expr.(*parser.ComputedAccessExpression); ok {
		expr = computed.Expr
		r = computed.Property.Expr.(*parser.RangeExpression)
	}
	e.addFlag(SliceFlag)
	e.write("__slice(() => ")
	e.emit(expr)
	if r != nil {
		e.write(", ")
		if r.Left != nil {
			e.emit(r.Left)
		} else {
			e.write("0")
		}
		if r.Right != nil {
			e.write(", ")
			e.emit(r.Right)
		}
	}
	e.write(")")
}
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
		if _, ok := u.Operand.Type().(parser.List); ok {
			e.emitSlice(u.Operand)
		} else {
			e.emitReference(u.Operand)
		}
	case parser.Mul:
		e.emit(u.Operand)
		e.write("()")
	}
}
