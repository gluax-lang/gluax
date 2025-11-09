package ast

import (
	"strings"

	"github.com/gluax-lang/gluax/common"
)

type Union struct {
	Types []Type // The types that form the union
	span  common.Span
}

func NewUnion(types []Type, span common.Span) *Union {
	return &Union{Types: types, span: span}
}

func (t *Union) isType() {}

func (t *Union) Span() common.Span {
	return t.span
}

type SemUnion struct {
	*Union
	Types []SemType // The types that form the union
}

func NewSemUnion(union *Union, types []SemType) *SemUnion {
	return &SemUnion{Union: union, Types: types}
}

func (u *SemUnion) TypeKind() SemTypeKind {
	return SemUnionKind
}

func (u *SemUnion) String() string {
	parts := make([]string, len(u.Types))
	for i, t := range u.Types {
		parts[i] = t.String()
	}
	return strings.Join(parts, " | ")
}

func (u *SemUnion) LSPString() string {
	return u.String()
}

func (u *SemUnion) Span() common.Span {
	return u.Union.Span()
}

func (t *SemType) IsUnion() bool {
	return t.Kind() == SemUnionKind
}

func (t *SemType) Union() *SemUnion {
	if !t.IsUnion() {
		panic("not a union type")
	}
	return t.data.(*SemUnion)
}
