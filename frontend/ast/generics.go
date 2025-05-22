package ast

import (
	"strings"

	"github.com/gluax-lang/gluax/frontend/common"
	"github.com/gluax-lang/gluax/frontend/lexer"
)

type GenericParam struct {
	Name lexer.TokIdent
}

type Generics struct {
	Params []GenericParam
	Span   common.Span
}

func NewGenerics(params []GenericParam, span common.Span) Generics {
	return Generics{Params: params, Span: span}
}

func (gs Generics) Len() int {
	return len(gs.Params)
}

func (gs Generics) IsEmpty() bool {
	return len(gs.Params) == 0
}

func (gs Generics) String() string {
	if len(gs.Params) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteRune('<')
	for i, p := range gs.Params {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(p.Name.Raw)
	}
	sb.WriteRune('>')
	return sb.String()
}
