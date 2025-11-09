package parser

import (
	"github.com/gluax-lang/gluax/frontend/ast"
)

func (p *parser) parsePathExpr(ctx ExprCtx, ident *ast.Ident) ast.Expr {
	path := p.parsePathInternal(ident, 0)

	if !ctx.IsCondition() && p.Token.Is("{") {
		return p.parseClassInit(path)
	}

	pathExpr := ast.NewExpr(&path)
	return pathExpr
}

// parseClassInitField parses a single `field: value` entry inside a class
// initializer.
func (p *parser) parseClassInitField() ast.ExprClassField {
	name := p.expectIdent()
	p.expect(":")
	value := p.parseExpr(ExprCtxNormal)
	return ast.ExprClassField{Name: name, Value: value}
}

func (p *parser) parseClassInit(path ast.Path) ast.Expr {
	p.expect("{")

	var fields []ast.ExprClassField
	p.parseCommaSeparatedDelimited("}", func(p *parser) {
		fields = append(fields, p.parseClassInitField())
	})

	spanEnd := p.prevSpan()
	span := SpanFrom(path.Span(), spanEnd)

	return ast.NewClassInit(path, fields, span)
}
