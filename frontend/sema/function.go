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

	// parameters
	var params []Type
	for _, param := range it.Params {
		ty := a.resolveType(child, param.Type)
		if param.Name != nil {
			if withBody {
				paramValue := ast.NewSemFunctionParam(param, ty)
				a.AddValue(child, param.Name.Raw, ast.NewValue(paramValue), param.Name.Span())
			}
		}
		params = append(params, ty)
	}

	returnType := a.nilType()
	if it.ReturnType != nil {
		returnType = a.resolveType(child, *it.ReturnType)
	}

	funcType := ast.SemFunction{
		Def:    *it,
		Params: params,
		Return: returnType,
	}
	child.Func = &funcType

	if returnType.IsTuple() {
		for _, elem := range returnType.Tuple().Elems {
			if elem.IsUnreachable() {
				a.panic((*it.ReturnType).Span(), "cannot have unreachable type inside a tuple return type")
			}
		}
	}

	if it.Body != nil && !it.IsGlobal() {
		if withBody {
			_ = a.handleBlock(child, it.Body)
			a.Matches(returnType, it.Body.Type(), it.Body.Span())
		}
	}

	if funcType.HasVarargReturn() && it.Errorable {
		a.panic(it.Span(), "cannot have vararg return type in erroable function")
	}

	return funcType
}
