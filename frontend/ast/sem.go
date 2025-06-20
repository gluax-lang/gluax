package ast

import (
	"fmt"
	"maps"
	"strings"

	"github.com/gluax-lang/gluax/common"
	"github.com/gluax-lang/gluax/frontend/lexer"
)

type SemTypeKind uint8

func (k SemTypeKind) String() string {
	switch k {
	case SemClassKind:
		return "class"
	case SemFunctionKind:
		return "function"
	case SemTupleKind:
		return "tuple"
	case SemVarargKind:
		return "vararg"
	case SemGenericKind:
		return "generic"
	case SemUnreachableKind:
		return "unreachable"
	case SemErrorKind:
		return "error"
	case SemDynTraitKind:
		return "dyn_trait"
	default:
		panic("unreachable")
	}
}

const (
	_ SemTypeKind = iota
	SemClassKind
	SemFunctionKind
	SemTupleKind
	SemVarargKind
	SemGenericKind
	SemUnreachableKind
	SemErrorKind
	SemDynTraitKind
)

type semTypeData interface {
	TypeKind() SemTypeKind
	String() string
	LSPString() string
}

type SemType struct {
	data semTypeData
	span common.Span
}

func NewSemType[T semTypeData](data T, span common.Span) SemType {
	return SemType{data: data, span: span}
}

func (t SemType) Data() semTypeData {
	return t.data
}

func (t SemType) Kind() SemTypeKind {
	return t.data.TypeKind()
}

func (t SemType) IsValid() bool {
	return t.data != nil
}

func (t SemType) LSPString() string {
	if t.data == nil {
		return "<nil>"
	}
	return t.data.LSPString()
}

func (t *SemType) SetSpan(span common.Span) {
	t.span = span
}

func (t SemType) Span() common.Span {
	return t.span
}

func (t SemType) NilableInnerType() SemType {
	if !t.IsNilable() {
		panic("not a nilable type")
	}
	return t.Class().InnerType()
}

func (t *SemType) Class() *SemClass {
	if t.Kind() != SemClassKind {
		panic("not a class")
	}
	return t.data.(*SemClass)
}

func (t SemType) Function() SemFunction {
	if t.Kind() != SemFunctionKind {
		panic("not a function")
	}
	return t.data.(SemFunction)
}

func (t SemType) Tuple() SemTuple {
	if t.Kind() != SemTupleKind {
		panic("not a tuple")
	}
	return t.data.(SemTuple)
}

func (t SemType) Vararg() SemVararg {
	if t.Kind() != SemVarargKind {
		panic("not a vararg")
	}
	return t.data.(SemVararg)
}

func (t SemType) DynTrait() SemDynTrait {
	if t.Kind() != SemDynTraitKind {
		panic("not a dyn trait")
	}
	return t.data.(SemDynTrait)
}

func (t SemType) Generic() SemGenericType {
	if t.Kind() != SemGenericKind {
		panic("not a generic")
	}
	return t.data.(SemGenericType)
}

func (t SemType) Unreachable() SemUnreachable {
	if t.Kind() != SemUnreachableKind {
		panic("not an unreachable")
	}
	return t.data.(SemUnreachable)
}

func (t SemType) IsClass() bool       { return t.Kind() == SemClassKind }
func (t SemType) IsFunction() bool    { return t.Kind() == SemFunctionKind }
func (t SemType) IsUnreachable() bool { return t.Kind() == SemUnreachableKind }
func (t SemType) IsError() bool       { return t.Kind() == SemErrorKind }
func (t SemType) IsGeneric() bool     { return t.Kind() == SemGenericKind }
func (t SemType) IsTuple() bool       { return t.Kind() == SemTupleKind }
func (t SemType) IsVararg() bool      { return t.Kind() == SemVarargKind }
func (t SemType) IsDynTrait() bool    { return t.Kind() == SemDynTraitKind }

func (t SemType) asClassName() *string {
	// has to be a class
	if t.Kind() != SemClassKind {
		return nil
	}
	name := t.Class().Def.Name.Raw
	return &name
}

