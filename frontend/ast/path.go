package ast

import (
	"strings"

	"github.com/gluax-lang/gluax/common"
)

// Path represents a path to a symbol/class function/type.
// It is a sequence of identifiers separated by "::".
type Path struct {
	Segments       []*PathSegment
	ResolvedSymbol *Symbol // resolved symbol, if any
}

func NewPath(segments []*PathSegment) Path {
	return Path{Segments: segments}
}

func NewSimplePath(ident Ident) Path {
	return Path{
		Segments: []*PathSegment{
			{Ident: ident, Generics: nil, span: ident.Span()},
		},
	}
}

func NewSimplePathWithGenerics(ident Ident, generics []Type) Path {
	span := ident.Span()
	if len(generics) > 0 {
		span = common.SpanFrom(ident.Span(), generics[len(generics)-1].Span())
	}
	return Path{
		Segments: []*PathSegment{
			{Ident: ident, Generics: generics, span: span},
		},
	}
}

func (p *Path) isType() {}
func (p *Path) ExprKind() ExprKind {
	return ExprKindPath
}

func (p *Path) Span() common.Span {
	// from the first segment to the last
	return common.SpanFrom(p.Segments[0].Span(), p.Segments[len(p.Segments)-1].Span())
}

func (p *Path) IsSelf() bool {
	return len(p.Segments) == 1 && p.Segments[0].Ident.Raw == "Self"
}

func (p *Path) IsVec() bool {
	return len(p.Segments) == 1 && p.Segments[0].Ident.Raw == "vec"
}

func (p *Path) IsMap() bool {
	return len(p.Segments) == 1 && p.Segments[0].Ident.Raw == "map"
}

func (p *Path) String() string {
	var sb strings.Builder
	sb.WriteString(p.Segments[0].Ident.Raw)
	for i := 1; i < len(p.Segments); i++ {
		sb.WriteString("::")
		sb.WriteString(p.Segments[i].Ident.Raw)
	}
	return sb.String()
}

func (p *Path) LastIdent() Ident {
	return p.Segments[len(p.Segments)-1].Ident
}

func (p *Path) LastSegment() *PathSegment {
	return p.Segments[len(p.Segments)-1]
}

type PathSegment struct {
	Ident    Ident
	Generics []Type
	span     common.Span
}

func NewPathSegment(ident Ident, generics []Type, span common.Span) *PathSegment {
	return &PathSegment{Ident: ident, Generics: generics, span: span}
}

func (ps *PathSegment) Span() common.Span {
	return ps.span
}
