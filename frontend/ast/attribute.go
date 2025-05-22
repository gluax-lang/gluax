package ast

import (
	"github.com/gluax-lang/gluax/frontend/common"
	"github.com/gluax-lang/gluax/frontend/lexer"
)

type AttributeInputKind uint8

const (
	AttrInputNone      AttributeInputKind = iota // `#[foo]`
	AttrInputTokenTree                           // `#[foo(bar, baz)]`
	AttrInputString                              // `#[foo = "bar"]`
)

// Attribute
//
//	#[foo]                    -> Path = foo
//	#[foo(bar, baz)]          -> Path = foo, Kind = TokenTree
//	#[foo = "bar"]            -> Path = foo, Kind = String, String = "bar"
//
type Attribute struct {
	// Simple path that identifies the attribute (`foo::bar`).
	Key Ident

	// What kind of input follows the path.
	Kind AttributeInputKind

	// When Kind == AttrInputTokenTree, every token inside the outer‐most
	// delimiter is preserved here *as-is* (no interpretation).
	TokenTree []lexer.Token

	// When Kind == AttrInputString, holds the string literal that appeared
	// after the `=`.
	String *lexer.TokString

	// For diagnostics.
	Span common.Span
}

func (a Attribute) IsInputNone() bool {
	return a.Kind == AttrInputNone
}

func (a Attribute) IsInputTokenTree() bool {
	return a.Kind == AttrInputTokenTree
}

func (a Attribute) IsInputString() bool {
	return a.Kind == AttrInputString
}
