package ast

import (
	"github.com/gluax-lang/gluax/common"
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
	if gs, ok := ty.(*GenericClass); ok {
		idents := gs.Path.Idents
		if len(idents) == 1 && idents[0].Raw == "option" {
			return true
		}
	}
	return false
}

func IsSelf(ty Type) bool {
	if p, ok := ty.(*Path); ok {
		idents := p.Idents
		if len(idents) == 1 && idents[0].Raw == "Self" {
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
	Type Type
	span common.Span
}

func NewVararg(ty Type, span common.Span) *Vararg {
	return &Vararg{Type: ty, span: span}
}

func (v *Vararg) isType() {}

func (v *Vararg) Span() common.Span {
	return v.span
}

/* GenericClass */

type GenericClass struct {
	Path     Path
	Generics []Type
	span     common.Span
}

func NewGenericClass(path Path, generics []Type, span common.Span) *GenericClass {
	return &GenericClass{Path: path, Generics: generics, span: span}
}

func (g *GenericClass) isType() {}

func (g *GenericClass) Span() common.Span {
	return g.span
}

/* Unreachable */
type Unreachable struct {
	span common.Span
}

func NewUnreachable(span common.Span) *Unreachable {
	return &Unreachable{span: span}
}

func (u *Unreachable) isType() {}

func (u *Unreachable) Span() common.Span {
	return u.span
}

/* Dyn Trait */

type DynTrait struct {
	Trait Path
	span  common.Span
}

func NewDynTrait(trait Path, span common.Span) *DynTrait {
	return &DynTrait{Trait: trait, span: span}
}

func (it *DynTrait) isType() {}

func (it *DynTrait) Span() common.Span {
	return it.span
}
