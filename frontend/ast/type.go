package ast

import (
	"github.com/gluax-lang/gluax/common"
)

type Type interface {
	isType()
	Span() common.Span
}

func IsNilable(ty Type) bool {
	if p, ok := ty.(*Path); ok {
		// A path is nilable if it's `nilable<T>`
		return len(p.Segments) == 1 && p.Segments[0].Ident.Raw == "nilable" && len(p.Segments[0].Generics) > 0
	}
	return false
}

func IsSelf(ty Type) bool {
	if p, ok := ty.(*Path); ok {
		return p.IsSelf()
	}
	return false
}

func IsVararg(ty Type) bool {
	_, ok := ty.(*Vararg)
	return ok
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
