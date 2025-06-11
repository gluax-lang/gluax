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
		g.Params = append(g.Params, p.parseGenericParam())
	})

	g.Span = SpanFrom(spanStart, p.prevSpan())
	return g
}

func (p *parser) parseGenericParam() ast.GenericParam {
	spanStart := p.span()
	ident := p.expectIdent()
	var constraints []ast.Path
	if p.tryConsume(":") {
		for {
			constraints = append(constraints, p.parsePath())
			if !p.tryConsume("+") {
				break
			}
		}
	}
	return ast.GenericParam{
		Name:        ident,
		Constraints: constraints,
		Span:        SpanFrom(spanStart, p.prevSpan()),
	}
}
