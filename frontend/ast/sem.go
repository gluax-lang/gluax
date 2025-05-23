package ast

import (
	"fmt"
	"strings"

	"github.com/gluax-lang/gluax/frontend/common"
	"github.com/gluax-lang/gluax/frontend/lexer"
)

type SemTypeKind uint8

func (k SemTypeKind) String() string {
	switch k {
	case SemStructKind:
		return "struct"
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
	default:
		panic("unreachable")
	}
}

const (
	_ SemTypeKind = iota
	SemStructKind
	SemFunctionKind
	SemTupleKind
	SemVarargKind
	SemGenericKind
	SemUnreachableKind
	SemErrorKind
)

type semTypeData interface {
	TypeKind() SemTypeKind
	Matches(SemType) bool
	StrictMatches(SemType) bool
	String() string
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

func (t *SemType) SetSpan(span common.Span) {
	t.span = span
}

func (t SemType) Span() common.Span {
	return t.span
}

func (t SemType) OptionInnerType() SemType {
	if !t.IsOption() {
		panic("not an option")
	}
	return t.Struct().OptionInnerType()
}

// make sure to not allow nested options to be assigned to anything at all
func checkOptionRules(ty SemType) bool {
	if !ty.IsOption() {
		return true
	}
	tyS := ty.Struct()
	inner := tyS.OptionInnerType()
	return !inner.IsOption()
}

func (t SemType) Matches(other SemType) bool {
	if t.IsError() || other.IsError() {
		return false
	}

	if !checkOptionRules(t) || !checkOptionRules(other) {
		return false
	}

	if other.IsUnreachable() {
		return true
	}

	if t.IsAny() {
		if other.IsTuple() {
			return false
		}
		if other.IsVararg() {
			return false
		}
		return true
	}

	return t.data.Matches(other)
}

func (t SemType) StrictMatches(other SemType) bool {
	if !checkOptionRules(t) || !checkOptionRules(other) {
		return false
	}
	if t.Kind() != other.Kind() {
		return false
	}
	return t.data.StrictMatches(other)
}

func (t *SemType) Struct() *SemStruct {
	if t.Kind() != SemStructKind {
		panic("not a struct")
	}
	return t.data.(*SemStruct)
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

func (t SemType) asStructName() *string {
	// has to be a struct
	if t.Kind() != SemStructKind {
		return nil
	}
	name := t.Struct().Def.Name.Raw
	return &name
}

func (t SemType) isNamed(wanted string) bool {
	name := t.asStructName()
	return name != nil && *name == wanted
}

func (t SemType) IsStruct() bool {
	return t.Kind() == SemStructKind
}

func (t SemType) IsNil() bool {
	return t.isNamed("nil")
}

func (t SemType) IsOption() bool {
	return t.isNamed("option")
}

func (t SemType) IsAny() bool {
	return t.isNamed("any")
}

func (t SemType) IsAnyFunc() bool {
	return t.isNamed("anyfunc")
}

func (t SemType) IsTable() bool {
	return t.isNamed("table")
}

func (t SemType) IsVec() bool {
	return t.isNamed("vec")
}

func (t SemType) IsMap() bool {
	return t.isNamed("map")
}

func (t SemType) IsBool() bool {
	return t.isNamed("bool")
}

func (t SemType) IsNumber() bool {
	return t.isNamed("number")
}

func (t SemType) IsString() bool {
	return t.isNamed("string")
}

func (t SemType) IsFunction() bool {
	return t.Kind() == SemFunctionKind
}

func (t SemType) IsUnreachable() bool {
	return t.Kind() == SemUnreachableKind
}

func (t SemType) IsError() bool {
	return t.Kind() == SemErrorKind
}

func (t SemType) IsGeneric() bool {
	return t.Kind() == SemGenericKind
}

func (t SemType) IsTuple() bool {
	return t.Kind() == SemTupleKind
}

func (t SemType) IsVararg() bool {
	return t.Kind() == SemVarargKind
}

func (t SemType) IsLogical() bool {
	return t.IsBool() || t.IsOption()
}

/* StructType */

type SemStruct struct {
	Def      *Struct
	Generics SemGenerics
	Fields   map[string]SemType
	Methods  map[string]SemFunction
	Scope    any
}

func NewSemStruct(def *Struct) *SemStruct {
	generics := SemGenerics{}
	fields := map[string]SemType{}
	methods := map[string]SemFunction{}
	return &SemStruct{
		Def:      def,
		Generics: generics,
		Fields:   fields,
		Methods:  methods,
	}
}

func (t *SemStruct) TypeKind() SemTypeKind { return SemStructKind }

func (t *SemStruct) Ref() *SemStruct {
	return t
}

func (t *SemStruct) IsGeneric() bool {
	return len(t.Def.Generics.Params) > 0
}

func (t *SemStruct) OptionInnerType() SemType {
	if !t.IsOption() {
		panic("not an option")
	}
	return t.Generics.Params[0]
}

func (s SemStruct) Matches(other SemType) bool {
	if s.IsAnyFunc() && (other.IsFunction() || other.IsAnyFunc()) {
		return true
	}

	if s.IsTable() && (other.IsTable() || other.IsVec() || other.IsMap()) {
		return true
	}

	if other.Kind() != SemStructKind {
		return false
	}

	oS := other.Struct()

	if s.IsOption() {
		inner := s.OptionInnerType()
		if other.IsNil() {
			return true
		}
		if other.IsOption() {
			otherInner := oS.OptionInnerType()
			return inner.StrictMatches(otherInner)
		}
		return inner.StrictMatches(other)
	}

	if IsBuiltinType(s.Def.Name.Raw) && IsBuiltinType(oS.Def.Name.Raw) {
		if s.Def.Name.Raw != oS.Def.Name.Raw {
			return false
		}
	} else if s.Def.Span() != oS.Def.Span() {
		return false
	}

	if len(s.Generics.Params) != len(oS.Generics.Params) {
		return false
	}

	for i, sg := range s.Generics.Params {
		og := oS.Generics.Params[i]
		if !sg.IsAny() && !sg.StrictMatches(og) {
			return false
		}
	}

	return true
}

func (s SemStruct) StrictMatches(other SemType) bool {
	if s.IsOption() {
		if !other.IsOption() {
			return false
		}
		otherS := other.Struct()
		return s.OptionInnerType().StrictMatches(otherS.OptionInnerType())
	}
	return s.Matches(other)
}

func (s SemStruct) String() string {
	if s.IsOption() {
		return "?" + s.OptionInnerType().String()
	}
	return s.Def.Name.Raw + s.Generics.String()
}

func (s SemStruct) GetMethod(name string) (SemFunction, bool) {
	f, ok := s.Methods[name]
	return f, ok
}

func (s SemStruct) IsOption() bool {
	return s.Def.Name.Raw == "option"
}

func (s SemStruct) IsAnyFunc() bool {
	return s.Def.Name.Raw == "anyfunc"
}

func (s SemStruct) IsTable() bool {
	return s.Def.Name.Raw == "table"
}

/* FunctionType */

type SemFunction struct {
	Def         Function
	Params      []SemType
	Return      SemType
	OwnerStruct *SemStruct
}

func (t SemFunction) TypeKind() SemTypeKind { return SemFunctionKind }

func (t SemFunction) Matches(other SemType) bool {
	if !other.IsFunction() {
		return false
	}
	if len(t.Params) != len(other.Function().Params) {
		return false
	}
	for i, p := range t.Params {
		if !p.StrictMatches(other.Function().Params[i]) {
			return false
		}
	}
	return t.Return.StrictMatches(other.Function().Return)
}

func (t SemFunction) StrictMatches(other SemType) bool {
	return t.Matches(other)
}

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

func (t SemFunction) ReturnCount() int {
	if t.HasVarargReturn() {
		panic("can't get return count when vararg return is present")
	}
	if t.Return.IsTuple() {
		return len(t.Return.Tuple().Elems)
	}
	return 1
}

/* Tuple */

type SemTuple struct{ Elems []SemType }

func (t SemTuple) TypeKind() SemTypeKind { return SemTupleKind }

func (t SemTuple) Matches(other SemType) bool {
	if !other.IsTuple() {
		return false
	}
	if len(t.Elems) != len(other.Tuple().Elems) {
		return false
	}
	for i, elem := range t.Elems {
		if !elem.Matches(other.Tuple().Elems[i]) {
			return false
		}
	}
	return true
}

func (t SemTuple) StrictMatches(other SemType) bool {
	if !other.IsTuple() {
		return false
	}
	if len(t.Elems) != len(other.Tuple().Elems) {
		return false
	}
	for i, elem := range t.Elems {
		if !elem.StrictMatches(other.Tuple().Elems[i]) {
			return false
		}
	}
	return true
}

func (t SemTuple) String() string {
	elems := make([]string, len(t.Elems))
	for i, e := range t.Elems {
		elems[i] = fmt.Sprint(e)
	}
	return "(" + strings.Join(elems, ", ") + ")"
}

/* Vararg */

type SemVararg struct{}

func (t SemVararg) TypeKind() SemTypeKind { return SemVarargKind }

func NewSemVararg() SemVararg {
	return SemVararg{}
}

func (t SemVararg) Matches(other SemType) bool {
	return other.IsVararg() || other.IsAny()
}

func (t SemVararg) StrictMatches(other SemType) bool {
	return other.IsVararg()
}

func (t SemVararg) String() string {
	return "..."
}

/* SemUnreachable */
type SemUnreachable struct{}

func (t SemUnreachable) Matches(other SemType) bool {
	// Unreachable matches any type.
	return true
}

func (t SemUnreachable) StrictMatches(other SemType) bool {
	// Unreachable matches any type.
	return true
}

func (t SemUnreachable) TypeKind() SemTypeKind { return SemUnreachableKind }

func (t SemUnreachable) String() string {
	return "unreachable"
}

func (t SemType) String() string {
	return t.data.String()
}

/* SemError */

type SemError struct{}

func (t SemError) Matches(other SemType) bool {
	return false
}

func (t SemError) StrictMatches(other SemType) bool {
	return false
}

func (t SemError) TypeKind() SemTypeKind { return SemErrorKind }

func (t SemError) String() string {
	return "error"
}

func NewErrorType(span common.Span) SemType {
	return SemType{data: SemError{}, span: span}
}

/* Generics */

type SemGenericType struct {
	Ident lexer.TokIdent // "T", "E"
	Bound bool
}

func NewSemGenericType(ident lexer.TokIdent, bound bool) SemType {
	return NewSemType(SemGenericType{Ident: ident, Bound: bound}, ident.Span())
}

func (t SemGenericType) TypeKind() SemTypeKind { return SemGenericKind }

func (t SemGenericType) Matches(other SemType) bool {
	if other.IsGeneric() {
		return t.Ident.Raw == other.Generic().Ident.Raw
	}
	return false
}

func (t SemGenericType) StrictMatches(other SemType) bool {
	// other is guaranteed to be a generic
	otherG := other.Generic()
	return t.Ident.Raw == otherG.Ident.Raw
}

func (gt SemGenericType) String() string {
	return gt.Ident.Raw
}

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
