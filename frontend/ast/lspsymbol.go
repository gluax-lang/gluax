package ast

import "github.com/gluax-lang/gluax/common"

type LSPSymbol interface {
	LSPString() string
	Span() common.Span
}

type LSPRef struct {
	decl LSPSymbol
	span common.Span
}

func NewLSPRef(decl LSPSymbol, span common.Span) LSPRef {
	return LSPRef{
		decl: decl,
		span: span,
	}
}

func (v LSPRef) LSPString() string {
	if v.decl == nil {
		return "<nil>"
	}
	return v.decl.LSPString()
}

func (v LSPRef) Span() common.Span {
	return v.decl.Span()
}

func (v LSPRef) RefSpan() common.Span {
	return v.span
}

func (v LSPRef) GetDecl() LSPSymbol {
	return v.decl
}

func (v LSPRef) GetSymbol() *Symbol {
	if sym, ok := v.decl.(Symbol); ok {
		return &sym
	}
	return nil
}

type LSPString struct {
	value string
	span  common.Span
}

func NewLSPString(value string, span common.Span) LSPString {
	return LSPString{
		value: value,
		span:  span,
	}
}

func (s LSPString) LSPString() string {
	return s.value
}

func (s LSPString) Span() common.Span {
	return s.span
}
