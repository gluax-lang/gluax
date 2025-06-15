package ast

import (
	"github.com/gluax-lang/gluax/common"
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
	case *Class:
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
	case *Class:
		v.Attributes = attrs
		return true
	case *Trait:
		v.Attributes = attrs
		return true
	}
	return false
}

/* Class */

type ClassField struct {
	Id     int // the field id, in order of declaration
	Name   lexer.TokIdent
	Type   Type
	Public bool
}

type ClassInstance struct {
	Args []SemType
	Type *SemClass
}

type ClassesStack []ClassInstance

type Class struct {
	Public         bool
	Name           lexer.TokIdent
	Generics       Generics
	Super          *Type // the type this class extends, if any
	Fields         []ClassField
	Attributes     Attributes
	IsGlobalDef    bool // true if this is a global definition
	Scope          any
	CreatedClasses ClassesStack
	span           common.Span
}

func NewClass(name lexer.TokIdent, generics Generics, super *Type, fields []ClassField, span common.Span) *Class {
	return &Class{
		Name:           name,
		Generics:       generics,
		Super:          super,
		Fields:         fields,
		CreatedClasses: make(ClassesStack, 0, 4),
		span:           span,
	}
}

func (si *Class) isItem() {}

func (si *Class) SetPublic(b bool) { si.Public = b }

func (si Class) Span() common.Span {
	return si.span
}

func (s *Class) AddClass(st *SemClass, concrete []SemType) {
	s.CreatedClasses = append(s.CreatedClasses, ClassInstance{concrete, st})
}

func (s *Class) GetClassStack() ClassesStack {
	return s.CreatedClasses
}

/* Impl Class */

type ImplClass struct {
	Generics      Generics
	Class         Type
	Methods       []Function
	Scope         any
	GenericsScope any
	span          common.Span
}

func NewImplClass(generics Generics, st Type, methods []Function, span common.Span) *ImplClass {
	return &ImplClass{Generics: generics, Class: st, Methods: methods, span: span}
}

func (is *ImplClass) isItem() {}

func (is *ImplClass) Span() common.Span {
	return is.span
}

/* Trait */

type Trait struct {
	Public      bool
	Name        lexer.TokIdent
	SuperTraits []Path // traits that this trait extends
	Methods     []Function
	Scope       any
	Attributes  Attributes
	Sem         *SemTrait // semantic information, if available
	span        common.Span
}

func NewTrait(name lexer.TokIdent, superTraits []Path, methods []Function, span common.Span) *Trait {
	return &Trait{Name: name, SuperTraits: superTraits, Methods: methods, span: span}
}

func (t *Trait) isItem() {}

func (t *Trait) SetPublic(b bool) { t.Public = b }

func (t Trait) Span() common.Span {
	return t.span
}

/* Impl Trait for Class */
type ImplTraitForClass struct {
	Generics Generics
	Trait    Path
	Class    Type // the type this trait is implemented for
	Methods  []Function
	span     common.Span
}

func NewImplTraitForClass(g Generics, trait Path, st Type, Methods []Function, span common.Span) *ImplTraitForClass {
	return &ImplTraitForClass{Generics: g, Trait: trait, Class: st, Methods: Methods, span: span}
}

func (it *ImplTraitForClass) isItem() {}

func (it *ImplTraitForClass) Span() common.Span {
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
