package sema

import (
	"fmt"

	"github.com/gluax-lang/gluax/frontend/ast"
)

func (a *Analysis) resolveType(scope *Scope, ty ast.Type) Type {
	switch t := ty.(type) {
	case *ast.Path:
		found := a.resolvePathType(scope, t)
		if found.IsStruct() {
			st := found.Struct()
			_ = a.resolveStruct(scope, st, nil, t.Span())
		}
		found.SetSpan(t.Span())
		return found
	case *ast.GenericStruct:
		if a.SetStructSetupSpan(ty.Span()) {
			defer a.ClearStructSetupSpan()
		}
		ty := a.resolvePathType(scope, &t.Path)
		if !ty.IsStruct() {
			a.Panic(fmt.Sprintf("expected struct type, got: %s", ty.String()), t.Span())
		}
		st := ty.Struct()
		st = a.resolveStruct(scope, st, t.Generics, t.Span())
		return ast.NewSemType(st, t.Span())
	case *ast.Tuple:
		elems := make([]Type, 0, len(t.Elems))
		for _, elem := range t.Elems {
			ty := a.resolveType(scope, elem)
			elems = append(elems, ty)
		}
		return ast.NewSemType(ast.SemTuple{Elems: elems}, t.Span())
	case *ast.Vararg:
		ty := a.resolveType(scope, t.Type)
		if !isInnerTypeRuleCompliant(ty) {
			a.Panic(fmt.Sprintf("type `%s` is not a valid vararg type", ty.String()), ty.Span())
		}
		return ast.NewSemType(ast.NewSemVararg(ty), t.Span())
	case *ast.Function:
		fun := a.handleFunctionSignature(scope, t)
		return ast.NewSemType(fun, t.Span())
	case *ast.Unreachable:
		return ast.NewSemType(ast.SemUnreachable{}, t.Span())
	case *ast.DynTrait:
		trait := a.resolvePathTrait(a.Scope, &t.Trait)
		return ast.NewSemDynTrait(trait, t.Span())
	default:
		panic("TODO TYPE")
	}
}
