package lexer

import "github.com/gluax-lang/gluax/frontend/common"

type TokIdent struct {
	Raw  string
	span common.Span
}

func (t TokIdent) isToken() {}

func (t TokIdent) Span() common.Span {
	return t.span
}

func (t TokIdent) String() string {
	return t.Raw
}

func (t TokIdent) Is(_ string) bool {
	return false
}

func (t TokIdent) AsString() string {
	return ""
}

func NewTokIdent(s string, span common.Span) TokIdent {
	return TokIdent{Raw: s, span: span}
}
