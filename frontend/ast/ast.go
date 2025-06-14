package ast

import "github.com/gluax-lang/gluax/frontend/lexer"

type Ident = lexer.TokIdent

type Ast struct {
	Imports     []*Import
	Uses        []*Use
	Funcs       []*Function
	ImplClasses []*ImplClass
	ImplTraits  []*ImplTraitForClass
	Lets        []*Let
	Classes     []*Class
	Traits      []*Trait

	TokenStream []lexer.Token
	Code        string
}
