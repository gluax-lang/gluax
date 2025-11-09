package sema

import "github.com/gluax-lang/gluax/frontend/ast"

func (a *Analysis) FindMethodsOnType(scope *Scope, ty Type, methodName string) []*ast.SemFunction {
	switch {
	case ty.IsClass():
		if methodName == "" {
			return a.FindAllClassAndTraitMethods(ty.Class(), scope)
		}
		return a.FindClassOrTraitMethod(ty.Class(), methodName, scope)
	default:
		return nil
	}
}
