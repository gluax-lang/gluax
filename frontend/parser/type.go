package parser

import (
	"github.com/gluax-lang/gluax/common"
	"github.com/gluax-lang/gluax/frontend/ast"
	"github.com/gluax-lang/gluax/frontend/lexer"
)

func (p *parser) parseTypeX(flags Flags) ast.Type {
	spanStart := p.span()

	var ty ast.Type

	if flags.Has(FlagFuncReturnUnreachable) && p.tryConsume("unreachable") {
		ty = ast.NewUnreachable(spanStart)
	} else if p.tryConsume("?") {
		if p.Token.Is("?") {
			common.PanicDiag("cannot have nested nilable types", p.span())
		}
		qSpan := p.prevSpan()
		innerType := p.parseType() // no flags, because tuple/vararg can't be nilable

		nilPath := ast.NewSimplePath(lexer.NewTokIdent("nil", qSpan))
		nilType := &nilPath

		ty = ast.NewUnion([]ast.Type{nilType, innerType}, SpanFrom(spanStart, p.prevSpan()))
	} else if p.Token.Is("func") {
		ty = p.parseFunctionType()
	} else if flags.Has(FlagTypeTuple) && p.Token.Is("(") {
		ty = p.parseTupleType(flags)
	} else if flags.Has(FlagTypeVarArg) && p.Token.Is("...") {
		p.advance()
		ty = ast.NewVararg(p.parseType(), SpanFrom(spanStart, p.prevSpan()))
	} else if lexer.IsIdent(p.Token) && p.Token.String() == "vec" {
		ty = p.parseVec()
	} else {
		path := p.parsePath(nil)
		ty = &path
	}

	if p.Token.Is("|") {
		ty = p.parseUnionType(ty, flags, spanStart)
	}

	return ty
}

func (p *parser) parseType() ast.Type {
	return p.parseTypeX(0)
}

func (p *parser) parseFunctionType() ast.Type {
	spanStart := p.span()
	p.advance() // skip `func`

	sig := p.parseFunctionSignature(FlagFuncParamVarArg)
	span := SpanFrom(spanStart, p.prevSpan())
	return ast.NewFunction(nil, sig, nil, nil, span)
}

func (p *parser) parseTupleType(flags Flags) ast.Type {
	start := p.span()
	p.advance() // consume '('

	var elems []ast.Type
	cleanFlags := flags.Clear(FlagTypeTuple)

	for !p.Token.Is(")") {
		ty := p.parseTypeX(cleanFlags)
		elems = append(elems, ty)
		// If this was a vararg, it must be the last element
		if _, ok := ty.(*ast.Vararg); ok {
			break
		}
		// try to consume a comma, if there isnt one, we are done
		if !p.tryConsume(",") {
			break
		}
	}

	p.expect(")")
	span := SpanFrom(start, p.prevSpan())

	if len(elems) == 1 {
		return elems[0]
	}
	return ast.NewTuple(elems, span)
}

func (p *parser) parseUnionType(first ast.Type, flags Flags, spanStart Span) ast.Type {
	types := []ast.Type{first}

	for p.Token.Is("|") {
		p.advance()
		next := p.parseTypeX(flags)
		types = append(types, next)
	}

	unionSpan := SpanFrom(spanStart, p.prevSpan())
	return ast.NewUnion(types, unionSpan)
}

func (p *parser) parseVec() ast.Type {
	spanStart := p.span()
	p.advance() // skip `vec`

	p.expect("<")
	ty := p.parseType()
	p.expect(">")

	return ast.NewVec(ty, SpanFrom(spanStart, p.prevSpan()))
}
