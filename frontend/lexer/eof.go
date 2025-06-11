package lexer

import "github.com/gluax-lang/gluax/common"

type TokEOF struct {
	span common.Span
}

func (t TokEOF) isToken() {}

func (t TokEOF) Span() common.Span {
	return t.span
}

func (t TokEOF) String() string {
	return "<EOF>"
}

func (t TokEOF) Is(_ string) bool {
	return false
}

func (t TokEOF) AsString() string {
	return ""
}

func IsEOF(t Token) bool {
	_, ok := t.(TokEOF)
	return ok
}

func IsIdent(t Token) bool {
	_, ok := t.(TokIdent)
	return ok
}
