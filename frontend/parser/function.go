package parser

import (
	"github.com/gluax-lang/gluax/frontend/ast"
	"github.com/gluax-lang/gluax/frontend/lexer"
)

func (p *parser) parseFunctionSignature(paramFlags Flags) ast.FunctionSignature {
	spanStart := p.span()

	params := p.parseFunctionParams(paramFlags)

	errorable := p.tryConsume("!")

	returnType := p.parseFunctionReturnType(FlagTypeTuple|FlagTypeVarArg|FlagFuncReturnUnreachable|FlagTypeImplTrait, SpanFrom(spanStart, p.prevSpan()))

	return ast.FunctionSignature{
		Params:     params,
		Errorable:  errorable,
		ReturnType: returnType,
	}
}

func (p *parser) parseFunctionParam(flags Flags) ast.FunctionParam {
	if flags.Has(FlagFuncParamVarArg) && p.tryConsume("...") {
		varargSpan := p.prevSpan()
		varargTy := p.parseType()
		varargSpan = SpanFrom(varargSpan, p.prevSpan())
		if !p.Token.Is(")") {
			p.expect(")")
		}
		var ty ast.Type = ast.NewVararg(varargTy, varargSpan)
		return ast.NewFunctionParam(nil, ty, varargSpan)
	}

	spanStart := p.span()

	var name *lexer.TokIdent
	if flags.Has(FlagFuncParamNamed) {
		ident := p.expectIdentMsgX("expected parameter name", FlagAllowUnderscore)
		p.expect(":")
		name = &ident
	}

	ty := p.parseTypeX(FlagTypeImplTrait)
	span := SpanFrom(spanStart, p.prevSpan())
	return ast.NewFunctionParam(name, ty, span)
}

func (p *parser) parseFunctionParams(flags Flags) []ast.FunctionParam {
	p.expect("(")
	var params []ast.FunctionParam
	p.parseCommaSeparatedDelimited(")", func(p *parser) {
		params = append(params, p.parseFunctionParam(flags))
	})
	return params
}

func (p *parser) parseFunctionReturnType(flags Flags, span Span) ast.Type {
	if p.tryConsume("->") {
		return p.parseTypeX(flags)
	}
	return ast.NilType(span)
}
