package parser

import (
	"github.com/gluax-lang/gluax/common"
	"github.com/gluax-lang/gluax/frontend/ast"
)

func (p *parser) parseGenerics() ast.Generics {
	// Empty initial value (no generics) - gets returned if we don't see '<'.
	g := ast.NewGenerics(nil, common.SpanDefault())

	spanStart := p.span()

	// No '<'  ->  early-return the zero struct.
	if !p.tryConsume("<") {
		return g
	}

	// Collect identifiers until '>'.
	p.parseCommaSeparatedDelimited(">", func(p *parser) {
		ident := p.expectIdent()
		param := ast.GenericParam{Name: ident}
		g.Params = append(g.Params, param)
	})

	g.Span = SpanFrom(spanStart, p.prevSpan())
	return g
}
