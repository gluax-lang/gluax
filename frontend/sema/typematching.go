package sema

import "github.com/gluax-lang/gluax/frontend/ast"

func (a *Analysis) matchTypes(t Type, other Type) bool {
	if t.IsError() || other.IsError() {
		return false
	}

	if other.IsUnreachable() {
		// Unreachable can match any type
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

	switch t.Kind() {
	case ast.SemClassKind:
		return a.matchClassType(t.Class(), other)
	case ast.SemFunctionKind:
		return a.matchFunctionType(t.Function(), other)
	case ast.SemTupleKind:
		return a.matchTupleType(t.Tuple(), other)
	case ast.SemVarargKind:
		return a.matchVarargType(t.Vararg(), other)
	case ast.SemDynTraitKind:
		return a.matchDynTraitType(t.DynTrait(), other)
	case ast.SemGenericKind:
		return a.matchGenericType(t.Generic(), other)
	case ast.SemUnreachableKind:
		return other.IsUnreachable()
	case ast.SemErrorKind:
		return false
	default:
		panic("todo")
	}
}

func (a *Analysis) MatchTypesStrict(t Type, other Type) bool {
	if t.Kind() != other.Kind() {
		return false
	}
	switch t.Kind() {
	case ast.SemClassKind:
		return a.matchClassTypeStrict(t.Class(), other)
	case ast.SemFunctionKind:
		return a.matchFunctionType(t.Function(), other)
	case ast.SemTupleKind:
		return a.matchTupleTypeStrict(t.Tuple(), other)
	case ast.SemVarargKind:
		return a.matchVarargTypeStrict(t.Vararg(), other)
	case ast.SemDynTraitKind:
		return a.matchDynTraitTypeStrict(t.DynTrait(), other)
	case ast.SemGenericKind:
		return a.matchGenericTypeStrict(t.Generic(), other)
	case ast.SemUnreachableKind:
		return other.IsUnreachable()
	case ast.SemErrorKind:
		return false
	default:
		panic("todo")
	}
}

/* Class */

func (a *Analysis) matchClassType(s *SemClass, other Type) bool {
	if s.IsAnyFunc() && (other.IsFunction() || other.IsAnyFunc()) {
		return true
	}

	if s.IsTable() && (other.IsTable() || other.IsVec() || other.IsMap()) {
		return true
	}

	if other.Kind() != ast.SemClassKind {
		return false
	}

	oS := other.Class()

	if oS.IsSubClassOf(s) {
		return true
	}

	if s.IsNilable() {
		inner := s.InnerType()
		if other.IsNil() {
			return true
		}
		if other.IsNilable() {
			otherInner := oS.InnerType()
			return a.matchTypes(inner, otherInner)
		}
		return a.matchTypes(inner, other)
	}

	if ast.IsBuiltinType(s.Def.Name.Raw) && ast.IsBuiltinType(oS.Def.Name.Raw) {
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
		if !sg.IsAny() && !a.matchTypes(sg, og) {
			return false
		}
	}

	return true
}

func (a *Analysis) matchClassTypeStrict(s *SemClass, other Type) bool {
	if other.Kind() != ast.SemClassKind {
		return false
	}

	oS := other.Class()

	if s.Def.Span() != oS.Def.Span() {
		return false
	}

	if len(s.Generics.Params) != len(oS.Generics.Params) {
		return false
	}

	for i, sg := range s.Generics.Params {
		og := oS.Generics.Params[i]
		if !a.MatchTypesStrict(sg, og) {
			return false
		}
	}

	return true
}

/* Function */

func (a *Analysis) matchFunctionType(f SemFunction, other Type) bool {
	if !other.IsFunction() {
		return false
	}
	if len(f.Params) != len(other.Function().Params) {
		return false
	}
	for i, p := range f.Params {
		if !a.MatchTypesStrict(p, other.Function().Params[i]) {
			return false
		}
	}
	return a.MatchTypesStrict(f.Return, other.Function().Return)
}

/* Tuple */

func (a *Analysis) matchTupleType(t SemTuple, other Type) bool {
	if !other.IsTuple() {
		return false
	}
	if len(t.Elems) != len(other.Tuple().Elems) {
		return false
	}
	for i, elem := range t.Elems {
		if !a.matchTypes(elem, other.Tuple().Elems[i]) {
			return false
		}
	}
	return true
}

func (a *Analysis) matchTupleTypeStrict(t SemTuple, other Type) bool {
	if !other.IsTuple() {
		return false
	}
	if len(t.Elems) != len(other.Tuple().Elems) {
		return false
	}
	for i, elem := range t.Elems {
		if !a.MatchTypesStrict(elem, other.Tuple().Elems[i]) {
			return false
		}
	}
	return true
}

/* Vararg */

func (a *Analysis) matchVarargType(v SemVararg, other Type) bool {
	if other.IsVararg() {
		return a.matchTypes(v.Type, other.Vararg().Type)
	}
	return a.matchTypes(v.Type, other)
}

func (a *Analysis) matchVarargTypeStrict(v SemVararg, other Type) bool {
	if !other.IsVararg() {
		return false
	}
	return a.MatchTypesStrict(v.Type, other.Vararg().Type)
}

/* DynTrait */

func (a *Analysis) matchDynTraitType(dt SemDynTrait, other Type) bool {
	trait := dt.Trait
	// if other.IsClass() {
	// 	st := other.Class()
	// 	return a.ClassImplementsTrait(st, trait)
	// }
	if other.IsDynTrait() {
		otherTrait := other.DynTrait().Trait
		// Check if otherTrait implements trait (not the other way around)
		// This allows coercion from more specific traits to more general ones
		// e.g. dyn Player can be used as dyn Entity if Player implements Entity
		return traitImplements(otherTrait, trait)
	}
	return false
}

func (a *Analysis) matchDynTraitTypeStrict(dt SemDynTrait, other Type) bool {
	if other.IsDynTrait() {
		otherTrait := other.DynTrait().Trait
		return dt.Trait == otherTrait
	}
	return false
}

/* Generic */

func (a *Analysis) matchGenericType(g SemGenericType, other Type) bool {
	if other.IsGeneric() {
		return a.matchGenericTypeStrict(g, other)
	}
	return false
}

func (a *Analysis) matchGenericTypeStrict(g SemGenericType, other Type) bool {
	// other is guaranteed to be a generic
	otherG := other.Generic()
	if g.Ident.Raw != otherG.Ident.Raw {
		return false
	}
	gTraits, otherGTraits := g.Traits, otherG.Traits
	if len(gTraits) != len(otherGTraits) {
		return false
	}
	for i, trait := range gTraits {
		if trait != otherGTraits[i] {
			return false
		}
	}
	return true
}
