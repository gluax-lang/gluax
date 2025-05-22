package parser

import (
	"github.com/gluax-lang/gluax/frontend/ast"
	"github.com/gluax-lang/gluax/frontend/common"
)

func (p *parser) parsePostfixExpr(ctx ExprCtx, left ast.Expr) ast.Expr {
	spanStart := left.Span()

	var op ast.PostfixOp
	switch {
	case p.Token.Is("("):
		op = p.parseCall(spanStart, nil)
	case p.Token.Is("unsafe_cast_as"):
		op = p.parseUnsafeCast()
	case p.Token.Is("else"):
		op = p.parseElse()
	case p.Token.Is("?"):
		op = p.parseUnwrapOption()
	case p.Token.Is("."):
		dotSpan := p.span()
		p.advance() // eat '.'

		field := p.expectIdent()
		if p.Token.Is("(") { // method call
			op = p.parseCall(dotSpan, &field)
		} else { // plain field access
			op = ast.NewDotAccess(field, dotSpan)
		}
	default:
		// nothing postfix-y ahead -> recursion ends
		return left
	}

	span := SpanFrom(spanStart, p.prevSpan())
	expr := ast.NewPostfixExpr(left, op, span)

	// Tail-recurse to see if *another* postfix operator follows
	return p.parsePostfixExpr(ctx, expr)
}

func (p *parser) parseCatch() *ast.Catch {
	spanStart := p.span()
	p.advance() // consume 'catch'
	name := p.expectIdent()
	block := p.parseBlock()
	span := SpanFrom(spanStart, p.prevSpan())
	return ast.NewCatch(name, block, span)
}

func (p *parser) parseCall(spanStart common.Span, method *ast.Ident) ast.PostfixOp {
	p.expect("(")

	var args []ast.Expr
	p.parseCommaSeparatedDelimited(")", func(p *parser) {
		args = append(args, p.parseExpr(ExprCtxNormal))
	})

	tryCall := false
	var catch *ast.Catch

	if p.tryConsume("!") {
		tryCall = true
	} else if p.Token.Is("catch") {
		catch = p.parseCatch()
	}

	spanStart = SpanFrom(spanStart, p.prevSpan())

	return ast.NewCall(method, args, tryCall, catch, spanStart)
}

func (p *parser) parseUnsafeCast() ast.PostfixOp {
	spanStart := p.span()
	p.advance() // consume 'as'
	ty := p.parseType()
	span := SpanFrom(spanStart, p.prevSpan())
	return ast.NewUnsafeCast(ty, span)
}

func (p *parser) parseElse() ast.PostfixOp {
	spanStart := p.span()
	p.advance() // consume 'else'
	value := p.parseExpr(ExprCtxNormal)
	span := SpanFrom(spanStart, p.prevSpan())
	return ast.NewElse(value, span)
}

func (p *parser) parseUnwrapOption() ast.PostfixOp {
	spanStart := p.span()
	p.advance() // consume '?'
	span := SpanFrom(spanStart, p.prevSpan())
	return ast.NewUnwrapOption(span)
}
