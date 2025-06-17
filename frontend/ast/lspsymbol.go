package ast

import "github.com/gluax-lang/gluax/common"

type LSPSymbol interface {
	LSPString() string
	Span() common.Span
}
