package ast

import (
	"strings"

	"github.com/gluax-lang/gluax/common"
)

// Path represents a path to a symbol/struct function/type.
// It is a sequence of identifiers separated by "::".
type Path struct {
	Idents         []Ident
	Generics       []Type  // generic parameters
	ResolvedSymbol *Symbol // resolved symbol, if any
}

func NewPath(idents []Ident) Path {
	return Path{Idents: idents}
}

func (p *Path) isType() {}
func (p *Path) ExprKind() ExprKind {
	return ExprKindPath
}

func (p *Path) Span() common.Span {
	// from the first ident to the last
	return common.SpanFrom(p.Idents[0].Span(), p.Idents[len(p.Idents)-1].Span())
}

func (p *Path) IsSelf() bool {
	return len(p.Idents) == 1 && p.Idents[0].Raw == "Self"
}

func (p *Path) IsVec() bool {
	return len(p.Idents) == 1 && p.Idents[0].Raw == "vec"
}

func (p *Path) IsMap() bool {
	return len(p.Idents) == 1 && p.Idents[0].Raw == "map"
}

func (p *Path) String() string {
	var sb strings.Builder
	sb.WriteString(p.Idents[0].Raw)
	for i := 1; i < len(p.Idents); i++ {
		sb.WriteString("::")
		sb.WriteString(p.Idents[i].Raw)
	}
	return sb.String()
}

func (p *Path) ToSnakeCase() string {
	var sb strings.Builder
	for i, ident := range p.Idents {
		if i > 0 {
			sb.WriteString("_")
		}
		sb.WriteString(strings.ToLower(ident.Raw))
	}
	return sb.String()
}
