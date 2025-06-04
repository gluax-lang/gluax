package ast

import (
	"github.com/gluax-lang/gluax/frontend/common"
	"github.com/gluax-lang/gluax/frontend/lexer"
)

type Item interface {
	isItem()
	Span() common.Span
}

func SetItemPublic(item Item, b bool) {
	switch v := item.(type) {
	case *Let:
		v.Public = b
	case *Struct:
		v.Public = b
	case *Use:
		v.Public = b
	case *Import:
		v.Public = b
	case *Function:
		v.Public = b
	}
}

func SetItemAttributes(item Item, attrs Attributes) bool {
	switch v := item.(type) {
	case *Function:
		v.Attributes = attrs
		return true
	case *Let:
		v.Attributes = attrs
		return true
	case *Struct:
		v.Attributes = attrs
		return true
	case *Trait:
		v.Attributes = attrs
		return true
	}
	return false
}

/* Struct */

type StructField struct {
	Id     int // the field id, in order of declaration
	Name   lexer.TokIdent
	Type   Type
	Public bool
}

type Struct struct {
	Public      bool
	Name        lexer.TokIdent
	Generics    Generics
	Fields      []StructField
	Attributes  Attributes
	IsGlobalDef bool // true if this is a global definition
	Scope       any
	span        common.Span
}

func NewStruct(name lexer.TokIdent, generics Generics, fields []StructField, span common.Span) *Struct {
	return &Struct{Name: name, Generics: generics, Fields: fields, span: span}
}

func (si *Struct) isItem() {}

func (si *Struct) SetPublic(b bool) { si.Public = b }

func (si Struct) Span() common.Span {
	return si.span
}

/* Impl Struct */

type ImplStruct struct {
	Generics      Generics
	Struct        Type
	Methods       []Function
	Scope         any
	GenericsScope any
	span          common.Span
}

func NewImplStruct(generics Generics, st Type, methods []Function, span common.Span) *ImplStruct {
	return &ImplStruct{Generics: generics, Struct: st, Methods: methods, span: span}
}

func (is *ImplStruct) isItem() {}

func (is *ImplStruct) Span() common.Span {
	return is.span
}

/* Trait */

type Trait struct {
	Public     bool
	Name       lexer.TokIdent
	Methods    []Function
	Scope      any
	Attributes Attributes
	span       common.Span
}

func NewTrait(name lexer.TokIdent, methods []Function, span common.Span) *Trait {
	return &Trait{Name: name, Methods: methods, span: span}
}

func (t *Trait) isItem() {}

func (t *Trait) SetPublic(b bool) { t.Public = b }

func (t Trait) Span() common.Span {
	return t.span
}

/* Impl Trait */
type ImplTrait struct {
	Trait  Type
	Struct Type // the type this trait is implemented for
	span   common.Span
}

func NewImplTrait(trait Type, st Type, span common.Span) *ImplTrait {
	return &ImplTrait{Trait: trait, Struct: st, span: span}
}

func (it *ImplTrait) isItem() {}

func (it *ImplTrait) Span() common.Span {
	return it.span
}

/* Import */

type Import struct {
	Public   bool
	Path     lexer.TokString
	As       *lexer.TokIdent
	SafePath string
	span     common.Span
}

func NewImport(path lexer.TokString, as *lexer.TokIdent, span common.Span) *Import {
	return &Import{Path: path, As: as, span: span}
}

func (i *Import) isItem() {}

func (i *Import) Span() common.Span {
	return i.span
}

/* Use */

type Use struct {
	Public bool
	Path   Path
	As     *lexer.TokIdent
	span   common.Span
}

func NewUse(path Path, as *lexer.TokIdent, span common.Span) Use {
	return Use{Path: path, As: as, span: span}
}

func (u *Use) isItem() {}

func (u *Use) Span() common.Span {
	return u.span
}

func (u Use) NameIdent() lexer.TokIdent {
	if u.As != nil {
		return *u.As
	}
	return u.Path.Idents[len(u.Path.Idents)-1]
}
