package emitter

import (
	"strings"
	"testing"

	"github.com/bmelicque/test-parser/parser"
)

type testToken struct {
	kind  parser.TokenKind
	value string
	loc   parser.Loc
}

func (t testToken) Kind() parser.TokenKind { return t.kind }
func (t testToken) Text() string           { return t.value }
func (t testToken) Loc() parser.Loc        { return t.loc }

func testEmitter(t *testing.T, source string, expected string, line int) {
	ast, _ := parser.Parse(strings.NewReader(source))
	emitter := makeEmitter()
	emitter.emit(ast[line])
	received := emitter.string()
	if emitter.string() != expected {
		t.Fatalf("expected output:\n%v\n\ngot:\n%v", expected, received)
	}
}
