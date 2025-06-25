package sema

import "github.com/gluax-lang/gluax/frontend/ast"

func (a *Analysis) FindMethodsOnType(scope *Scope, ty Type, methodName string) []*ast.SemFunction {
	switch {
	case ty.IsClass():
		if methodName == "" {
			return a.FindAllClassAndTraitMethods(ty.Class(), scope)
		}
		return a.FindClassOrTraitMethod(ty.Class(), methodName, scope)
	case ty.IsGeneric():
		generic := ty.Generic()
		return a.FindGenericMethods(&generic, methodName)
	default:
		return nil
	}
}

func (a *Analysis) FindGenericMethods(generic *SemGenericType, methodName string) []*SemFunction {
	var methods []*SemFunction
	for _, trait := range generic.Traits {
		ms := a.GetTraitMethods(trait, methodName)
		methods = append(methods, ms...)
	}
	return methods
}
