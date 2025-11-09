package parser

import (
	"github.com/gluax-lang/gluax/frontend/ast"
)

func (p *parser) parsePath(firstIdent *ast.Ident) ast.Path {
	return p.parsePathInternal(firstIdent, 0)
}

func (p *parser) parsePathInternal(firstIdent *ast.Ident, flags Flags) ast.Path {
	var segments []*ast.PathSegment

	var spanStart Span

	ident := ast.Ident{}
	if firstIdent != nil {
		ident = *firstIdent
		spanStart = firstIdent.Span()
	} else {
		ident = p.expectIdent()
		spanStart = ident.Span()
	}

	segments = append(segments, ast.NewPathSegment(ident, SpanFrom(spanStart, p.prevSpan())))

	for p.tryConsume("::") {
		ident := p.expectIdent()
		segments = append(segments, ast.NewPathSegment(ident, SpanFrom(ident.Span(), p.prevSpan())))
	}
	return ast.NewPath(segments)
}
