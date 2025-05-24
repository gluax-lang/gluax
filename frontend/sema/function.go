package sema

import (
	"github.com/gluax-lang/gluax/frontend/ast"
)

func (a *Analysis) handleFunctionSignature(scope *Scope, it *ast.Function) ast.SemFunction {
	return a.handleFunctionImpl(scope, it, false)
}

func (a *Analysis) handleFunction(scope *Scope, it *ast.Function) ast.SemFunction {
	return a.handleFunctionImpl(scope, it, true)
}

func (a *Analysis) handleFunctionImpl(scope *Scope, it *ast.Function, withBody bool) ast.SemFunction {
	child := scope.Child(false)
	child.InFunc = true
	child.IsFuncErroable = it.Errorable

	// parameters
	var params []Type
	for _, param := range it.Params {
		ty := a.resolveType(child, param.Type)
		if param.Name != nil {
			paramValue := ast.NewSemFunctionParam(param, ty)
			a.AddValue(child, param.Name.Raw, ast.NewValue(paramValue), param.Name.Span())
		}
		params = append(params, ty)
	}

	// return type
	returnType := a.resolveType(child, it.ReturnType)
	child.FuncReturnType = &returnType

	if returnType.IsTuple() {
		for _, elem := range returnType.Tuple().Elems {
			if elem.IsUnreachable() {
				a.Panic("cannot have unreachable type inside a tuple return type", it.ReturnType.Span())
			}
		}
	}

	if withBody && it.Body != nil && !a.Project.processingGlobals {
		_ = a.handleBlock(child, it.Body)
		a.Matches(returnType, it.Body.Type(), it.Body.Span())
	}

	funcType := ast.SemFunction{
		Def:    *it,
		Params: params,
		Return: returnType,
	}

	if funcType.HasVarargReturn() && it.Errorable {
		a.Panic("cannot have vararg return type in erroable function", it.Span())
	}

	return funcType
}
