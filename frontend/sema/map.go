package sema

import (
	"github.com/gluax-lang/gluax/common"
	"github.com/gluax-lang/gluax/frontend/ast"
)

func (a *Analysis) mapType(keyTy, valueTy Type, span common.Span) Type {
	if keyTy.IsNilable() || keyTy.IsNil() {
		a.Errorf(span, "map key type cannot be nilable/nil")
	}
	for _, m := range a.State.CreatedMaps {
		if a.MatchTypesStrict(m.Key, keyTy) && a.MatchTypesStrict(m.Value, valueTy) {
			return ast.NewSemType(m, span)
		}
	}
	mapT := ast.NewSemMap(keyTy, valueTy, span)
	a.State.CreatedMaps = append(a.State.CreatedMaps, mapT)
	return ast.NewSemType(mapT, span)
}
