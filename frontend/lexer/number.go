package lexer

import "github.com/gluax-lang/gluax/frontend/common"

type TokNumber struct {
	Raw  string
	span common.Span
}

func (t TokNumber) isToken() {}

func (t TokNumber) Span() common.Span {
	return t.span
}

func (t TokNumber) String() string {
	return t.Raw
}

func (t TokNumber) Is(_ string) bool {
	return false
}

func (t TokNumber) AsString() string {
	return ""
}

func newTokNumber(s string, span common.Span) TokNumber {
	return TokNumber{Raw: s, span: span}
}
