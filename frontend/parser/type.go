package parser

import (
	"github.com/gluax-lang/gluax/common"
	"github.com/gluax-lang/gluax/frontend/ast"
	"github.com/gluax-lang/gluax/frontend/lexer"
)

func (p *parser) parseTypeX(flags Flags) ast.Type {
	spanStart := p.span()

	if flags.Has(FlagFuncReturnUnreachable) && p.tryConsume("unreachable") {
		return ast.NewUnreachable(spanStart)
	}

	if p.tryConsume("?") {
		if p.Token.Is("?") {
			common.PanicDiag("cannot have nested nilable types", p.span())
		}
		qSpan := p.prevSpan()
		innerType := p.parseType() // no flags, because tuple/vararg can't be nilable

		nilable := ast.NewSimplePathWithGenerics(
			lexer.NewTokIdent("nilable", qSpan),
			[]ast.Type{innerType},
		)
		return &nilable
	}

	if p.Token.Is("Self") {
		p.advance()
		selfPath := ast.NewSimplePath(lexer.NewTokIdent("Self", p.prevSpan()))
		return &selfPath
	}

	if p.Token.Is("func") {
		return p.parseFunctionType()
	}

	if flags.Has(FlagTypeTuple) && p.Token.Is("(") {
		return p.parseTupleType(flags)
	}

	if flags.Has(FlagTypeVarArg) && p.Token.Is("...") {
		p.advance()
		return ast.NewVararg(p.parseType(), SpanFrom(spanStart, p.prevSpan()))
	}

	path := p.parsePath(nil)
	return &path
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
