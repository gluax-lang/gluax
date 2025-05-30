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

func SetItemAttributes(item Item, attrs []Attribute) bool {
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
	Methods     []Function
	Attributes  []Attribute
	IsGlobalDef bool // true if this is a global definition
	span        common.Span
}

func NewStruct(name lexer.TokIdent, generics Generics, fields []StructField, methods []Function, span common.Span) *Struct {
	return &Struct{Name: name, Generics: generics, Fields: fields, Methods: methods, span: span}
}

func (si *Struct) isItem() {}

func (si *Struct) SetPublic(b bool) { si.Public = b }

func (si Struct) Span() common.Span {
	return si.span
}

/* Import */

type Import struct {
	Public bool
	Path   lexer.TokString
	As     *lexer.TokIdent
	span   common.Span
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
