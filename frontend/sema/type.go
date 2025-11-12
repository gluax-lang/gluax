package sema

import (
	"fmt"

	"github.com/gluax-lang/gluax/frontend/ast"
)

func (a *Analysis) resolveType(scope *Scope, ty ast.Type) Type {
	switch t := ty.(type) {
	case *ast.Path:
		found := a.resolvePathType(scope, t)
		if found.IsClass() {
			st := found.Class()
			_ = a.resolveClass(st)
		}
		found.SetSpan(t.Span())
		return found
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
	case *ast.Union:
		types := make([]Type, 0, len(t.Types))
		for _, ty := range t.Types {
			ty := a.resolveType(scope, ty)
			if ty.IsTuple() || ty.IsVararg() || ty.IsUnreachable() {
				a.Errorf(ty.Span(), "cannot use %s in a union type", ty.String())
				return a.nilType()
			}
			types = append(types, ty)
		}
		return ast.NewSemType(ast.NewSemUnion(t, types), t.Span())
	default:
		panic(fmt.Sprintf("TODO TYPE: %T", t))
	}
}
