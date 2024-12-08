package emitter

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/bmelicque/test-parser/parser"
)

type EmitterFlag int

const (
	NoFlags   EmitterFlag = 0
	PanicFlag EmitterFlag = 1 << iota
	SliceFlag
	SumFlag
)

type Emitter struct {
	depth        int
	flags        EmitterFlag
	builder      strings.Builder
	thisName     string
	constructors map[string]map[string]parser.Expression
	uninlinables map[parser.Node]int
}

func makeEmitter() *Emitter {
	return &Emitter{
		depth:        0,
		flags:        NoFlags,
		builder:      strings.Builder{},
		constructors: map[string]map[string]parser.Expression{},
		uninlinables: map[parser.Node]int{},
	}
}

func (e *Emitter) addFlag(flag EmitterFlag) {
	e.flags |= flag
}

func (e *Emitter) hasFlag(flag EmitterFlag) bool {
	return (e.flags & flag) != NoFlags
}

func (e *Emitter) write(str string) {
	e.builder.WriteString(str)
}

func (e *Emitter) indent() {
	for i := 0; i < e.depth; i++ {
		e.builder.WriteString("    ")
	}
}

func (e Emitter) string() string {
	return e.builder.String()
}

func (e *Emitter) emit(node parser.Node) {
	//TODO: if not node that needs extraction, look if contains one
	if !isUninlinable(node) {
		e.extractUninlinables(node)
	}
	switch node := node.(type) {
	// Statements
	case *parser.Assignment:
		e.emitAssignment(node)
	case *parser.Block:
		e.emitBlockStatement(node)
	case *parser.CatchExpression:
		e.emitCatchStatement(node)
	case *parser.ForExpression:
		e.emitFor(node)
	case *parser.IfExpression:
		e.emitIfStatement(node)
	case *parser.MatchExpression:
		e.emitMatchStatement(*node)
	case *parser.Exit:
		e.emitExit(node)
	case parser.Expression:
		e.emitExpression(node)
	default:
		panic(fmt.Sprintf("Cannot emit type '%v' (not implemented yet)", reflect.TypeOf(node)))
	}
}

func (e *Emitter) emitExpression(expr parser.Expression) {
	switch expr := expr.(type) {
	case *parser.Block:
		e.emitBlockExpression(expr)
	case *parser.BinaryExpression:
		e.emitBinaryExpression(expr)
	case *parser.CallExpression:
		e.emitCallExpression(expr, true)
	case *parser.CatchExpression:
		id, ok := e.uninlinables[expr]
		if !ok {
			panic("Catch expression should have been extracted!")
		}
		e.write(fmt.Sprintf("_tmp%v", id))
		delete(e.uninlinables, expr)
	case *parser.ComputedAccessExpression:
		e.emitComputedAccessExpression(expr)
	case *parser.FunctionExpression:
		e.emitFunctionExpression(expr)
	case *parser.Identifier:
		e.emitIdentifier(expr)
	case *parser.IfExpression:
		e.emitIfExpression(expr)
	case *parser.InstanceExpression:
		e.emitInstanceExpression(expr)
	case *parser.Literal:
		e.write(expr.Token.Text())
	case *parser.ParenthesizedExpression:
		e.write("(")
		e.emit(expr.Expr)
		e.write(")")
	case *parser.PropertyAccessExpression:
		e.emitPropertyAccessExpression(expr)
	case *parser.TupleExpression:
		e.emitTupleExpression(expr)
	case *parser.UnaryExpression:
		e.emitUnaryExpression(expr)
	}
}

func EmitProgram(nodes []parser.Node) string {
	e := makeEmitter()
	for _, node := range nodes {
		e.emit(node)
	}
	e.write("\n")
	if e.hasFlag(PanicFlag) {
		e.emitPanic()
	}
	if e.hasFlag(SliceFlag) {
		e.emitSliceConstructor()
	}
	if e.hasFlag(SumFlag) {
		e.write("class _Sum {\n")
		e.write("    constructor(_tag, _value) {\n")
		e.write("        this._tag = _tag;\n")
		e.write("        if (arguments.length > 1) { this._value = _value }\n")
		e.write("    }\n}\n")
	}
	return e.string()
}

func (e *Emitter) emitPanic() {
	e.write(`
function panic(msg) {
	const div = document.createElement("div");
	div.addEventListener("click", () => {
		throw msg;
	});
	div.click();
}`)
}

func (e *Emitter) emitSliceConstructor() {
	e.write(`class __Slice {
	constructor(getter, start = 0, end = getter().length) {
		this._ = getter;
		this.start = start;
		this.end = end;
		this.length = end - this.start;
	}

	get(index) {
		return index < this.length ? this._()[this.start + index] : undefined;
	}

	set(index, value) {
		if (index >= this.length) throw new OutOfRangeError();
		this._()[this.start + index] = value;
	}

	clone() {
		return this._().slice(this.start, this.end);
	}

	ref(index) {
		if (index >= this.length) throw new OutOfRangeError();
		return function (value) {
			if (arguments.length === 0) return this._()[this.start + index];
			this._()[this.start + index] = value;
		};
	}

	*[Symbol.iterator]() {
		let i = 0;
		while (i++ < this.length) yield this.ref(i);
	}
}`)
}
