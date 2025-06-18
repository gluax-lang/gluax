package parser

import (
	"github.com/gluax-lang/gluax/frontend/ast"
)

func (p *parser) parsePath(firstIdent *ast.Ident) ast.Path {
	return p.parsePathInternal(firstIdent, 0)
}

func (p *parser) parsePathInternal(firstIdent *ast.Ident, flags Flags) ast.Path {
	var segments []*ast.PathSegment

	ident := ast.Ident{}
	if firstIdent != nil {
		ident = *firstIdent
	} else {
		ident = p.expectIdent()
	}

	generics := p.parseOptionalGenerics(flags)
	segments = append(segments, ast.NewPathSegment(ident, generics))

	for p.tryConsume("::") {
		ident := p.expectIdent()
		generics := p.parseOptionalGenerics(flags)
		segments = append(segments, ast.NewPathSegment(ident, generics))
	}
	return ast.NewPath(segments)
}

func (p *parser) parseOptionalGenerics(flags Flags) []ast.Type {
	if flags&FlagTurboFishGenerics != 0 {
		if p.Token.Is("::") && p.peek().Is("<") {
			p.advance() // Consume '::'
			p.advance() // Consume '<'
			return p.parseGenericList()
		}
		return nil
	}

	if p.tryConsume("<") {
		return p.parseGenericList()
	}
	return nil
}

func (p *parser) parseGenericList() []ast.Type {
	var generics []ast.Type

	if p.tryConsume(">") {
		return generics
	}

	p.parseCommaSeparatedDelimited(">", func(p *parser) {
		generics = append(generics, p.parseType())
	})
	return generics
}
