package ast

import (
	"github.com/gluax-lang/gluax/common"
)

type Map struct {
	Key   Type // key type
	Value Type // value type
	span  common.Span
}

func NewMap(key Type, value Type, span common.Span) *Map {
	return &Map{Key: key, Value: value, span: span}
}

func (t *Map) isType() {}

func (t *Map) Span() common.Span {
	return t.span
}

type SemMap struct {
	Key   SemType // key type
	Value SemType // value type
	Span_ common.Span
}

func NewSemMap(key SemType, value SemType, span common.Span) *SemMap {
	return &SemMap{Key: key, Value: value, Span_: span}
}

func (u *SemMap) TypeKind() SemTypeKind {
	return SemMapKind
}

func (u *SemMap) String() string {
	return "map<" + u.Key.String() + ", " + u.Value.String() + ">"
}

func (u *SemMap) LSPString() string {
	return u.String()
}

func (u *SemMap) Span() common.Span {
	return u.Span_
}

func (t *SemType) IsMap() bool {
	return t.Kind() == SemMapKind
}

func (t *SemType) Map() *SemMap {
	if !t.IsMap() {
		panic("not a map type")
	}
	return t.data.(*SemMap)
}
