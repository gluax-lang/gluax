package parser

import (
	"github.com/gluax-lang/gluax/frontend/ast"
	"github.com/gluax-lang/gluax/frontend/common"
	"github.com/gluax-lang/gluax/frontend/lexer"
)

// ExprCtx tells the parser how to treat the upcoming expression.
//
// Normal     - ordinary expression parsing.
//
// Condition  - expression appears in a conditional position (e.g. `if ident {}`).
type ExprCtx int

const (
	ExprCtxNormal ExprCtx = iota
	ExprCtxCondition
)

func (c ExprCtx) IsCondition() bool {
	return c == ExprCtxCondition
}

func (c ExprCtx) IsNormal() bool {
	return c == ExprCtxNormal
}

func (c ExprCtx) String() string {
	switch c {
	case ExprCtxNormal:
		return "Normal"
	case ExprCtxCondition:
		return "Condition"
	default:
		panic("unreachable")
	}
}

func (p *parser) parseExpr(ctx ExprCtx) ast.Expr {
	return p.parseBinaryExpr(ctx, 0)
}

func (p *parser) parsePrimaryExpr(ctx ExprCtx) ast.Expr {
	switch v := p.Token.(type) {
	case lexer.TokIdent:
		// if its "nil", then return nil expr/value
		if v.Raw == "nil" {
			p.advance() // consume "nil"
			return ast.NewNilExpr(p.prevSpan())
		}
		return p.parsePathExpr(ctx, nil)
	case lexer.TokNumber:
		p.advance() // consume number
		return ast.NewNumberExpr(v)
	case lexer.TokString:
		p.advance() // consume string
		return ast.NewStringExpr(v)
	}

	if p.Token.Is("@") && lexer.IsIdentStr(p.peek(), "raw") {
		return p.parseRunRawExpr()
	}

	tok := p.Token
	switch tok.AsString() {
	case "_":
		common.PanicDiag("`_` can only be used to denote a variable/parameter name", tok.Span())
		panic("unreachable")
	case "Self":
		p.advance() // consume Self
		Self := lexer.NewTokIdent("Self", tok.Span())
		return p.parsePathExpr(ctx, &Self)
	case "true", "false":
		p.advance() // consume bool
		return ast.NewBoolExpr(tok)
	case "func":
		return p.parseFunctionExpr()
	case "if":
		return p.parseIfExpr()
	case "while":
		return p.parseWhileExpr()
	case "loop":
		return p.parseLoopExpr()
	case "{":
		block := p.parseBlock()
		blockExpr := ast.NewExpr(&block)
		return blockExpr
	case "(":
		return p.parseParenthesizedExpr()
	case "...":
		p.advance()
		return ast.NewVarargExpr(p.prevSpan())
	default:
		common.PanicDiag("expected expression", tok.Span())
		panic("unreachable")
	}
}

func (p *parser) parseFunctionExpr() ast.Expr {
	spanStart := p.span()
	p.advance() // skip `func`
	sig := p.parseFunctionSignature(FlagFuncParamVarArg | FlagFuncParamNamed)
	body := p.parseBlock()
	span := SpanFrom(spanStart, p.prevSpan())
	fun := ast.NewFunction(nil, sig, &body, nil, span)
	return ast.NewExpr(fun)
}

func (p *parser) parseIfExpr() ast.Expr {
	spanStart := p.span()

	p.advance() // consume "if"

	mainCond := p.parseExpr(ExprCtxCondition)
	mainBlock := p.parseBlock()

	mainGB := ast.NewGuardedBlock(mainCond, mainBlock)

	var branches []ast.GuardedBlock
	var elseBlock *ast.Block

	for p.tryConsume("else") {
		if p.tryConsume("if") {
			cond := p.parseExpr(ExprCtxCondition)
			body := p.parseBlock()
			gb := ast.NewGuardedBlock(cond, body)
			branches = append(branches, gb)
		} else {
			block := p.parseBlock()
			elseBlock = &block
			break
		}
	}

	return ast.NewIfExpr(mainGB, branches, elseBlock, SpanFrom(spanStart, p.prevSpan()))
}

func (p *parser) parseWhileExpr() ast.Expr {
	spanStart := p.span()

	p.advance() // consume "while"

	var label *ast.Ident
	if p.tryConsume(":") {
		ident := p.expectIdent()
		label = &ident
		p.expect(";")
	}

	cond := p.parseExpr(ExprCtxCondition)
	body := p.parseBlock()

	return ast.NewWhileExpr(label, cond, body, SpanFrom(spanStart, p.prevSpan()))
}

func (p *parser) parseLoopExpr() ast.Expr {
	spanStart := p.span()

	p.advance() // consume "loop"

	var label *ast.Ident
	if p.tryConsume(":") {
		ident := p.expectIdent()
		label = &ident
	}

	body := p.parseBlock()

	return ast.NewLoopExpr(label, body, SpanFrom(spanStart, p.prevSpan()))
}

func (p *parser) parseParenthesizedExpr() ast.Expr {
	spanStart := p.span()

	p.advance() // consume "("

	expr := p.parseExpr(ExprCtxNormal)

	if p.Token.Is(",") {
		values := []ast.Expr{expr}
		for p.tryConsume(",") {
			values = append(values, p.parseExpr(ExprCtxNormal))
		}
		p.expect(")")
		return ast.NewTupleExpr(values, SpanFrom(spanStart, p.prevSpan()))
	}

	p.expect(")")

	// return ast.NewParenthesizedExpr(expr, SpanFrom(spanStart, p.prevSpan()))
	return expr
}

func (p *parser) parseRunRawExpr() ast.Expr {
	spanStart := p.span()
	p.advance() // consume "@"
	p.advance() // consume "lua"
	atLuaSpan := SpanFrom(spanStart, p.prevSpan())

	p.expect("(")

	code := p.expectString()

	var args []ast.Expr
	for {
		if p.tryConsume(",") {
			args = append(args, p.parseExpr(ExprCtxNormal))
		} else {
			break
		}
	}

	p.expect(")")

	returnType := p.parseFunctionReturnType(FlagTypeTuple|FlagTypeVarArg|FlagFuncReturnUnreachable, atLuaSpan)

	return ast.NewRunRawExpr(code, args, returnType, SpanFrom(spanStart, p.prevSpan()))
}
