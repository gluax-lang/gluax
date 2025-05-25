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
	}
	return false
}

/* Struct */

type StructInstantiation struct {
	Args []SemType
	Type *SemStruct
}

type StructsStack []StructInstantiation

type StructField struct {
	Name   lexer.TokIdent
	Type   Type
	Public bool
}

type Struct struct {
	Public   bool
	Name     lexer.TokIdent
	Generics Generics
	Fields   []StructField
	Methods  []Function
	SemStack []StructInstantiation
	span     common.Span
}

func NewStruct(name lexer.TokIdent, generics Generics, fields []StructField, methods []Function, span common.Span) *Struct {
	return &Struct{Name: name, Generics: generics, Fields: fields, Methods: methods, span: span}
}

func (si *Struct) isItem() {}

func (si *Struct) SetPublic(b bool) { si.Public = b }

func (si Struct) Span() common.Span {
	return si.span
}

func (si *Struct) AddToStack(semTy *SemStruct, concrete []SemType) {
	si.SemStack = append(si.SemStack, StructInstantiation{Args: concrete, Type: semTy})
}

func (si *Struct) GetFromStack(concrete []SemType) *SemStruct {
	for _, inst := range si.SemStack {
		if len(inst.Args) != len(concrete) {
			continue
		}
		same := true
		for i, ty := range concrete {
			if !ty.StrictMatches(inst.Args[i]) {
				same = false
				break
			}
		}
		if same {
			return inst.Type.Ref() // reuse cached *StructType
		}
	}
	return nil
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
