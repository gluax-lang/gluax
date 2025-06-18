package parser

import (
	"github.com/gluax-lang/gluax/frontend/ast"
)

func (p *parser) parsePathExpr(ctx ExprCtx, ident *ast.Ident) ast.Expr {
	path := p.parsePathInternal(ident, FlagTurboFishGenerics)

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
	if path.IsVec() {
		return p.parseVecInit(path)
	}
	if path.IsMap() {
		return p.parseMapInit(path)
	}

	p.expect("{")

	var fields []ast.ExprClassField
	p.parseCommaSeparatedDelimited("}", func(p *parser) {
		fields = append(fields, p.parseClassInitField())
	})

	spanEnd := p.prevSpan()
	span := SpanFrom(path.Span(), spanEnd)

	return ast.NewClassInit(path, fields, span)
}

func (p *parser) parseVecInit(path ast.Path) ast.Expr {
	spanStart := p.span()

	p.expect("{")

	var values []ast.Expr
	p.parseCommaSeparatedDelimited("}", func(p *parser) {
		values = append(values, p.parseExpr(ExprCtxNormal))
	})

	spanEnd := p.prevSpan()
	span := SpanFrom(spanStart, spanEnd)

	var generics []ast.Type
	if len(path.Segments) > 0 && len(path.Segments[0].Generics) > 0 {
		generics = path.Segments[0].Generics
	}

	return ast.NewVecInitExpr(generics, values, span)
}

func (p *parser) parseMapEntry() ast.ExprMapEntry {
	key := p.parseExpr(ExprCtxNormal)
	p.expect(":")
	value := p.parseExpr(ExprCtxNormal)
	return ast.ExprMapEntry{
		Key:   key,
		Value: value,
	}
}

func (p *parser) parseMapInit(path ast.Path) ast.Expr {
	spanStart := p.span()

	p.expect("{")

	var entries []ast.ExprMapEntry
	p.parseCommaSeparatedDelimited("}", func(p *parser) {
		entries = append(entries, p.parseMapEntry())
	})

	spanEnd := p.prevSpan()
	span := SpanFrom(spanStart, spanEnd)

	var generics []ast.Type
	if len(path.Segments) > 0 && len(path.Segments[0].Generics) > 0 {
		generics = path.Segments[0].Generics
	}

	return ast.NewMapInitExpr(generics, entries, span)
}
