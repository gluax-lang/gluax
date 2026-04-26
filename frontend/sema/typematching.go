package sema

import "github.com/gluax-lang/gluax/frontend/ast"

func (a *Analysis) typesMatch(t, other Type, strict bool) bool {
	if t.IsError() || other.IsError() {
		return false
	}
	if other.IsUnreachable() {
		return true
	}
	if !strict && t.IsAny() {
		return !other.IsTuple() && !other.IsVararg()
	}
	if strict && t.Kind() != other.Kind() {
		return false
	}

	switch t.Kind() {
	case ast.SemClassKind:
		return a.classMatch(t.Class(), other, strict)
	case ast.SemFunctionKind:
		return a.funcMatch(t.Function(), other)
	case ast.SemTupleKind:
		return a.tupleMatch(t.Tuple(), other, strict)
	case ast.SemVarargKind:
		return a.varargMatch(t.Vararg(), other, strict)
	case ast.SemUnionKind:
		return a.unionMatch(t.Union(), other, strict)
	case ast.SemVecKind:
		return a.vecMatch(t.Vec(), other, strict)
	case ast.SemMapKind:
		return a.mapMatch(t.Map(), other, strict)
	case ast.SemUnreachableKind:
		return other.IsUnreachable()
	default:
		return false
	}
}

func (a *Analysis) classMatch(s *SemClass, other Type, strict bool) bool {
	if !strict {
		if s.IsAnyFunc() && (other.IsFunction() || other.IsAnyFunc()) {
			return true
		}
		if s.IsTable() && (other.IsTable() || other.IsVec() || other.IsMap()) {
			return true
		}
	}
	if other.Kind() != ast.SemClassKind {
		return false
	}
	oS := other.Class()
	if !strict && oS.IsSubClassOf(s) {
		return true
	}
	if strict {
		return s.Def.Span() == oS.Def.Span()
	}
	if ast.IsBuiltinType(s.Def.Name.Raw) && ast.IsBuiltinType(oS.Def.Name.Raw) {
		return s.Def.Name.Raw == oS.Def.Name.Raw
	}
	return s.Def.Span() == oS.Def.Span()
}

func (a *Analysis) funcMatch(f *SemFunction, other Type) bool {
	return other.IsFunction() && a.matchFunction(f, other.Function())
}

func (a *Analysis) matchFunction(f, other *SemFunction) bool {
	if f.Def.Errorable != other.Def.Errorable {
		return false
	}
	if len(f.Params) != len(other.Params) {
		return false
	}
	for i, p := range f.Params {
		if !a.typesMatch(p, other.Params[i], true) {
			return false
		}
	}
	return a.typesMatch(f.Return, other.Return, true)
}

func (a *Analysis) tupleMatch(t SemTuple, other Type, strict bool) bool {
	if !other.IsTuple() {
		return false
	}
	oElems := other.Tuple().Elems
	if len(t.Elems) != len(oElems) {
		return false
	}
	for i, elem := range t.Elems {
		if !a.typesMatch(elem, oElems[i], strict) {
			return false
		}
	}
	return true
}

func (a *Analysis) varargMatch(v SemVararg, other Type, strict bool) bool {
	if other.IsVararg() {
		return a.typesMatch(v.Type, other.Vararg().Type, strict)
	}
	if strict {
		return false
	}
	return a.typesMatch(v.Type, other, false)
}

func (a *Analysis) unionMatch(u *SemUnion, other Type, strict bool) bool {
	if other.IsUnion() {
		otherU := other.Union()
		for _, oT := range otherU.Types {
			found := false
			for _, t := range u.Types {
				if a.typesMatch(t, oT, strict) {
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
	for _, t := range u.Types {
		if a.typesMatch(t, other, strict) {
			return true
		}
	}
	return false
}

func (a *Analysis) vecMatch(v *SemVec, other Type, strict bool) bool {
	if !other.IsVec() {
		return false
	}
	return a.typesMatch(v.Ty, other.Vec().Ty, strict)
}

func (a *Analysis) mapMatch(m *SemMap, other Type, strict bool) bool {
	if !other.IsMap() {
		return false
	}
	o := other.Map()
	return a.typesMatch(m.Key, o.Key, strict) && a.typesMatch(m.Value, o.Value, strict)
}
