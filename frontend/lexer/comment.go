package lexer

import "github.com/gluax-lang/gluax/common"

// TokComment represents a comment token.
type TokComment struct {
	Text      string
	span      common.Span
	Multiline bool
}

func (t TokComment) isToken() {}

func (t TokComment) Span() common.Span {
	return t.span
}

func (t TokComment) String() string {
	return t.Text
}

func (t TokComment) Is(_ string) bool {
	return false
}

func (t TokComment) AsString() string {
	return ""
}
