package parser

import (
	"github.com/gluax-lang/gluax/frontend/ast"
)

func (p *parser) parsePathExpr(ctx ExprCtx, ident *ast.Ident) ast.Expr {
	path := ast.Path{Idents: []ast.Ident{}, Generics: []ast.Type{}}

	if ident != nil {
		path.Idents = append(path.Idents, *ident)
	} else {
		path.Idents = append(path.Idents, p.expectIdent())
	}

	for p.tryConsume("::") {
		// `::<` -> turbofish generics
		if p.Token.Is("<") {
			generics := p.parseTurbofishGenerics()
			if p.Token.Is("{") {
				return p.parseStructInit(path, generics)
			} else {
				path.Generics = generics
				p.expect("::")
			}
		}

		// Ordinary path segment.
		path.Idents = append(path.Idents, p.expectIdent())

		if len(path.Generics) > 0 {
			break
		}
	}

	// Struct initializer without turbofish:  Foo::Bar { ... }
	if !ctx.IsCondition() && p.Token.Is("{") {
		return p.parseStructInit(path, nil)
	}

	return ast.NewExpr(&path)
}

// parseStructInitField parses a single `field: value` entry inside a struct
// initializer.
func (p *parser) parseStructInitField() ast.ExprStructField {
	name := p.expectIdent()
	p.expect(":")
	value := p.parseExpr(ExprCtxNormal)
	return ast.ExprStructField{
		Name:  name,
		Value: value,
	}
}

// parseTurbofishGenerics parses the `<T, U, V>` part after `::<`.
func (p *parser) parseTurbofishGenerics() []ast.Type {
	p.expect("<")
	var generics []ast.Type
	p.parseCommaSeparatedDelimited(">", func(p *parser) {
		generics = append(generics, p.parseType())
	})
	return generics
}

func (p *parser) parseStructInit(ty ast.Path, generics []ast.Type) ast.Expr {
	if ty.IsVec() {
		return p.parseVecInit(generics)
	}

	if ty.IsMap() {
		return p.parseMapInit(generics)
	}

	p.expect("{")

	var fields []ast.ExprStructField
	p.parseCommaSeparatedDelimited("}", func(p *parser) {
		fields = append(fields, p.parseStructInitField())
	})

	spanEnd := p.prevSpan()
	span := SpanFrom(ty.Span(), spanEnd)

	return ast.NewStructInit(ty, generics, fields, span)
}

func (p *parser) parseVecInit(generics []ast.Type) ast.Expr {
	spanStart := p.span()

	p.expect("{")

	var values []ast.Expr
	p.parseCommaSeparatedDelimited("}", func(p *parser) {
		values = append(values, p.parseExpr(ExprCtxNormal))
	})

	spanEnd := p.prevSpan()
	span := SpanFrom(spanStart, spanEnd)

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

func (p *parser) parseMapInit(generics []ast.Type) ast.Expr {
	spanStart := p.span()

	p.expect("{")

	var entries []ast.ExprMapEntry
	p.parseCommaSeparatedDelimited("}", func(p *parser) {
		entries = append(entries, p.parseMapEntry())
	})

	spanEnd := p.prevSpan()
	span := SpanFrom(spanStart, spanEnd)

	return ast.NewMapInitExpr(generics, entries, span)
}
