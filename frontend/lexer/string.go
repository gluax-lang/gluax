package lexer

import "github.com/gluax-lang/gluax/frontend/common"

// TokString represents a string token.
type TokString struct {
	Raw       string
	span      common.Span
	Multiline bool
}

func (t TokString) isToken() {}

func (t TokString) Span() common.Span {
	return t.span
}

func (t TokString) String() string {
	return t.Raw
}

func (t TokString) Is(_ string) bool {
	return false
}

func (t TokString) AsString() string {
	return ""
}

func NewTokString(s string, span common.Span) TokString {
	return TokString{Raw: s, span: span}
}
