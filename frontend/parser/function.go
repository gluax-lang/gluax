package parser

import (
	"github.com/gluax-lang/gluax/frontend/ast"
	"github.com/gluax-lang/gluax/frontend/common"
	"github.com/gluax-lang/gluax/frontend/lexer"
)

func (p *parser) parseFunctionSignature(paramFlags Flags) ast.FunctionSignature {
	spanStart := p.span()

	params := p.parseFunctionParams(paramFlags)

	errorable := p.tryConsume("!")

	returnType := p.parseFunctionReturnType(FlagTypeTuple|FlagTypeVarArg|FlagFuncReturnUnreachable, SpanFrom(spanStart, p.prevSpan()))

	return ast.FunctionSignature{
		Params:     params,
		Errorable:  errorable,
		ReturnType: returnType,
	}
}

func (p *parser) parseFunctionParam(flags Flags, isFirst bool) ast.FunctionParam {
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
		name = &ident
		if flags.Has(FlagFuncParamSelf) && ident.Raw == "self" {
			if isFirst {
				SelfPath := ast.NewPath([]ast.Ident{lexer.NewTokIdent("Self", ident.Span())})
				return ast.NewFunctionParam(name, &SelfPath, SpanFrom(spanStart, p.prevSpan()))
			} else {
				common.PanicDiag("`self` can only be used as the first parameter in this context", ident.Span())
			}
		}
		p.expect(":")
	}

	ty := p.parseType()
	span := SpanFrom(spanStart, p.prevSpan())
	return ast.NewFunctionParam(name, ty, span)
}

func (p *parser) parseFunctionParams(flags Flags) []ast.FunctionParam {
	p.expect("(")
	var params []ast.FunctionParam
	isFirst := true
	p.parseCommaSeparatedDelimited(")", func(p *parser) {
		params = append(params, p.parseFunctionParam(flags, isFirst))
		isFirst = false
	})
	return params
}

func (p *parser) parseFunctionReturnType(flags Flags, span Span) ast.Type {
	if p.tryConsume("->") {
		return p.parseTypeX(flags)
	}
	return ast.NilType(span)
}
