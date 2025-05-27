package parser

import "github.com/gluax-lang/gluax/frontend/ast"

func (p *parser) parseUnaryExpr(ctx ExprCtx) ast.Expr {
	spanStart := p.span()
	switch p.Token.AsString() {
	case "-":
		p.advance()
		operand := p.parseUnaryExpr(ctx)
		return ast.NewUnaryExpr(ast.UnaryOpNegate, operand, SpanFrom(spanStart, p.prevSpan()))
	case "!":
		p.advance()
		operand := p.parseUnaryExpr(ctx)
		return ast.NewUnaryExpr(ast.UnaryOpNot, operand, SpanFrom(spanStart, p.prevSpan()))
	case "~":
		p.advance()
		operand := p.parseUnaryExpr(ctx)
		return ast.NewUnaryExpr(ast.UnaryOpBitwiseNot, operand, SpanFrom(spanStart, p.prevSpan()))
	case "#":
		p.advance()
		operand := p.parseUnaryExpr(ctx)
		return ast.NewUnaryExpr(ast.UnaryOpLength, operand, SpanFrom(spanStart, p.prevSpan()))
	}
	return p.parsePostfixExpr(ctx, p.parsePrimaryExpr(ctx))
}
