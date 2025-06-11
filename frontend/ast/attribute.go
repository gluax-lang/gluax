package ast

import (
	"slices"

	"github.com/gluax-lang/gluax/common"
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

// Has returns true if an attribute with any of the given keys exists.
// Panics if called with zero arguments.
func (attrs Attributes) Has(keys ...string) bool {
	if len(keys) == 0 {
		panic("Attributes.Has: at least one key must be provided")
	}
	for _, attr := range attrs {
		if slices.Contains(keys, attr.Key.Raw) {
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

// HasTokenTreeArgs returns true if there is an attribute #[key(args...)]
// where all args are present as identifier tokens in the token tree (order-insensitive).
func (attrs Attributes) HasTokenTreeArgs(key string, args ...string) bool {
	for _, attr := range attrs {
		if attr.Key.Raw == key && attr.IsInputTokenTree() {
			found := make(map[string]bool)
			for _, tok := range attr.TokenTree {
				switch t := tok.(type) {
				case lexer.TokIdent:
					found[t.Raw] = true
				case lexer.TokKeyword:
					found[t.String()] = true
				case lexer.TokString:
					found[t.Raw] = true
				case lexer.TokNumber:
					found[t.Raw] = true
				}
			}
			for _, arg := range args {
				if !found[arg] {
					return false
				}
			}
			return true
		}
	}
	return false
}
