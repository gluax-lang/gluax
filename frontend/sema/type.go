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
			if !t.IsSelf() {
				if !st.Def.Generics.IsEmpty() {
					// If *all* generics are still unbound, user wrote something
					// like `Inner` instead of `Inner<T>`.
					if st.Generics.UnboundCount() == st.Generics.Len() {
						a.Panic(fmt.Sprintf(
							"struct `%s` is generic but no generic arguments were provided",
							st.Def.Name.Raw,
						), t.Span())
					}
				}
			}
		}
		found.SetSpan(t.Span())
		return found
	case *ast.GenericStruct:
		ty := a.resolvePathType(scope, &t.Path)
		if !ty.IsStruct() {
			a.Panic(fmt.Sprintf("expected struct type, got: %s", ty.String()), t.Span())
		}
		st := ty.Struct()
		if st.Def.Generics.IsEmpty() {
			a.Panic(fmt.Sprintf("struct `%s` is not generic but generics were provided", st.Def.Name.Raw), t.Span())
		}
		if len(t.Generics) != st.Def.Generics.Len() {
			a.Panic(fmt.Sprintf("expected %d generics, got %d", st.Def.Generics.Len(), len(t.Generics)), t.Span())
		}
		concrete := make([]Type, 0, len(t.Generics))
		for _, g := range t.Generics {
			concrete = append(concrete, a.resolveType(scope, g))
		}
		conSt := a.instantiateStruct(st.Def, concrete, false)
		return ast.NewSemType(conSt, t.Span())
	// case *ast.Optional:
	// 	resolved := a.resolveType(scope, t.Type)
	// 	if resolved.IsAny() || resolved.IsNil() || resolved.IsGeneric() {
	// 		a.Panic("optional type cannot be any, nil, or generic", t.Span())
	// 	}
	// 	return ast.NewSemType(ast.NewSemOptional(resolved), t.Span())
	case *ast.Tuple:
		elems := make([]Type, 0, len(t.Elems))
		for _, elem := range t.Elems {
			ty := a.resolveType(scope, elem)
			elems = append(elems, ty)
		}
		return ast.NewSemType(ast.SemTuple{Elems: elems}, t.Span())
	case *ast.Vararg:
		return ast.NewSemType(ast.NewSemVararg(), t.Span())
	case *ast.Function:
		fun := a.handleFunctionSignature(scope, t)
		return ast.NewSemType(fun, t.Span())
	case *ast.Unreachable:
		return ast.NewSemType(ast.SemUnreachable{}, t.Span())
	default:
		panic("TODO TYPE")
	}
}
