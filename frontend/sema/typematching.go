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
	case ast.SemUnionKind:
		return a.matchUnionType(t.Union(), other)
	case ast.SemVecKind:
		return a.matchVecType(t.Vec(), other)
	case ast.SemMapKind:
		return a.matchMapType(t.Map(), other)
	case ast.SemUnreachableKind:
		return other.IsUnreachable()
	case ast.SemErrorKind:
		return false
	default:
		return false
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
	case ast.SemUnionKind:
		return a.matchUnionTypeStrict(t.Union(), other)
	case ast.SemVecKind:
		return a.matchVecTypeStrict(t.Vec(), other)
	case ast.SemMapKind:
		return a.matchMapTypeStrict(t.Map(), other)
	case ast.SemUnreachableKind:
		return other.IsUnreachable()
	case ast.SemErrorKind:
		return false
	default:
		return false
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

	if ast.IsBuiltinType(s.Def.Name.Raw) && ast.IsBuiltinType(oS.Def.Name.Raw) {
		if s.Def.Name.Raw != oS.Def.Name.Raw {
			return false
		}
	} else if s.Def.Span() != oS.Def.Span() {
		return false
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

	return true
}

/* Function */

func (a *Analysis) matchFunction(f *SemFunction, other *SemFunction) bool {
	if f.Def.Errorable != other.Def.Errorable {
		return false
	}
	if len(f.Params) != len(other.Params) {
		return false
	}
	for i, p := range f.Params {
		if !a.MatchTypesStrict(p, other.Params[i]) {
			return false
		}
	}
	return a.MatchTypesStrict(f.Return, other.Return)
}

func (a *Analysis) matchFunctionType(f *SemFunction, other Type) bool {
	return other.IsFunction() && a.matchFunction(f, other.Function())
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

/* Union */

func (a *Analysis) matchUnionType(u *SemUnion, other Type) bool {
	if other.IsUnion() {
		otherU := other.Union()
		for _, oT := range otherU.Types {
			found := false
			for _, t := range u.Types {
				if a.matchTypes(t, oT) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	}
	// If other is not a union, match if any member matches
	for _, t := range u.Types {
		if a.matchTypes(t, other) {
			return true
		}
	}
	return false
}

func (a *Analysis) matchUnionTypeStrict(u *SemUnion, other Type) bool {
	if other.IsUnion() {
		otherU := other.Union()
		for _, oT := range otherU.Types {
			found := false
			for _, t := range u.Types {
				if a.MatchTypesStrict(t, oT) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	}
	// If other is not a union, match if any member matches
	for _, t := range u.Types {
		if a.MatchTypesStrict(t, other) {
			return true
		}
	}
	return false
}

/* Vec */

func (a *Analysis) matchVecType(v *SemVec, other Type) bool {
	if !other.IsVec() {
		return false
	}
	return a.matchTypes(v.Ty, other.Vec().Ty)
}

func (a *Analysis) matchVecTypeStrict(v *SemVec, other Type) bool {
	if !other.IsVec() {
		return false
	}
	return a.MatchTypesStrict(v.Ty, other.Vec().Ty)
}

/* Map */

func (a *Analysis) matchMapType(m *SemMap, other Type) bool {
	if !other.IsMap() {
		return false
	}
	otherMap := other.Map()
	return a.matchTypes(m.Key, otherMap.Key) && a.matchTypes(m.Value, otherMap.Value)
}

func (a *Analysis) matchMapTypeStrict(m *SemMap, other Type) bool {
	if !other.IsMap() {
		return false
	}
	otherMap := other.Map()
	return a.MatchTypesStrict(m.Key, otherMap.Key) && a.MatchTypesStrict(m.Value, otherMap.Value)
}
