package sema

import "github.com/gluax-lang/gluax/frontend/ast"

func (a *Analysis) FindMethodsOnType(scope *Scope, ty Type, methodName string) []*ast.SemFunction {
	switch {
	case ty.IsClass():
		if methodName == "" {
			return a.GetClassMethodsRecursively(ty.Class())
		}
		method := a.FindClassMethod(ty.Class(), methodName)
		if method != nil {
			return []*ast.SemFunction{method}
		}
		return nil
	default:
		return nil
	}
}
