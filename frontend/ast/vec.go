package ast

import (
	"github.com/gluax-lang/gluax/common"
)

type Vec struct {
	Ty   Type // element type
	span common.Span
}

func NewVec(ty Type, span common.Span) *Vec {
	return &Vec{Ty: ty, span: span}
}

func (t *Vec) isType() {}

func (t *Vec) Span() common.Span {
	return t.span
}

type SemVec struct {
	Ty      SemType // element type
	Span_   common.Span
	Methods []*SemFunction
}

func NewSemVec(ty SemType, span common.Span) *SemVec {
	return &SemVec{Ty: ty, Span_: span}
}

func (u *SemVec) TypeKind() SemTypeKind {
	return SemVecKind
}

func (u *SemVec) String() string {
	return "vec<" + u.Ty.String() + ">"
}

func (u *SemVec) LSPString() string {
	return u.String()
}

func (u *SemVec) Span() common.Span {
	return u.Span_
}

func (t *SemType) IsVec() bool {
	return t.Kind() == SemVecKind
}

func (t *SemType) Vec() *SemVec {
	if !t.IsVec() {
		panic("not a vec type")
	}
	return t.data.(*SemVec)
}
