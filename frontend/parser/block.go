package parser

import "github.com/gluax-lang/gluax/frontend/ast"

func (p *parser) parseBlock() ast.Block {
	spanStart := p.span()

	p.expect("{")

	var stmts []ast.Stmt
	for !p.Token.Is("}") {
		stmts = append(stmts, p.parseStmt())
	}

	p.expect("}")

	span := SpanFrom(spanStart, p.prevSpan())
	return ast.NewBlock(stmts, span)
}
