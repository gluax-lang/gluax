package sema

import (
	"maps"
	"slices"

	"github.com/gluax-lang/gluax/frontend/ast"
)

func (a *Analysis) FindMethodsOnType(scope *Scope, ty Type, methodName string) []*ast.SemFunction {
	switch {
	case ty.IsClass():
		if methodName == "" {
			return slices.Collect(maps.Values(ty.Class().GetMethods(true)))
		}
		method := ty.Class().GetMethod(methodName, true)
		if method != nil {
			return []*ast.SemFunction{method}
		}
		return nil
	case ty.IsVec():
		method := a.findVecMethod(ty, methodName)
		if method != nil {
			return []*ast.SemFunction{method}
		}
		return nil
	default:
		return nil
	}
}
