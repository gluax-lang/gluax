package ast

import "github.com/gluax-lang/gluax/frontend/lexer"

type Ident = lexer.TokIdent

type Ast struct {
	Imports     []*Import
	Uses        []*Use
	Funcs       []*Function
	ImplClasses []*ImplClass
	Lets        []*Let
	Classes     []*Class

	TokenStream []lexer.Token
	Code        string
}
