package ast

import (
	"strings"

	"github.com/gluax-lang/gluax/frontend/common"
)

type SymbolKind uint8

const (
	SymValue SymbolKind = iota // vars, params, functions  (see Value.Kind below)
	SymType                    // struct / alias / type-def
	SymImport
	SymTrait

	SymStructField
)

type symbolData interface {
	SymbolKind() SymbolKind
	AstString() string // for lsp
}

func (v *Value) SymbolKind() SymbolKind          { return SymValue }
func (t *SemType) SymbolKind() SymbolKind        { return SymType }
func (i *SemImport) SymbolKind() SymbolKind      { return SymImport }
func (f *SemStructField) SymbolKind() SymbolKind { return SymStructField }

type Symbol struct {
	Name   string
	Span   common.Span
	public bool
	data   symbolData
}

func NewSymbol[T symbolData](name string, data T, span common.Span, public bool) Symbol {
	return Symbol{
		Name:   name,
		Span:   span,
		data:   data,
		public: public,
	}
}

func (s *Symbol) AstString() string {
	if s.data == nil {
		return "<nil>"
	}
	return s.data.AstString()
}

func (s *Symbol) SetPublic(b bool) {
	s.public = b
}

func (s *Symbol) IsPublic() bool {
	return s.public
}

func (s *Symbol) Kind() SymbolKind {
	return s.data.SymbolKind()
}

func (s *Symbol) IsValue() bool {
	return s.Kind() == SymValue
}

func (s *Symbol) IsType() bool {
	return s.Kind() == SymType
}

func (s *Symbol) IsImport() bool {
	return s.Kind() == SymImport
}

func (s *Symbol) Value() *Value {
	if s.Kind() != SymValue {
		panic("not a value")
	}
	return s.data.(*Value)
}

func (s *Symbol) Type() *SemType {
	if s.Kind() != SymType {
		panic("not a type")
	}
	return s.data.(*SemType)
}

func (s *Symbol) Import() *SemImport {
	if s.Kind() != SymImport {
		panic("not an import")
	}
	return s.data.(*SemImport)
}

type SemImport struct {
	Path     string
	Def      Import
	Analysis any // basically sema.Analysis, but to avoid a circular dependency lol
	Scope    any // another hack, to avoid circular dependency with sema.Scope
}

func NewSemImport(def Import, path string, analysis any) SemImport {
	return SemImport{Def: def, Analysis: analysis, Path: path}
}

func (t SemImport) Matches(other SemType) bool {
	return false
}

func (t SemImport) StrictMatches(other SemType) bool {
	return false
}

func (t SemImport) String() string {
	return t.Path
}

func (i SemImport) AstString() string {
	var sb strings.Builder
	sb.WriteString("import ")
	sb.WriteString(i.Def.As.Raw)
	sb.WriteString(" (\"")
	sb.WriteString(i.Def.Path.Raw)
	sb.WriteString("\")")
	return sb.String()
}
