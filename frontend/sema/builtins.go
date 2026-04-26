package sema

import (
	"github.com/gluax-lang/gluax/common"
	"github.com/gluax-lang/gluax/frontend/ast"
	"github.com/gluax-lang/gluax/frontend/lexer"
)

func (a *Analysis) getBuiltinType(name string) Type {
	ty := a.Scope.GetType(name)
	if ty == nil {
		a.panicf(common.SpanDefault(), "unknown type: %s", name)
	}
	if ty.Kind() != ast.SemClassKind {
		a.panicf(common.SpanDefault(), "expected class type, got: %s", ty.Kind())
	}
	return *ty
}

func (a *Analysis) nilType() Type    { return a.getBuiltinType("nil") }
func (a *Analysis) boolType() Type   { return a.getBuiltinType("bool") }
func (a *Analysis) numberType() Type { return a.getBuiltinType("number") }
func (a *Analysis) stringType() Type { return a.getBuiltinType("string") }
func (a *Analysis) anyType() Type    { return a.getBuiltinType("any") }

func (a *Analysis) varArgsType(elem Type, span Span) Type {
	return ast.NewSemType(SemVararg{Type: elem}, span)
}

func (a *Analysis) unionType(span Span, types ...Type) Type {
	if len(types) == 0 {
		panic("unionType called with no types")
	}
	return ast.NewSemType(&SemUnion{Types: types, Span_: span}, span)
}

func (a *Analysis) nilableType(base Type, span Span) Type {
	// if type is any or already nilable, return as is
	if base.IsAny() || base.IsNilable() {
		return base
	}
	return a.unionType(span, a.nilType(), base)
}

func (a *Analysis) tupleType(span Span, elems ...Type) Type {
	if len(elems) == 0 {
		panic("tupleType called with no types")
	}
	return ast.NewSemType(ast.SemTuple{Elems: elems}, span)
}

func (a *Analysis) functionType(name string, params []Type, returnType Type, span Span) Type {
	ident := lexer.NewTokIdent(name, span)
	funcT := &SemFunction{
		Def:    ast.Function{Name: &ident, Span_: span},
		Params: params,
		Return: returnType,
	}
	return ast.NewSemType(funcT, span)
}

func (a *Analysis) vecType(ty Type, span common.Span) Type {
	for _, v := range a.State.CreatedVecs {
		if a.typesMatch(v.Ty, ty, true) {
			return ast.NewSemType(v, span)
		}
	}
	vecT := ast.NewSemVec(ty, span)
	a.State.CreatedVecs = append(a.State.CreatedVecs, vecT)
	return ast.NewSemType(vecT, span)
}

func (a *Analysis) mapType(keyTy, valueTy Type, span common.Span) Type {
	if keyTy.IsNilable() || keyTy.IsNil() {
		a.Errorf(span, "map key type cannot be nilable/nil")
	}
	for _, m := range a.State.CreatedMaps {
		if a.typesMatch(m.Key, keyTy, true) && a.typesMatch(m.Value, valueTy, true) {
			return ast.NewSemType(m, span)
		}
	}
	mapT := ast.NewSemMap(keyTy, valueTy, span)
	a.State.CreatedMaps = append(a.State.CreatedMaps, mapT)
	return ast.NewSemType(mapT, span)
}
