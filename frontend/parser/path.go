package parser

import (
	"github.com/gluax-lang/gluax/frontend/ast"
)

func (p *parser) parsePath() ast.Path {
	var idents []ast.Ident
	for {
		ident := p.expectIdent()
		idents = append(idents, ident)
		if !p.tryConsume("::") {
			break
		}
	}
	return ast.Path{Idents: idents}
}
