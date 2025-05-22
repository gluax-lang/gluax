package parser

import (
	"github.com/gluax-lang/gluax/frontend/ast"
	"github.com/gluax-lang/gluax/frontend/common"
	"github.com/gluax-lang/gluax/frontend/lexer"
)

func (p *parser) parseStmt() ast.Stmt {
	switch p.Token.AsString() {
	case "let":
		return p.parseLet(false)
	case "return":
		return p.parseReturn()
	case "throw":
		return p.parseThrow()
	case "break":
		return p.parseBreak()
	case "continue":
		return p.parseContinue()
	default:
		return p.parseAssignmentOrStmtExpr()
	}
}

func (p *parser) parseLet(isItem bool) *ast.Let {
	spanStart := p.span()
	p.advance()

	var (
		names []lexer.TokIdent
		types []*ast.Type
	)

	for {
		id := p.expectIdentMsgX("expected variable name", FlagAllowUnderscore)
		names = append(names, id)

		var tyPtr *ast.Type
		if p.tryConsume(":") {
			t := p.parseType()
			tyPtr = &t
		}
		types = append(types, tyPtr)

		if !p.tryConsume(",") {
			break
		}
	}

	p.expect("=")

	var values []ast.Expr
	p.parseCommaSeparatedDelimited(";", func(p *parser) {
		values = append(values, p.parseExpr(ExprCtxNormal))
	})

	if len(values) == 0 {
		common.PanicDiag("missing right-hand side in let statement", p.prevSpan())
	}

	span := SpanFrom(spanStart, p.prevSpan())
	let := ast.NewLet(names, types, values, span, isItem)
	if isItem {
		let.IsGlobalDef = p.processingGlobals
	}
	return let
}

func (p *parser) parseReturn() ast.Stmt {
	spanStart := p.span()
	p.advance() // skip the initial `return`

	var exprs []ast.Expr
	if !p.tryConsume(";") {
		for {
			exprs = append(exprs, p.parseExpr(ExprCtxNormal))
			if p.tryConsume(",") {
				continue
			}
			p.expect(";")
			break
		}
	} else {
		// bare `return;` -> implicit `nil`
		exprs = append(exprs, ast.NewNilExpr(spanStart))
	}

	span := SpanFrom(spanStart, p.prevSpan())
	return ast.NewReturnStmt(exprs, span)
}

func (p *parser) parseThrow() ast.Stmt {
	spanStart := p.span()
	p.advance() // skip `throw`

	expr := p.parseExpr(ExprCtxNormal)

	p.expect(";")

	span := SpanFrom(spanStart, p.prevSpan())
	return ast.NewThrowStmt(expr, span)
}

func (p *parser) parseBreak() ast.Stmt {
	spanStart := p.span()
	p.advance() // skip `break`

	var label *ast.Ident
	if lexer.IsIdent(p.Token) {
		ident := p.expectIdent()
		label = &ident
	}

	p.expect(";")

	span := SpanFrom(spanStart, p.prevSpan())
	return ast.NewBreakStmt(label, span)
}

func (p *parser) parseContinue() ast.Stmt {
	spanStart := p.span()
	p.advance() // skip `continue`

	var label *ast.Ident
	if lexer.IsIdent(p.Token) {
		ident := p.expectIdent()
		label = &ident
	}

	p.expect(";")

	span := SpanFrom(spanStart, p.prevSpan())
	return ast.NewContinueStmt(label, span)
}

func isValidAssignmentTarget(expr ast.Expr) bool {
	switch expr.Kind() {
	case ast.ExprKindPath:
		return true // every plain path is an l-value

	case ast.ExprKindPostfix:
		pf := expr.Postfix()
		if _, ok := pf.Op.(*ast.DotAccess); ok {
			return true
		}
	}
	return false
}

func (p *parser) parseAssignmentOrStmtExpr() ast.Stmt {
	spanStart := p.span()
	firstExpr := p.parseExpr(ExprCtxNormal)

	if tok := p.Token.AsString(); tok != "," && tok != "=" {
		hasSemi := p.tryConsume(";") // semicolon means "statement expression", otherwise implicit return like in Rust
		endSpan := p.prevSpan()
		return ast.NewStmtExpr(firstExpr, hasSemi, SpanFrom(spanStart, endSpan))
	}

	lhsExprs := []ast.Expr{firstExpr}
	for p.tryConsume(",") {
		lhsExprs = append(lhsExprs, p.parseExpr(ExprCtxNormal))
	}

	p.expect("=")

	for _, lhs := range lhsExprs {
		if !isValidAssignmentTarget(lhs) {
			common.PanicDiag("invalid left-hand side of assignment", lhs.Span())
		}
	}

	var rhsExprs []ast.Expr
	rhsExprs = append(rhsExprs, p.parseExpr(ExprCtxNormal))
	for p.tryConsume(",") {
		rhsExprs = append(rhsExprs, p.parseExpr(ExprCtxNormal))
	}

	p.expect(";")

	return ast.NewAssignment(lhsExprs, rhsExprs, SpanFrom(spanStart, p.prevSpan()))
}
