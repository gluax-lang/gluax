package parser

import (
	"github.com/gluax-lang/gluax/frontend/ast"
)

func (p *parser) parsePathExpr(ctx ExprCtx, ident *ast.Ident) ast.Expr {
	path := ast.Path{Idents: []ast.Ident{}, Generics: make(map[ast.Ident][]ast.Type)}

	if ident != nil {
		path.Idents = append(path.Idents, *ident)

		if p.tryConsume("::") {
			path.Idents = append(path.Idents, p.expectIdent())
			return p.parsePathCall(path, nil, false)
		}
	} else {
		currentIdent := p.expectIdent()
		path.Idents = append(path.Idents, currentIdent)

		for p.tryConsume("::") {
			// `::<` -> turbofish generics
			if p.Token.Is("<") {
				generics := p.parseTurbofishGenerics()
				if p.Token.Is("{") {
					return p.parseStructInit(path, generics)
				} else {
					path.Generics[currentIdent] = generics
					return p.parsePathCall(path, generics, true)
				}
			}

			// Ordinary path segment.
			currentIdent = p.expectIdent()
			path.Idents = append(path.Idents, currentIdent)
		}
	}

	// Struct initializer without turbofish:  Foo::Bar { ... }
	if !ctx.IsCondition() && p.Token.Is("{") {
		return p.parseStructInit(path, nil)
	}

	if len(path.Idents) > 1 && p.Token.Is("(") {
		return p.parsePathCall(path, nil, false)
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
	spanStart := p.span()
	p.expect("{")

	var fields []ast.ExprStructField
	p.parseCommaSeparatedDelimited("}", func(p *parser) {
		fields = append(fields, p.parseStructInitField())
	})

	spanEnd := p.prevSpan()
	span := SpanFrom(spanStart, spanEnd)

	return ast.NewStructInit(ty, generics, fields, span)
}

func (p *parser) parsePathCall(path ast.Path, generics []ast.Type, parseMethod bool) ast.Expr {
	var methodName ast.Ident
	if parseMethod {
		p.expect("::")
		methodName = p.expectIdent()
	} else {
		methodName = path.Idents[len(path.Idents)-1]
		path.Idents = path.Idents[:len(path.Idents)-1]
	}
	call := p.parseCall(methodName.Span(), nil)
	span := SpanFrom(path.Span(), p.prevSpan())
	return ast.NewPathCall(path, methodName, generics, *call.(*ast.Call), span)
}
