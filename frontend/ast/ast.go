package ast

import "github.com/gluax-lang/gluax/frontend/lexer"

type Ident = lexer.TokIdent

type Ast struct {
	Imports     []*Import
	Uses        []*Use
	Funcs       []*Function
	ImplStructs []*ImplStruct
	ImplTraits  []*ImplTraitForStruct
	Lets        []*Let
	Structs     []*Struct
	Traits      []*Trait

	TokenStream []lexer.Token
	Code        string
}
