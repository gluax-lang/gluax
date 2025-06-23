package ast

import (
	"strings"

	"github.com/gluax-lang/gluax/common"
)

type SymbolKind uint8

const (
	SymValue SymbolKind = iota // vars, params, functions  (see Value.Kind below)
	SymType                    // class / alias / type-def
	SymImport
	SymTrait

	SymClassField
)

type symbolData interface {
	SymbolKind() SymbolKind
	LSPString() string
}

func (v *Value) SymbolKind() SymbolKind          { return SymValue }
func (t *SemType) SymbolKind() SymbolKind        { return SymType }
func (i *SemImport) SymbolKind() SymbolKind      { return SymImport }
func (f *SemaClassField) SymbolKind() SymbolKind { return SymClassField }
func (t *SemTrait) SymbolKind() SymbolKind       { return SymTrait }

type Symbol struct {
	Name   string
	span   common.Span
	public bool
	data   symbolData
}

func NewSymbol[T symbolData](name string, data T, span common.Span, public bool) Symbol {
	return Symbol{
		Name:   name,
		span:   span,
		data:   data,
		public: public,
	}
}

func (s Symbol) Data() symbolData {
	return s.data
}

func (s Symbol) Span() common.Span {
	return s.span
}

func (s *Symbol) SetSpan(span common.Span) {
	s.span = span
}

func (s Symbol) LSPString() string {
	if s.data == nil {
		return "<nil>"
	}
	return s.data.LSPString()
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

func (s *Symbol) IsTrait() bool {
	return s.Kind() == SymTrait
}

func (s *Symbol) Trait() *SemTrait {
	if s.Kind() != SymTrait {
		panic("not a trait")
	}
	return s.data.(*SemTrait)
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

func (t SemImport) String() string {
	return t.Path
}

func (i SemImport) LSPString() string {
	var sb strings.Builder
	sb.WriteString("import ")
	sb.WriteString(i.Def.As.Raw)
	sb.WriteString(" (\"")
	sb.WriteString(i.Def.Path.Raw)
	sb.WriteString("\")")
	return sb.String()
}

func (i SemImport) Span() common.Span {
	return i.Def.span
}

type SemTrait struct {
	Def         *Trait
	SuperTraits []*SemTrait // traits that this trait extends
	Methods     map[string]SemFunction
	Scope       any
}

func NewSemTrait(def *Trait) SemTrait {
	methodMap := make(map[string]SemFunction)
	return SemTrait{
		Def:     def,
		Methods: methodMap,
	}
}

func (t SemTrait) String() string {
	return t.Def.Name.Raw
}

func (t SemTrait) LSPString() string {
	var sb strings.Builder
	sb.WriteString("trait ")
	sb.WriteString(t.Def.Name.Raw)
	// if len(t.Methods) > 0 {
	// 	sb.WriteString(" {")
	// 	for name, method := range t.Methods {
	// 		sb.WriteString(method.AstString())
	// 	}
	// 	sb.WriteString("}")
	// }
	return sb.String()
}

func (t SemTrait) Span() common.Span {
	return t.Def.Name.Span()
}
