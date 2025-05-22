package lexer

import (
	"github.com/gluax-lang/gluax/frontend/common"
)

type Token interface {
	isToken()
	Span() common.Span
	String() string
	Is(string) bool
	// AsString used for keywords and punctuations, to make it easier to switch on tokens for them
	AsString() string
}
