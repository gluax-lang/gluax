package sema

import (
	"github.com/gluax-lang/gluax/common"
	"github.com/gluax-lang/gluax/frontend/ast"
)

func (a *Analysis) vecType(ty Type, span common.Span) Type {
	for _, v := range a.State.CreatedVecs {
		if a.MatchTypesStrict(v.Ty, ty) {
			return ast.NewSemType(v, span)
		}
	}
	vecT := ast.NewSemVec(ty, span)
	a.State.CreatedVecs = append(a.State.CreatedVecs, vecT)
	return ast.NewSemType(vecT, span)
}
