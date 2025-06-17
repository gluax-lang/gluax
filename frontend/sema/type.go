package sema

import (
	"github.com/gluax-lang/gluax/frontend/ast"
)

func (a *Analysis) resolveType(scope *Scope, ty ast.Type) Type {
	switch t := ty.(type) {
	case *ast.Path:
		found := a.resolvePathType(scope, t)
		if found.IsClass() {
			st := found.Class()
			_ = a.resolveClass(scope, st, nil, t.Span())
		}
		found.SetSpan(t.Span())
		return found
	case *ast.GenericClass:
		if a.SetClassSetupSpan(ty.Span()) {
			defer a.ClearClassSetupSpan()
		}
		ty := a.resolvePathType(scope, &t.Path)
		if !ty.IsClass() {
			a.panicf(t.Span(), "expected class type, got: %s", ty.String())
		}
		st := ty.Class()
		st = a.resolveClass(scope, st, t.Generics, t.Span())
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
		if !isValidAsGenericTypeArgument(ty) {
			a.panicf(ty.Span(), "type `%s` is not a valid vararg type", ty.String())
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
