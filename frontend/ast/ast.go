package ast

import "github.com/gluax-lang/gluax/frontend/lexer"

type Ident = lexer.TokIdent

type Ast struct {
	Items       []Item
	TokenStream []lexer.Token
	Code        string
}