func (t SemType) isNamed(wanted string) bool {
	name := t.asClassName()
	return name != nil && *name == wanted
}

func (t SemType) IsNil() bool     { return t.isNamed("nil") }
func (t SemType) IsNilable() bool { return t.isNamed("nilable") }
func (t SemType) IsAny() bool     { return t.isNamed("any") }
func (t SemType) IsAnyFunc() bool { return t.isNamed("anyfunc") }
func (t SemType) IsTable() bool   { return t.isNamed("table") }
func (t SemType) IsVec() bool     { return t.isNamed("vec") }
func (t SemType) IsMap() bool     { return t.isNamed("map") }
func (t SemType) IsBool() bool    { return t.isNamed("bool") }
func (t SemType) IsNumber() bool  { return t.isNamed("number") }
func (t SemType) IsString() bool  { return t.isNamed("string") }
func (t SemType) IsLogical() bool { return t.IsBool() || t.IsNilable() }

/* ClassType */

type SemaClassField struct {
	Ty  SemType
	Def ClassField
	Idx int // Index of the field in the class, used for field access
}

func NewSemClassField(def ClassField, ty SemType, idx int) SemaClassField {
	return SemaClassField{
		Ty:  ty,
		Def: def,
		Idx: idx,
	}
}

func (f SemaClassField) IsPublic() bool {
	return f.Def.Public
}

func (f SemaClassField) LSPString() string {
	var sb strings.Builder
	if f.IsPublic() {
		sb.WriteString("pub ")
	}
	sb.WriteString(f.Def.Name.Raw)
	sb.WriteString(": ")
	sb.WriteString(f.Ty.String())
	return sb.String()
}

func (f SemaClassField) Span() common.Span {
	return f.Def.Name.Span()
}

type SemClass struct {
	Def      *Class
	Generics SemGenerics
	Super    *SemClass
	Fields   map[string]SemaClassField
	Scope    any
}

func NewSemClass(def *Class) *SemClass {
	generics := SemGenerics{}
	fields := map[string]SemaClassField{}
	return &SemClass{
		Def:      def,
		Generics: generics,
		Fields:   fields,
	}
}

func (t *SemClass) TypeKind() SemTypeKind { return SemClassKind }

func (t *SemClass) Ref() *SemClass {
	return t
}

func (t *SemClass) IsGeneric() bool {
	return len(t.Def.Generics.Params) > 0
}

func (t *SemClass) InnerType() SemType {
	return t.Generics.Params[0]
}

func (t *SemClass) InnerType2() (SemType, SemType) {
	return t.Generics.Params[0], t.Generics.Params[1]
}

func (s SemClass) String() string {
	if s.IsNilable() {
		return "?" + s.InnerType().String()
	}
	return s.Def.Name.Raw + s.Generics.String()
}

func (s SemClass) LSPString() string {
	var sb strings.Builder
	sb.WriteString("class ")
	sb.WriteString(s.Def.Name.Raw)
	sb.WriteString(s.Generics.String())
	fieldsLen := len(s.Fields)
	if fieldsLen == 0 {
		sb.WriteString(" {}")
	} else {
		sb.WriteString(" {\n")
		i := 0
		for _, field := range s.Fields {
			sb.WriteString(fmt.Sprintf("\t%s: %s", field.Def.Name.Raw, field.Ty.String()))
			if i < fieldsLen-1 {
				sb.WriteString(",\n")
			} else {
				sb.WriteString("\n")
			}
			i++
		}
		sb.WriteString("}")
	}
	return sb.String()
}

func (s SemClass) IsNilable() bool {
	return s.Def.Name.Raw == "nilable"
}

func (s SemClass) IsAnyFunc() bool {
	return s.Def.Name.Raw == "anyfunc"
}

func (s SemClass) IsTable() bool {
	return s.Def.Name.Raw == "table"
}

func (s SemClass) GetField(name string) (SemaClassField, bool) {
	if field, ok := s.Fields[name]; ok {
		return field, true
	}
	return SemaClassField{}, false
}

