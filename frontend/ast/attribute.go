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

type Attributes []Attribute

// Has returns true if an attribute with the given key exists.
func (attrs Attributes) Has(key string) bool {
	for _, attr := range attrs {
		if attr.Key.Raw == key {
			return true
		}
	}
	return false
}

// Get returns the first attribute with the given key, or nil if not found.
func (attrs Attributes) Get(key string) *Attribute {
	for _, attr := range attrs {
		if attr.Key.Raw == key {
			return &attr
		}
	}
	return nil
}

func (attrs Attributes) GetAll(key string) []Attribute {
	var result []Attribute
	for _, attr := range attrs {
		if attr.Key.Raw == key {
			result = append(result, attr)
		}
	}
	return result
}

// GetString returns the string value of the first attribute with the given key,
// or empty string if not found or not a string attribute.
func (attrs Attributes) GetString(key string) *string {
	attr := attrs.Get(key)
	if attr != nil && attr.IsInputString() && attr.String != nil {
		return &attr.String.Raw
	}
	return nil
}
