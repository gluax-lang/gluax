package parser

import "github.com/gluax-lang/gluax/frontend/ast"

func (p *parser) parseQPath() *ast.QPath {
	spanStart := p.span()
	p.advance() // skip `<`

	ty := p.parseType()

	p.expect("as")

	as := p.parsePath()

	p.expect(">")

	p.expect("::")

	methodName := p.expectIdent()

	span := SpanFrom(spanStart, p.prevSpan())

	return ast.NewQPath(ty, as, methodName, span)
}

func (p *parser) parseQPathExpr() ast.Expr {
	return ast.NewExpr(p.parseQPath())
}