func (s SemClass) AllFields() map[string]SemaClassField {
	allFields := make(map[string]SemaClassField, len(s.Fields))
	maps.Copy(allFields, s.Fields)
	return allFields
}

func (s SemClass) IsSubClassOf(other *SemClass) bool {
	if s.Super == nil {
		return false
	}
	if s.Super == other {
		return true
	}
	return s.Super.IsSubClassOf(other)
}

func (c SemClass) IsFullyConcrete() bool {
	for _, g := range c.Generics.Params {
		if g.IsGeneric() {
			return false
		}
	}
	return true
}

func (c SemClass) IsGlobal() bool {
	return c.Def.IsGlobal()
}

func (c SemClass) GlobalName() string {
	return c.Def.GlobalName()
}

func (c SemClass) GetFieldIndex(name string) int {
	field, exists := c.Fields[name]
	if !exists {
		panic(fmt.Sprintf("field '%s' does not exist in class '%s'", name, c.Def.Name.Raw))
	}
	return field.Idx
}

/* FunctionType */

type SemFunction struct {
	Def    Function
	Params []SemType
	Return SemType

	Class    *SemClass
	Trait    *SemTrait // Trait this function is defined in, if any
	Scope    any       // Scope for this function, used for generics resolution and other shit
	Generics Generics
}

func (t SemFunction) TypeKind() SemTypeKind { return SemFunctionKind }

func (t SemFunction) String() string {
	def := t.Def

	var sb strings.Builder
	sb.WriteString("func")

	if def.Name != nil {
		sb.WriteString(" ")
		sb.WriteString(def.Name.Raw)
	}

	sb.WriteString("(")
	for i, ty := range t.Params {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(ty.String())
	}
	sb.WriteString(")")

	if def.Errorable {
		sb.WriteString(" !")
	}

	if !t.Return.IsNil() {
		sb.WriteString(" -> ")
		sb.WriteString(t.Return.String())
	}

	return sb.String()
}

func (t SemFunction) HasVarargParam() bool {
	if len(t.Params) == 0 {
		return false
	}
	lastParam := t.Params[len(t.Params)-1]
	return lastParam.IsVararg()
}

func (t SemFunction) HasVarargReturn() bool {
	ret := t.Return
	if ret.IsVararg() {
		return true
	}
	if ret.IsTuple() {
		for _, p := range ret.Tuple().Elems {
			if p.IsVararg() {
				return true
			}
		}
	}
	return false
}

func (t SemFunction) VarargParamType() SemType {
	if !t.HasVarargParam() {
		panic("no vararg param")
	}
	return t.Params[len(t.Params)-1].Vararg().Type
}

func (t SemFunction) VarargReturnType() SemType {
	if !t.HasVarargReturn() {
		panic("no vararg return")
	}
	if t.Return.IsVararg() {
		return t.Return.Vararg().Type
	}
	if t.Return.IsTuple() {
		for _, elem := range t.Return.Tuple().Elems {
			if elem.IsVararg() {
				return elem.Vararg().Type
			}
		}
	}
	panic("no vararg return type found")
}

func (t SemFunction) ReturnCount() int {
	if t.HasVarargReturn() {
		panic("can't get return count when vararg return is present")
	}
	if t.Return.IsTuple() {
		return len(t.Return.Tuple().Elems)
	}
	return 1
}

func (f SemFunction) LSPString() string {
	return f.String()
}

func (f SemFunction) Span() common.Span {
	return f.Def.Name.Span()
}

func (f SemFunction) FirstReturnType() SemType {
	if f.HasVarargReturn() {
		panic("can't get first return type when vararg return is present")
	}
	if f.Return.IsTuple() {
		if len(f.Return.Tuple().Elems) == 0 {
			panic("tuple return has no elements")
		}
		return f.Return.Tuple().Elems[0]
	}
	return f.Return
}

func (f SemFunction) IsGlobal() bool {
	return f.Def.IsGlobal()
}

func (f SemFunction) GlobalName() string {
	return f.Def.GlobalName()
}

