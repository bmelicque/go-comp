package emitter

import (
	"slices"

	"github.com/bmelicque/test-parser/parser"
)

func (e *Emitter) emitIdentifier(i *parser.Identifier) {
	text := i.Token.Text()
	if text == e.thisName {
		e.write("this")
		return
	}
	if text == "panic" {
		e.addFlag(PanicFlag)
	}
	e.write(getSanitizedName(text))
}

var reservedWords = []string{
	"abstract", "arguments", "boolean", "byte", "case", "char", "class",
	"const", "debugger", "default", "delete", "do", "double", "enum", "eval",
	"export", "extends", "final", "finally", "float", "function", "goto",
	"implements", "import", "in", "instanceof", "int", "interface", "let",
	"long", "native", "new", "null", "package", "private", "protected",
	"public", "short", "static", "super", "switch", "synchronized", "this",
	"throw", "throws", "transient", "typeof", "var", "void", "volatile",
	"while", "with", "yield",
}

func getSanitizedName(name string) string {
	if slices.Contains(reservedWords, name) {
		return name + "_"
	}
	return name
}
