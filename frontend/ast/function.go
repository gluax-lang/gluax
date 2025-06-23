package ast

import (
	"github.com/gluax-lang/gluax/common"
	"github.com/gluax-lang/gluax/frontend/lexer"
)

type FunctionSignature struct {
	Params     []FunctionParam
	Errorable  bool
	ReturnType *Type
}

type Function struct {
	Public     bool
	Name       *lexer.TokIdent // nil if anonymous
	Params     []FunctionParam
	Errorable  bool
	ReturnType *Type
	Body       *Block // nil if abstract
	Attributes Attributes
	sem        *SemFunction
	span       common.Span
	IsItem     bool
}

func NewFunction(name *lexer.TokIdent, sig FunctionSignature, body *Block, attributes Attributes, span common.Span) *Function {
	return &Function{
		Public:     false,
		Name:       name,
		Params:     sig.Params,
		Errorable:  sig.Errorable,
		ReturnType: sig.ReturnType,
		Body:       body,
		Attributes: attributes,
		span:       span,
	}
}

func (f *Function) ExprKind() ExprKind { return ExprKindFunction }
func (f *Function) isItem()            {}
func (f *Function) isType()            {}

func (f *Function) Span() common.Span {
	return f.span
}

func (f *Function) SetSem(sem *SemFunction) {
	f.sem = sem
}

func (f *Function) Sem() *SemFunction {
	return f.sem
}

func (f *Function) IsGlobal() bool {
	return f.Attributes.Has("global")
}

func (f *Function) GlobalName() string {
	if f.IsGlobal() {
		if renameTo := f.Attributes.GetString("global"); renameTo != nil {
			return *renameTo
		}
		return f.Name.Raw
	}
	panic("function is not global, cannot get global name")
}

func (f *Function) IsFirstParamSelf() bool {
	params := f.Params
	if len(params) < 1 {
		return false
	}
	firstParam := params[0]
	if firstParam.Name == nil {
		return false
	}
	return firstParam.Name.Raw == "self"
}

type FunctionParam struct {
	Name *lexer.TokIdent // nil if defining function as a type definition
	Type Type            // nil if vararg
	span common.Span
}

func NewFunctionParam(name *lexer.TokIdent, ty Type, span common.Span) FunctionParam {
	return FunctionParam{Name: name, Type: ty, span: span}
}

func (p FunctionParam) Span() common.Span {
	return p.span
}

func (p FunctionParam) String() string {
	if p.Name == nil {
		return "..."
	}
	return p.Name.Raw
}
