package sema

import "github.com/gluax-lang/gluax/frontend/ast"

func (a *Analysis) FindMethodsOnType(scope *Scope, ty Type, methodName string) []ast.SemFunction {
	switch {
	case ty.IsClass():
		return a.FindClassOrTraitMethod(ty.Class(), methodName, scope)
	case ty.IsGeneric():
		generic := ty.Generic()
		return a.FindGenericMethods(&generic, methodName)
	case ty.IsDynTrait():
		return a.GetTraitMethods(ty.DynTrait().Trait, methodName)
	default:
		return nil
	}
}

func (a *Analysis) FindGenericMethods(generic *SemGenericType, methodName string) []SemFunction {
	var methods []SemFunction
	for _, trait := range generic.Traits {
		ms := a.GetTraitMethods(trait, methodName)
		methods = append(methods, ms...)
	}
	return methods
}
