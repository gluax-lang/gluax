package parser

import (
	"github.com/gluax-lang/gluax/frontend/ast"
	"github.com/gluax-lang/gluax/frontend/lexer"
)

func (p *parser) parseFunctionSignature(paramFlags Flags) ast.FunctionSignature {
	spanStart := p.span()

	params := p.parseFunctionParams(paramFlags)

	errorable := p.tryConsume("!")

	var returnType ast.Type
	if p.tryConsume("->") {
		returnType = p.parseTypeX(FlagTypeTuple | FlagTypeVarArg | FlagFuncReturnUnreachable)
	} else {
		returnType = ast.NilType(SpanFrom(spanStart, p.prevSpan()))
	}

	return ast.FunctionSignature{
		Params:     params,
		Errorable:  errorable,
		ReturnType: returnType,
	}
}

func (p *parser) parseFunctionParam(flags Flags) ast.FunctionParam {
	if flags.Has(FlagFuncParamVarArg) && p.tryConsume("...") {
		if !p.Token.Is(")") {
			p.expect(")")
		}
		var ty ast.Type = ast.NewVararg(p.prevSpan())
		return ast.NewFunctionParam(nil, ty, p.prevSpan())
	}

	spanStart := p.span()

	var name *lexer.TokIdent
	if flags.Has(FlagFuncParamNamed) {
		ident := p.expectIdentMsgX("expected parameter name", FlagAllowUnderscore)
		p.expect(":")
		name = &ident
	}

	ty := p.parseType()
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
