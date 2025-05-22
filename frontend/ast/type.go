package ast

import (
	"github.com/gluax-lang/gluax/frontend/common"
	"github.com/gluax-lang/gluax/frontend/lexer"
)

type Type interface {
	isType()
	Span() common.Span
}

func NilType(span common.Span) Type {
	return &Path{
		Idents: []Ident{lexer.NewTokIdent("nil", span)},
	}
}

func IsOption(ty Type) bool {
	if gs, ok := ty.(*GenericStruct); ok {
		idents := gs.Path.Idents
		if len(idents) == 1 && idents[0].Raw == "option" {
			return true
		}
	}
	return false
}

func IsVararg(ty Type) bool {
	if _, ok := ty.(*Vararg); ok {
		return true
	}
	return false
}

/* Tuple */

type Tuple struct {
	Elems []Type
	span  common.Span
}

func NewTuple(elems []Type, span common.Span) *Tuple {
	return &Tuple{Elems: elems, span: span}
}

func (t *Tuple) isType() {}

func (t *Tuple) Span() common.Span {
	return t.span
}

/* Vararg */

type Vararg struct {
	span common.Span
}

func NewVararg(span common.Span) *Vararg {
	return &Vararg{span: span}
}

func (v *Vararg) isType() {}

func (v *Vararg) Span() common.Span {
	return v.span
}

/* GenericStruct */

type GenericStruct struct {
	Path     Path
	Generics []Type
	span     common.Span
}

func NewGenericStruct(path Path, generics []Type, span common.Span) *GenericStruct {
	return &GenericStruct{Path: path, Generics: generics, span: span}
}

func (g *GenericStruct) isType() {}

func (g *GenericStruct) Span() common.Span {
	return g.span
}
