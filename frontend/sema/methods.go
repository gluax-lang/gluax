package sema

import (
	"maps"
	"slices"

	"github.com/gluax-lang/gluax/frontend/ast"
)

func (a *Analysis) FindMethodsOnType(scope *Scope, ty Type, methodName string, span *Span) []*ast.SemFunction {
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
		newSpan := ty.Span()
		if span != nil {
			newSpan = *span
		}
		switch methodName {
		case "push":
			method := a.functionType("push", []Type{ty, ty.Vec().Ty}, a.nilType(), newSpan).Data().(*ast.SemFunction)
			method.IsVecOrMapMethod = true
			return []*ast.SemFunction{method}
		case "pop":
			method := a.functionType("pop", []Type{ty}, ty.Vec().Ty, newSpan).Data().(*ast.SemFunction)
			method.IsVecOrMapMethod = true
			return []*ast.SemFunction{method}
		case "unpack":
			method := a.functionType("unpack", []Type{ty}, a.varArgsType(ty.Vec().Ty, ty.Span()), newSpan).Data().(*ast.SemFunction)
			method.IsVecOrMapMethod = true
			return []*ast.SemFunction{method}
		}
		return nil
	default:
		return nil
	}
}