func (f SemFunction) ReturnTypes() []SemType {
	if f.Return.IsTuple() {
		return f.Return.Tuple().Elems
	}
	return []SemType{f.Return}
}

func (f SemFunction) Attributes() Attributes {
	return f.Def.Attributes
}

/* Tuple */

type SemTuple struct{ Elems []SemType }

func (t SemTuple) TypeKind() SemTypeKind { return SemTupleKind }

func (t SemTuple) String() string {
	elems := make([]string, len(t.Elems))
	for i, e := range t.Elems {
		elems[i] = fmt.Sprint(e)
	}
	return "(" + strings.Join(elems, ", ") + ")"
}

func (t SemTuple) LSPString() string {
	return t.String()
}

/* Vararg */

type SemVararg struct {
	Type SemType // The type of the vararg elements
}

func (t SemVararg) TypeKind() SemTypeKind { return SemVarargKind }

func NewSemVararg(ty SemType) SemVararg {
	return SemVararg{Type: ty}
}

func (t SemVararg) String() string    { return "..." + t.Type.String() }
func (t SemVararg) LSPString() string { return "..." + t.Type.LSPString() }

/* SemUnreachable */
type SemUnreachable struct{}

func (t SemUnreachable) TypeKind() SemTypeKind { return SemUnreachableKind }

func (t SemUnreachable) String() string    { return "unreachable" }
func (t SemUnreachable) LSPString() string { return "unreachable" }

func (t SemType) String() string {
	return t.data.String()
}

/* SemError */

type SemError struct{}

func NewErrorType(span common.Span) SemType {
	return SemType{data: SemError{}, span: span}
}

func (t SemError) TypeKind() SemTypeKind { return SemErrorKind }

func (t SemError) String() string    { return "error" }
func (t SemError) LSPString() string { return t.String() }

/* SemDynTrait */

type SemDynTrait struct {
	Trait *SemTrait
}

func NewSemDynTrait(trait *SemTrait, span common.Span) SemType {
	return SemType{
		data: SemDynTrait{trait},
		span: span,
	}
}

func (t SemDynTrait) TypeKind() SemTypeKind { return SemDynTraitKind }

func (t SemDynTrait) String() string {
	return "todo"
}

func (t SemDynTrait) LSPString() string {
	return "todo"
}

/* Generics */

type SemGenericType struct {
	Ident  lexer.TokIdent // "T", "E"
	Traits []*SemTrait    // Traits that this generic type implements, e.g. "T: Eq + Ord"
	Bound  bool
}

func NewSemGenericType(ident lexer.TokIdent, traits []*SemTrait, bound bool) SemType {
	return NewSemType(SemGenericType{Ident: ident, Traits: traits, Bound: bound}, ident.Span())
}

func (t SemGenericType) TypeKind() SemTypeKind { return SemGenericKind }

func (gt SemGenericType) String() string {
	var sb strings.Builder
	sb.WriteString(gt.Ident.Raw)
	if len(gt.Traits) > 0 {
		sb.WriteString(": ")
		for i, trait := range gt.Traits {
			if i > 0 {
				sb.WriteString(" + ")
			}
			sb.WriteString(trait.Def.Name.Raw)
		}
	}
	return sb.String()
}
func (gt SemGenericType) LSPString() string { return gt.String() }

type SemGenerics struct {
	Params []SemType
}

func NewSemGenerics(params []SemType) *SemGenerics {
	return &SemGenerics{Params: params}
}

func (gs SemGenerics) String() string {
	if len(gs.Params) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteByte('<')
	for i, param := range gs.Params {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(param.String())
	}
	sb.WriteByte('>')
	return sb.String()
}

func (gs SemGenerics) Len() int {
	return len(gs.Params)
}

func (gs SemGenerics) IsEmpty() bool {
	return len(gs.Params) == 0
}

func (gs SemGenerics) BoundCount() int {
	n := 0
	for _, g := range gs.Params {
		if !g.IsGeneric() || g.Generic().Bound {
			n++
		}
	}
	return n
}

func (gs SemGenerics) UnboundCount() int {
	return len(gs.Params) - gs.BoundCount()
}
